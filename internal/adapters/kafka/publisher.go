package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	_ "github.com/segmentio/kafka-go/snappy"
	"simple-orderbook/internal/core/domain"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokerAddr string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokerAddr),
			Balancer: &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll, // Ensure all replicas acknowledge
			MaxAttempts: 5,
			WriteTimeout: 10 * time.Second,
			ReadTimeout: 10 * time.Second,
			Compression: kafka.Snappy,
		},
	}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event *domain.OutboxEvent) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: event.Type,
		Key:   []byte(event.ID.String()),
		Value: event.Payload,
		Time: event.CreatedAt,
	})
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}