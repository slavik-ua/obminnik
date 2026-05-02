package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	_ "github.com/segmentio/kafka-go/snappy"
	"simple-orderbook/internal/core/domain"
)

type Writer struct {
	writer *kafka.Writer
}

func NewKafkaWriter(brokerAddr, topic string) *Writer {
	return &Writer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokerAddr),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			Async:        true,
			BatchTimeout: 1 * time.Millisecond,
			BatchSize:    100,
			RequiredAcks: kafka.RequireOne,
			MaxAttempts:  5,
			WriteTimeout: 10 * time.Second,
			ReadTimeout:  10 * time.Second,
			Compression:  kafka.Snappy,
		},
	}
}

func (p *Writer) Publish(ctx context.Context, event *domain.OutboxEvent) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("market-1"),
		Value: event.Payload,
		Time:  event.CreatedAt,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
		},
	})
}

func (p *Writer) Close() error {
	return p.writer.Close()
}
