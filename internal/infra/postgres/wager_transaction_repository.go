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
		NullableStr(wagerTransaction.ProviderID()), NullableStr(wagerTransaction.ExternalTransactionID()), NullableStr(wagerTransaction.IdempotencyKey()), NullableStr(wagerTransaction.PayloadHash()),
		NullableStr(wagerTransaction.RoundID()), NullableStr(wagerTransaction.GameID()), NullableStr(wagerTransaction.ReferenceExternalTransactionID()),
		NullableStr(wagerTransaction.ResolvedReferenceID()), NullableStr(wagerTransaction.FailureCode()),
		wagerTransaction.CreatedAt(), wagerTransaction.UpdatedAt(),
	)
	if err != nil {
		if isUniqueViolation(err, "wager_transactions_provider_external_id_key") {
			return app.ErrIdempotencyKeyRaceLost
		}

		return fmt.Errorf("postgres: error on creating wager transaction: %w", err)
	}

	return nil
}

func (r *wagerTransactionRepository) FindByID(ctx context.Context, transactionID string, providerID string) (*domain.WagerTransaction, error) {
	exec := executor(ctx, r.pool)

	query := `
		SELECT
			id,
			kind,
			status,
			wallet_id,
			player_id,
			amount_minor_units,
			currency,
			provider_id,
			external_transaction_id,
			idempotency_key,
			payload_hash,
			round_id,
			game_id,
			reference_external_transaction_id,
			resolved_reference_id,
			failure_code,
			created_at,
			updated_at
		FROM wager_transactions
		WHERE id = $1
		AND provider_id = $2
	`
	row := exec.QueryRow(ctx, query, transactionID, providerID)

	var (
		id                    string
		kind                  string
		status                string
		walletID              string
		playerID              string
		amountMinorUnits      int64
		currency              string
		providerId            *string
		externalTransactionID *string
		idempotencyKey        *string
		payloadHash           *string
		roundID               *string
		gameID                *string
		referenceExternalTxID *string
		resolvedReferenceID   *string
		failureCode           *string
		createdAt             time.Time
		updatedAt             time.Time
	)

	if err := row.Scan(
		&id,
		&kind,
		&status,
		&walletID,
		&playerID,
		&amountMinorUnits,
		&currency,
		&providerId,
		&externalTransactionID,
		&idempotencyKey,
		&payloadHash,
		&roundID,
		&gameID,
		&referenceExternalTxID,
		&resolvedReferenceID,
		&failureCode,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWagerTransactionNotFound
		}

		return nil, fmt.Errorf("postgres: error on finding wager transaction by id: %w", err)
	}

	balance, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance from database: %w", err)
	}

	return domain.RehydrateWagerTransaction(domain.RehydrateWagerTransactionParams{
		ID:                    id,
		Kind:                  domain.WagerKind(kind),
		Status:                domain.WagerStatus(status),
		WalletID:              walletID,
		PlayerID:              playerID,
		Money:                 balance,
		ProviderID:            providerId,
		ExternalTransactionID: externalTransactionID,
		IdempotencyKey:        idempotencyKey,
		PayloadHash:           payloadHash,
		RoundID:               roundID,
		GameID:                gameID,
		ReferenceExternalTxID: referenceExternalTxID,
		ResolvedReferenceID:   resolvedReferenceID,
		FailureCode:           failureCode,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	})
}

