package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalidInboxEntry = errors.New("inbox: invalid state entry")

type InboxEntry struct {
	consumerName string
	messageID    string
	payloadHash  string
	receivedAt   time.Time
	completedAt  *time.Time
}

func NewInboxEntry(consumerName string, messageID string, payloadHash string) (*InboxEntry, error) {

	if consumerName == "" || messageID == "" || payloadHash == "" {
		return nil, fmt.Errorf("%w: missing required fields consumerName=%s, messageID=%s, payloadHash=%s", ErrInvalidInboxEntry, consumerName, messageID, payloadHash)
	}

	return &InboxEntry{
		consumerName: consumerName,
		messageID:    messageID,
		payloadHash:  payloadHash,
		receivedAt:   time.Now().UTC(),
	}, nil
}

func RehydrateInboxEntry(
	consumerName string,
	messageID string,
	payloadHash string,
	receivedAt time.Time,
	completedAt *time.Time,
) (*InboxEntry, error) {

	entry, err := NewInboxEntry(consumerName, messageID, payloadHash)
	if err != nil {
		return nil, err
	}

	entry.receivedAt = receivedAt
	entry.completedAt = completedAt

	return entry, nil
}

func (e *InboxEntry) ConsumerName() string  { return e.consumerName }
func (e *InboxEntry) MessageID() string     { return e.messageID }
func (e *InboxEntry) PayloadHash() string   { return e.payloadHash }
func (e *InboxEntry) ReceivedAt() time.Time { return e.receivedAt }
func (e *InboxEntry) IsCompleted() bool     { return e.completedAt != nil }

func (e *InboxEntry) MatchesPayload(payloadHash string) bool {
	return e.payloadHash == payloadHash
}

func (e *InboxEntry) MarkCompleted() error {
	if e.IsCompleted() {
		return fmt.Errorf("%w: already completed", ErrInvalidInboxEntry)
	}
	now := time.Now().UTC()
	e.completedAt = &now
	return nil
}
