package postgres

import (
	"context"
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type wagerTransactionRepository struct {
	pool *pgxpool.Pool
}

func NewWagerTransactionRepository(pool *pgxpool.Pool) app.WagerTransactionRepository {
	return &wagerTransactionRepository{pool: pool}
}

func (r *wagerTransactionRepository) Save(ctx context.Context, wagerTransaction *domain.WagerTransaction) error {
	exec := executor(ctx, r.pool)

	_, err := exec.Exec(ctx, `
			INSERT INTO wager_transactions (
			id, kind, status, wallet_id, player_id,
			amount_minor_units, currency,
			provider_id, external_transaction_id, idempotency_key, payload_hash,
			round_id, game_id, reference_external_transaction_id,
			resolved_reference_id, failure_code,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`,
		wagerTransaction.ID(), wagerTransaction.Kind(), wagerTransaction.Status(), wagerTransaction.WalletID(), wagerTransaction.PlayerID(),
		wagerTransaction.Money().MinorUnits(), wagerTransaction.Money().Currency(),
		NullIfEmpty(string(wagerTransaction.ProviderID())), NullIfEmpty(string(wagerTransaction.ExternalTransactionID())), NullIfEmpty(wagerTransaction.IdempotencyKey()), NullIfEmpty(wagerTransaction.PayloadHash()),
		NullIfEmpty(string(wagerTransaction.RoundID())), NullIfEmpty(string(wagerTransaction.GameID())), NullIfEmpty(wagerTransaction.ReferenceExternalTransactionID()),
		NullIfEmpty(string(wagerTransaction.ResolvedReferenceID())), NullIfEmpty(wagerTransaction.FailureCode()),
		wagerTransaction.CreatedAt(), wagerTransaction.UpdatedAt(),
	)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("postgres: error on creating wager transaction: %w", err)
	}

	return nil
}
