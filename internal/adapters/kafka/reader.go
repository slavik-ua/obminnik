package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Reader struct {
	reader *kafka.Reader
}

func NewKafkaReader(brokerAddr, topic, groupID string) *Reader {
	return &Reader{
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

func (r *Reader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return r.reader.FetchMessage(ctx)
}

//nolint:gocritic // msg is passed by value because kafka-go requires it
func (r *Reader) CommitMessage(ctx context.Context, msg kafka.Message) error {
	return r.reader.CommitMessages(ctx, msg)
}

func (r *Reader) Close() error {
	return r.reader.Close()
}
