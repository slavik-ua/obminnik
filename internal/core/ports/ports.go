package ports

import (
	"context"
	"encoding/json"
)

type OutboxNotifier interface {
	Notify()
}

type BroadcastEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Broadcaster interface {
	Broadcast(ctx context.Context, event BroadcastEvent) error
}
