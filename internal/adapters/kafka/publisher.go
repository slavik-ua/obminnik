package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	_ "github.com/segmentio/kafka-go/snappy"
	"simple-orderbook/internal/core/domain"
)

type KafkaWriter struct {
	writer *kafka.Writer
}

func NewKafkaWriter(brokerAddr string, topic string) *KafkaWriter {
	return &KafkaWriter{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokerAddr),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			Async:        true,
			BatchTimeout: 5 * time.Millisecond,
			RequiredAcks: kafka.RequireAll, // Ensure all replicas acknowledge
			MaxAttempts:  5,
			WriteTimeout: 10 * time.Second,
			ReadTimeout:  10 * time.Second,
			Compression:  kafka.Snappy,
		},
	}
}

func (p *KafkaWriter) Publish(ctx context.Context, event *domain.OutboxEvent) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ID.String()),
		Value: event.Payload,
		Time:  event.CreatedAt,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
		},
	})
}

func (p *KafkaWriter) Close() error {
	return p.writer.Close()
}
