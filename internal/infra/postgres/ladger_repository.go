package postgres

import (
	"context"
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ledgerRepository struct {
	pool *pgxpool.Pool
}

func NewLedgerRepository(pool *pgxpool.Pool) app.LedgerRepository {
	return &ledgerRepository{pool: pool}
}

func (r *ledgerRepository) Append(ctx context.Context, ledgerEntry *domain.WalletLedgerEntry) error {
	exec := executor(ctx, r.pool)

	_, err := exec.Exec(ctx, `
			INSERT INTO wallet_ledger_entries (
			id, wallet_id, transaction_id, direction,
			amount_minor_units, currency,
			balance_before_minor_units, balance_after_minor_units, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`,
		ledgerEntry.ID(),
		ledgerEntry.WalletID(),
		ledgerEntry.TransactionID(),
		ledgerEntry.Direction(),
		ledgerEntry.Amount().MinorUnits(),
		ledgerEntry.Amount().Currency(),
		ledgerEntry.BalanceBefore().MinorUnits(),
		ledgerEntry.BalanceAfter().MinorUnits(),
		ledgerEntry.CreatedAt(),
	)

	if err != nil {
		return fmt.Errorf("postgres: error on appending ledger entry: %w", err)
	}

	return nil
}
