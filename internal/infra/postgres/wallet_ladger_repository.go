package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type walletLedgerRepository struct {
	pool *pgxpool.Pool
}

func NewLedgerRepository(pool *pgxpool.Pool) app.WalletLedgerRepository {
	return &walletLedgerRepository{pool: pool}
}

func (r *walletLedgerRepository) Append(ctx context.Context, ledgerEntry *domain.WalletLedgerEntry) error {
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

func (r *walletLedgerRepository) ListWalletLedger(ctx context.Context, params app.ListWalletLedgerParams) (*app.WalletLedgerPagination, error) {
	exec := executor(ctx, r.pool)

	query := `
			SELECT
			id,
			wallet_id,
			transaction_id,
			direction,
			amount_minor_units,
			currency,
			balance_before_minor_units,
			balance_after_minor_units,
			created_at
		FROM wallet_ledger_entries
		WHERE wallet_id = $1
		  AND (
				$2::timestamptz IS NULL
				OR (created_at, id) < ($2, $3)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`

	var cursorCreatedAt *time.Time
	var cursorID *string

	if params.Cursor != nil {
		cursorCreatedAt = &params.Cursor.CreatedAt
		cursorID = &params.Cursor.ID
	}

	rows, err := exec.Query(ctx, query, params.WalletID, cursorCreatedAt, cursorID, params.Limit+1)

	if err != nil {
		return nil, fmt.Errorf("postgres: error on listing ledger entries: %w", err)
	}

	defer rows.Close()

	var ledgerEntries []domain.WalletLedgerEntry

	for rows.Next() {
		var (
			id, walletID, transactionID string
			direction                   string
			amountMinorUnits            int64
			currency                    string
			balanceBeforeMinorUnits     int64
			balanceAfterMinorUnits      int64
			createdAt                   time.Time
		)

		if err := rows.Scan(
			&id,
			&walletID,
			&transactionID,
			&direction,
			&amountMinorUnits,
			&currency,
			&balanceBeforeMinorUnits,
			&balanceAfterMinorUnits,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("postgres: error on scanning ledger entry: %w", err)
		}

		amount, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
		if err != nil {
			return nil, fmt.Errorf("postgres: failure on get amount from database: %w", err)
		}

		balanceBefore, err := domain.FromMinorUnits(balanceBeforeMinorUnits, domain.Currency(currency))
		if err != nil {
			return nil, fmt.Errorf("postgres: failure on get balance before from database: %w", err)
		}

		balanceAfter, err := domain.FromMinorUnits(balanceAfterMinorUnits, domain.Currency(currency))
		if err != nil {
			return nil, fmt.Errorf("postgres: failure on get balance after from database: %w", err)
		}

		ledger, err := domain.RehydrateWalletLedgerEntry(domain.RehydrateWalletLedgerEntryParams{
			ID:            id,
			WalletID:      walletID,
			TransactionID: transactionID,
			Direction:     domain.Direction(direction),
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			CreatedAt:     createdAt,
		})

		if err != nil {
			return nil, fmt.Errorf("postgres: failure on rehydrate ledger entry: %w", err)
		}

		ledgerEntries = append(ledgerEntries, *ledger)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error on scanning ledger entry: %w", err)
	}

	hasNext := len(ledgerEntries) > params.Limit

	if hasNext {
		ledgerEntries = ledgerEntries[:params.Limit]
	}

	var nextCursor *app.WalletLedgerCursor

	if hasNext {
		lastEntry := ledgerEntries[len(ledgerEntries)-1]

		nextCursor = &app.WalletLedgerCursor{
			CreatedAt: lastEntry.CreatedAt(),
			ID:        lastEntry.ID(),
		}
	}

	return &app.WalletLedgerPagination{
		Entries: ledgerEntries,
		Cursor:  nextCursor,
		HasNext: hasNext,
	}, nil
}

func (r *walletLedgerRepository) FindByTransactionID(ctx context.Context, transactionID string) (*domain.WalletLedgerEntry, error) {
	exec := executor(ctx, r.pool)

	query := `
			SELECT
			id,
			wallet_id,
			transaction_id,
			direction,
			amount_minor_units,
			currency,
			balance_before_minor_units,
			balance_after_minor_units,
			created_at
		FROM wallet_ledger_entries
		WHERE transaction_id = $1
	`

	row := exec.QueryRow(ctx, query, transactionID)

	var (
		id, walletID, transactionId string
		direction                   string
		amountMinorUnits            int64
		currency                    string
		balanceBeforeMinorUnits     int64
		balanceAfterMinorUnits      int64
		createdAt                   time.Time
	)

	if err := row.Scan(
		&id,
		&walletID,
		&transactionId,
		&direction,
		&amountMinorUnits,
		&currency,
		&balanceBeforeMinorUnits,
		&balanceAfterMinorUnits,
		&createdAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app.ErrLedgerNotFound
		}
	}

	amount, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get amount from database: %w", err)
	}

	balanceBefore, err := domain.FromMinorUnits(balanceBeforeMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance before from database: %w", err)
	}

	balanceAfter, err := domain.FromMinorUnits(balanceAfterMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance after from database: %w", err)
	}

	return domain.RehydrateWalletLedgerEntry(domain.RehydrateWalletLedgerEntryParams{
		ID:            id,
		WalletID:      walletID,
		TransactionID: transactionID,
		Direction:     domain.Direction(direction),
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		CreatedAt:     createdAt,
	})
}
