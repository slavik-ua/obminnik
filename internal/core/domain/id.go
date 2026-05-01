package domain

import "github.com/google/uuid"

type IDGenerator interface {
	Next() uuid.UUID
}
