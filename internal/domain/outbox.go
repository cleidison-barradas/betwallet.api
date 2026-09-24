package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalidOutboxEntry = errors.New("outbox: invalid state entry")

type OutboxEventID string

type OutboxEntry struct {
	eventID       OutboxEventID
	aggregateID   string
	eventType     string
	payload       []byte
	occurredAt    time.Time
	attempts      int
	nextAttemptAt time.Time
	publishedAt   *time.Time
}

func NewOutboxEntry(
	eventID OutboxEventID,
	aggregateID string,
	eventType string,
	payload []byte,
) (*OutboxEntry, error) {

	if aggregateID == "" || eventType == "" || len(payload) == 0 {
		return nil, fmt.Errorf("%w: missing required fields aggregateID=%s, eventType=%s, payload=%v", ErrInvalidOutboxEntry, aggregateID, eventType, payload)
	}

	now := time.Now().UTC()

	entry := &OutboxEntry{
		eventID:       eventID,
		aggregateID:   aggregateID,
		eventType:     eventType,
		payload:       payload,
		occurredAt:    now,
		nextAttemptAt: now,
	}
	return entry, nil
}

func RehydrateOutboxEntry(
	eventID OutboxEventID,
	aggregateID string,
	eventType string,
	payload []byte,
	occurredAt time.Time,
	attempts int,
	nextAttemptAt time.Time,
	publishedAt *time.Time,
) (*OutboxEntry, error) {
	entry, err := NewOutboxEntry(eventID, aggregateID, eventType, payload)
	if err != nil {
		return nil, err
	}
	entry.occurredAt = occurredAt
	entry.attempts = attempts
	entry.nextAttemptAt = nextAttemptAt
	entry.publishedAt = publishedAt
	return entry, nil
}

func (e *OutboxEntry) EventID() OutboxEventID   { return e.eventID }
func (e *OutboxEntry) AggregateID() string      { return e.aggregateID }
func (e *OutboxEntry) EventType() string        { return e.eventType }
func (e *OutboxEntry) Payload() []byte          { return e.payload }
func (e *OutboxEntry) OccurredAt() time.Time    { return e.occurredAt }
func (e *OutboxEntry) Attempts() int            { return e.attempts }
func (e *OutboxEntry) NextAttemptAt() time.Time { return e.nextAttemptAt }
func (e *OutboxEntry) IsPublished() bool        { return e.publishedAt != nil }
func (e *OutboxEntry) PublishedAt() *time.Time  { return e.publishedAt }

func (e *OutboxEntry) IsDue(now time.Time) bool {
	return !e.IsPublished() && !now.Before(e.nextAttemptAt)
}

func (e *OutboxEntry) MarkPublished() error {
	if e.IsPublished() {
		return fmt.Errorf("%w: event has published", ErrInvalidOutboxEntry)
	}
	now := time.Now().UTC()
	e.publishedAt = &now
	return nil
}

func (e *OutboxEntry) ScheduleRetry(backoff time.Duration) error {
	if e.IsPublished() {
		return fmt.Errorf("%w: event has published", ErrInvalidOutboxEntry)
	}

	e.attempts++
	e.nextAttemptAt = time.Now().UTC().Add(backoff)
	return nil
}
