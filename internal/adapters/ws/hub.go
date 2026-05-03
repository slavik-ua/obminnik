package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"simple-orderbook/internal/core/ports"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait) * 9 / 10
	maxMessageSize = 65536
	clientBufSize  = 256
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients        map[*Client]bool
	mu             sync.RWMutex
	redis          *redis.Client
	register       chan *Client
	unregister     chan *Client
	tradeChan      chan []byte
	obMu           sync.Mutex
	latestSnapshot []byte
}

func NewHub(redis *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		redis:      redis,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		tradeChan:  make(chan []byte, 2048),
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("error", "err", err)
			}
			break
		}
	}
}

func (h *Hub) Broadcast(ctx context.Context, event ports.BroadcastEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return h.redis.Publish(ctx, "market_updates", data).Err()
}

func (h *Hub) broadcastToAll(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			go func(c *Client) { h.unregister <- c }(client)
		}
	}
}

func (h *Hub) Run(ctx context.Context) {
	pubsub := h.redis.Subscribe(ctx, "market_updates")
	defer pubsub.Close()

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var event ports.BroadcastEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				slog.Error("hub: unmarshal failed", "error", err)
				continue
			}

			if event.Type == "TRADES_EXECUTED" {
				select {
				case h.tradeChan <- []byte(msg.Payload):
				default:
					slog.Warn("hub: trade channel full")
				}
			} else if event.Type == "ORDERBOOK_UPDATE" {
				h.obMu.Lock()
				h.latestSnapshot = []byte(msg.Payload)
				h.obMu.Unlock()
			}
		}
	}()

	obTicker := time.NewTicker(100 * time.Millisecond)
	defer obTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.DisconnectClient(client)

		case tradeData := <-h.tradeChan:
			h.broadcastToAll(tradeData)

		case <-obTicker.C:
			h.obMu.Lock()
			snap := h.latestSnapshot
			h.latestSnapshot = nil
			h.obMu.Unlock()

			if snap != nil {
				h.broadcastToAll(snap)
			}
		}
	}
}

func (h *Hub) DisconnectClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
		c.conn.Close()
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("update failed", "err", err)
		return
	}
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, clientBufSize),
	}

	h.register <- client
	go client.writePump()
	go client.readPump()
}
