package postgres

import (
	"context"
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) app.OutboxRepository {
	return &outboxRepository{pool: pool}
}

func (r *outboxRepository) Save(ctx context.Context, outboxEntry *domain.OutboxEntry) error {
	exec := executor(ctx, r.pool)

	_, err := exec.Exec(ctx, `
			INSERT INTO outbox_entries (
			event_id, aggregate_id, event_type, payload,
			occurred_at, attempts, next_attempt_at, published_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`,
		outboxEntry.EventID(),
		outboxEntry.AggregateID(),
		outboxEntry.EventType(),
		string(outboxEntry.Payload()),
		outboxEntry.OccurredAt(),
		outboxEntry.Attempts(),
		outboxEntry.NextAttemptAt(),
		outboxEntry.PublishedAt(),
	)

	if err != nil {
		return fmt.Errorf("postgres: failure on save outbox entry on database: %w", err)
	}
	return nil
}