func (r *wagerTransactionRepository) FindByProvider(ctx context.Context, params app.FindWagerTransactionByProviderParams) (*domain.WagerTransaction, error) {
	exec := executor(ctx, r.pool)

	query := `
		SELECT
			id,
			kind,
			status,
			wallet_id,
			player_id,
			amount_minor_units,
			currency,
			provider_id,
			external_transaction_id,
			idempotency_key,
			payload_hash,
			round_id,
			game_id,
			reference_external_transaction_id,
			resolved_reference_id,
			failure_code,
			created_at,
			updated_at
		FROM wager_transactions
		WHERE provider_id = $1
		AND external_transaction_id = $2
	`
	row := exec.QueryRow(ctx, query, params.ProviderID, params.ExternalTxID)

	var (
		id                    string
		kind                  string
		status                string
		walletID              string
		playerID              string
		amountMinorUnits      int64
		currency              string
		providerID            *string
		externalTransactionID *string
		idempotencyKey        *string
		payloadHash           *string
		roundID               *string
		gameID                *string
		referenceExternalTxID *string
		resolvedReferenceID   *string
		failureCode           *string
		createdAt             time.Time
		updatedAt             time.Time
	)

	if err := row.Scan(
		&id,
		&kind,
		&status,
		&walletID,
		&playerID,
		&amountMinorUnits,
		&currency,
		&providerID,
		&externalTransactionID,
		&idempotencyKey,
		&payloadHash,
		&roundID,
		&gameID,
		&referenceExternalTxID,
		&resolvedReferenceID,
		&failureCode,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWagerTransactionNotFound
		}

		return nil, fmt.Errorf("postgres: error on finding wager transaction by id: %w", err)
	}

	balance, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance from database: %w", err)
	}

	return domain.RehydrateWagerTransaction(domain.RehydrateWagerTransactionParams{
		ID:                    id,
		Kind:                  domain.WagerKind(kind),
		Status:                domain.WagerStatus(status),
		WalletID:              walletID,
		PlayerID:              playerID,
		Money:                 balance,
		ProviderID:            providerID,
		ExternalTransactionID: externalTransactionID,
		IdempotencyKey:        idempotencyKey,
		PayloadHash:           payloadHash,
		RoundID:               roundID,
		GameID:                gameID,
		ReferenceExternalTxID: referenceExternalTxID,
		ResolvedReferenceID:   resolvedReferenceID,
		FailureCode:           failureCode,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	})
}

func (r *wagerTransactionRepository) FindByIdempotencyKey(ctx context.Context, idemPotencyKey string) (*domain.WagerTransaction, error) {
	exec := executor(ctx, r.pool)

	query := `
		SELECT
			id,
			kind,
			status,
			wallet_id,
			player_id,
			amount_minor_units,
			currency,
			provider_id,
			external_transaction_id,
			idempotency_key,
			payload_hash,
			round_id,
			game_id,
			reference_external_transaction_id,
			resolved_reference_id,
			failure_code,
			created_at,
			updated_at
		FROM wager_transactions
		WHERE idempotency_key = $1
	`
	row := exec.QueryRow(ctx, query, idemPotencyKey)

	var (
		id                    string
		kind                  string
		status                string
		walletID              string
		playerID              string
		amountMinorUnits      int64
		currency              string
		providerID            *string
		externalTransactionID *string
		idempotencyKey        *string
		payloadHash           *string
		roundID               *string
		gameID                *string
		referenceExternalTxID *string
		resolvedReferenceID   *string
		failureCode           *string
		createdAt             time.Time
		updatedAt             time.Time
	)

	if err := row.Scan(
		&id,
		&kind,
		&status,
		&walletID,
		&playerID,
		&amountMinorUnits,
		&currency,
		&providerID,
		&externalTransactionID,
		&idempotencyKey,
		&payloadHash,
		&roundID,
		&gameID,
		&referenceExternalTxID,
		&resolvedReferenceID,
		&failureCode,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWagerTransactionNotFound
		}

		return nil, fmt.Errorf("postgres: error on finding wager transaction by id: %w", err)
	}

	balance, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance from database: %w", err)
	}

	return domain.RehydrateWagerTransaction(domain.RehydrateWagerTransactionParams{
		ID:                    id,
		Kind:                  domain.WagerKind(kind),
		Status:                domain.WagerStatus(status),
		WalletID:              walletID,
		PlayerID:              playerID,
		Money:                 balance,
		ProviderID:            providerID,
		ExternalTransactionID: externalTransactionID,
		IdempotencyKey:        idempotencyKey,
		PayloadHash:           payloadHash,
		RoundID:               roundID,
		GameID:                gameID,
		ReferenceExternalTxID: referenceExternalTxID,
		ResolvedReferenceID:   resolvedReferenceID,
		FailureCode:           failureCode,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	})
}
