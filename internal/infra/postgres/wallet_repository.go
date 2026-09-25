package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type walletRepository struct {
	pool *pgxpool.Pool
}

func NewWalletRepository(pool *pgxpool.Pool) app.WalletRepository {
	return &walletRepository{pool: pool}
}

func (r *walletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	exec := executor(ctx, r.pool)

	_, err := exec.Exec(ctx, `
		INSERT INTO wallets (id, player_id, currency, balance_minor_units, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, wallet.ID(), wallet.PlayerID(), wallet.Currency(), wallet.Balance().MinorUnits(), wallet.Version(), wallet.CreatedAt(), wallet.UpdatedAt(),
	)
	if err != nil {
		if isUniqueViolation(err, "wallets_player_id_currency_key") {
			return app.ErrWalletAlreadyExists
		}
		return fmt.Errorf("postgres: error on creating wallet: %w", err)
	}
	return nil
}

func (r *walletRepository) FindByID(ctx context.Context, walletID domain.WalletID) (*domain.Wallet, error) {
	exec := executor(ctx, r.pool)

	row := exec.QueryRow(ctx, `
		SELECT id, player_id, currency, balance_minor_units, version, created_at, updated_at
		FROM wallets
		WHERE id = $1
	`, walletID)

	var (
		walletId, playerId, currency string
		balanceMinorUnits, version   int64
		createdAt, updatedAt         time.Time
	)

	if err := row.Scan(&walletId, &playerId, &currency, &balanceMinorUnits, &version, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app.ErrWalletNotFound
		}

		return nil, fmt.Errorf("postgres: error on finding wallet by id: %w", err)
	}

	balance, err := domain.FromMinorUnits(balanceMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance from database: %w", err)
	}

	return domain.RehydrateWallet(
		walletID,
		domain.PlayerID(playerId),
		balance,
		version,
		createdAt,
		updatedAt,
	)
}

func (r *walletRepository) Update(ctx context.Context, wallet *domain.Wallet, prevVersion int64) error {
	exec := executor(ctx, r.pool)

	tag, err := exec.Exec(ctx, `
		UPDATE wallets
		SET balance_minor_units = $1, version = $2, updated_at = $3
		WHERE id = $4 AND version = $5
	`, wallet.Balance().MinorUnits(), wallet.Version(), wallet.UpdatedAt(), wallet.ID(), prevVersion,
	)
	if err != nil {
		return fmt.Errorf("postgres: error on updating wallet: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return app.ErrWalletConcurrentUpdate
	}

	return nil
}

func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
	}
	return false
}
