package app

import "github.com/google/uuid"

type IDGenerator interface {
	NewID() uuid.UUID
}
