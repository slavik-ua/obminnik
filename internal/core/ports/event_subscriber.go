package ports

import (
	"context"
	"github.com/segmentio/kafka-go"
)

type KafkaReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessage(ctx context.Context, msg kafka.Message) error
	Close() error
}
