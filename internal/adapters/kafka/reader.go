package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaReader struct {
	reader *kafka.Reader
}

func NewKafkaReader(brokerAddr, topic, groupID string) *KafkaReader {
	return &KafkaReader{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        []string{brokerAddr},
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			MaxWait:        5 * time.Millisecond,
			StartOffset:    kafka.FirstOffset,
			CommitInterval: 1 * time.Millisecond,
		}),
	}
}

func (r *KafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return r.reader.FetchMessage(ctx)
}

func (r *KafkaReader) CommitMessage(ctx context.Context, msg kafka.Message) error {
	return r.reader.CommitMessages(ctx, msg)
}

func (r *KafkaReader) Close() error {
	return r.reader.Close()
}
