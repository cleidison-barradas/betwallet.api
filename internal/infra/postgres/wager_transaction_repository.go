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
		NullableStr(wagerTransaction.ProviderID()), NullIfEmpty(string(wagerTransaction.ExternalTransactionID())), NullIfEmpty(wagerTransaction.IdempotencyKey()), NullIfEmpty(wagerTransaction.PayloadHash()),
		NullableStr(wagerTransaction.RoundID()), NullableStr(wagerTransaction.GameID()), NullIfEmpty(wagerTransaction.ReferenceExternalTransactionID()),
		NullableStr(wagerTransaction.ResolvedReferenceID()), NullIfEmpty(wagerTransaction.FailureCode()),
		wagerTransaction.CreatedAt(), wagerTransaction.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("postgres: error on creating wager transaction: %w", err)
	}

	return nil
}

func (r *wagerTransactionRepository) FindByID(ctx context.Context, transactionID domain.TransactionID) (*domain.WagerTransaction, error) {
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
	`
	row := exec.QueryRow(ctx, query, transactionID)

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
			return nil, domain.ErrWagerTransctionNotFound
		}

		return nil, fmt.Errorf("postgres: error on finding wager transaction by id: %w", err)
	}

	balance, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance from database: %w", err)
	}

	var providerId *domain.ProviderID
	if providerID != nil {
		val := domain.ProviderID(*providerID)
		providerId = &val
	}

	var roundId *domain.RoundID
	if roundID != nil {
		val := domain.RoundID(*roundID)
		roundId = &val
	}

	var gameId *domain.GameID
	if gameID != nil {
		val := domain.GameID(*gameID)
		gameId = &val
	}

	var resolvedReferenceId *domain.TransactionID
	if resolvedReferenceID != nil {
		val := domain.TransactionID(*resolvedReferenceID)
		resolvedReferenceId = &val
	}

	var referenceExternalTxIDStr string
	if referenceExternalTxID != nil {
		referenceExternalTxIDStr = *referenceExternalTxID
	}

	var externalTransactionIDStr string
	if externalTransactionID != nil {
		externalTransactionIDStr = *externalTransactionID
	}

	var idempotencyKeyStr string
	if idempotencyKey != nil {
		idempotencyKeyStr = *idempotencyKey
	}

	var payloadHashStr string
	if payloadHash != nil {
		payloadHashStr = *payloadHash
	}

	var failureCodeStr string
	if failureCode != nil {
		failureCodeStr = *failureCode
	}

	return domain.RehydrateWagerTransaction(
		domain.TransactionID(id),
		domain.WagerKind(kind),
		domain.WagerStatus(status),
		domain.WalletID(walletID),
		domain.PlayerID(playerID),
		domain.Money(balance),
		providerId,
		referenceExternalTxIDStr,
		idempotencyKeyStr,
		payloadHashStr,
		roundId,
		gameId,
		externalTransactionIDStr,
		resolvedReferenceId,
		failureCodeStr,
		createdAt,
		updatedAt,
	)
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
			return nil, domain.ErrWagerTransctionNotFound
		}

		return nil, fmt.Errorf("postgres: error on finding wager transaction by id: %w", err)
	}

	balance, err := domain.FromMinorUnits(amountMinorUnits, domain.Currency(currency))
	if err != nil {
		return nil, fmt.Errorf("postgres: failure on get balance from database: %w", err)
	}

	var providerId *domain.ProviderID
	if providerID != nil {
		val := domain.ProviderID(*providerID)
		providerId = &val
	}

	var roundId *domain.RoundID
	if roundID != nil {
		val := domain.RoundID(*roundID)
		roundId = &val
	}

	var gameId *domain.GameID
	if gameID != nil {
		val := domain.GameID(*gameID)
		gameId = &val
	}

	var resolvedReferenceId *domain.TransactionID
	if resolvedReferenceID != nil {
		val := domain.TransactionID(*resolvedReferenceID)
		resolvedReferenceId = &val
	}

	var referenceExternalTxIDStr string
	if referenceExternalTxID != nil {
		referenceExternalTxIDStr = *referenceExternalTxID
	}

	var externalTransactionIDStr string
	if externalTransactionID != nil {
		externalTransactionIDStr = *externalTransactionID
	}

	var idempotencyKeyStr string
	if idempotencyKey != nil {
		idempotencyKeyStr = *idempotencyKey
	}

	var payloadHashStr string
	if payloadHash != nil {
		payloadHashStr = *payloadHash
	}

	var failureCodeStr string
	if failureCode != nil {
		failureCodeStr = *failureCode
	}

	return domain.RehydrateWagerTransaction(
		domain.TransactionID(id),
		domain.WagerKind(kind),
		domain.WagerStatus(status),
		domain.WalletID(walletID),
		domain.PlayerID(playerID),
		domain.Money(balance),
		providerId,
		referenceExternalTxIDStr,
		idempotencyKeyStr,
		payloadHashStr,
		roundId,
		gameId,
		externalTransactionIDStr,
		resolvedReferenceId,
		failureCodeStr,
		createdAt,
		updatedAt,
	)
}
