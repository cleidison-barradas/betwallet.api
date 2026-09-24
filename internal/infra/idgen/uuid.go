package idgen

import (
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/google/uuid"
)

type UUIDGenerator struct{}

func New() app.IDGenerator {
	return &UUIDGenerator{}
}

func (UUIDGenerator) NewID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("idgen: failure on generate uuid v7: %v", err))
	}

	return id
}
