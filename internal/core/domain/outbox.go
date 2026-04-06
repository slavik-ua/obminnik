package domain

import (
	"github.com/google/uuid"
	"time"
)

type OutboxEvent struct {
	ID          uuid.UUID
	Type        string
	Payload     []byte
	CreatedAt   time.Time
	ProcessedAt time.Time
}
