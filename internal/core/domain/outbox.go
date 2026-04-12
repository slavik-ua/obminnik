package domain

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID          uuid.UUID
	Type        string
	Payload     []byte
	CreatedAt   time.Time
	ProcessedAt time.Time
}
