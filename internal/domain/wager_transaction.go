package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrWagerInvalidTransactionState = errors.New("wager_transaction: invalid state")
	ErrWagerInvalidTransactionType  = errors.New("wager_transaction: invalid transaction type")
)

type WagerKind string

const (
	KindOpening  WagerKind = "OPENING"
	KindBet      WagerKind = "BET"
	KindWin      WagerKind = "WIN"
	KindLoss     WagerKind = "LOSS"
	KingRefund   WagerKind = "REFUND"
	kingRollback WagerKind = "ROLLBACK"
)

func (k WagerKind) isExternal() bool {
	switch k {
	case KindBet, KindWin, KindLoss, KingRefund, kingRollback:
		return true
	default:
		return false
	}
}

func (k WagerKind) requiresReference() bool {
	return k == KingRefund || k == kingRollback
}

type WagerStatus string

const (
	StatusPending          WagerStatus = "PENDING"
	StatusPendingReference WagerStatus = "PENDING_REFERENCE"
	StatusProcessed        WagerStatus = "PROCESSED"
	StatusRejected         WagerStatus = "REJECTED"
	StatusFailed           WagerStatus = "FAILED"
)

func (s WagerStatus) isTerminal() bool {
	switch s {
	case StatusProcessed, StatusRejected, StatusFailed:
		return true
	default:
		return false
	}
}

type TransactionID uuid.UUID
type ProviderID string
type RoundID string
type GameID string

type WagerTransaction struct {
	id                    TransactionID
	kind                  WagerKind
	status                WagerStatus
	walletID              WalletID
	playerID              PlayerID
	money                 Money
	providerID            ProviderID
	externalTransactionID string
	idempotencyKey        string
	payloadHash           string
	roundID               RoundID
	gameID                GameID
	referenceExternalTxID string
	resolvedReferenceID   TransactionID
	failureCode           string
	createdAt             time.Time
	updatedAt             time.Time
}

type ExternalWagerInput struct {
	ID                             TransactionID
	Kind                           WagerKind
	WalletID                       WalletID
	PlayerID                       PlayerID
	Money                          Money
	ProviderID                     ProviderID
	ExternalTransactionID          string
	IdempotencyKey                 string
	PayloadHash                    string
	RoundID                        RoundID
	GameID                         GameID
	ReferenceExternalTransactionID string
}

func NewExternalWagerTransaction(input ExternalWagerInput) (*WagerTransaction, error) {
	if !input.Kind.isExternal() {
		return nil, fmt.Errorf("%w: invalid transaction type", ErrWagerInvalidTransactionType)
	}

	if input.PlayerID == "" {
		return nil, fmt.Errorf("%w: player id is required", ErrWagerInvalidTransactionState)
	}

	if input.ProviderID == "" || input.ExternalTransactionID == "" || input.IdempotencyKey == "" {
		return nil, fmt.Errorf("%w: provider id, external transaction id and idempotency key are required", ErrWagerInvalidTransactionState)
	}

	if input.RoundID == "" || input.GameID == "" {
		return nil, fmt.Errorf("%w: round id and game id are required", ErrWagerInvalidTransactionState)
	}

	if err := validateMoneyForKind(input.Kind, input.Money); err != nil {
		return nil, err
	}

	hasReference := input.ReferenceExternalTransactionID != ""

	if input.Kind.requiresReference() && !hasReference {
		return nil, fmt.Errorf("%w: reference external transaction id is required", ErrWagerInvalidTransactionState)
	}

	if input.Kind.requiresReference() && hasReference {
		return nil, fmt.Errorf("%w: reference external transaction id is required", ErrWagerInvalidTransactionState)
	}

	now := time.Now().UTC()

	return &WagerTransaction{
		id:                    input.ID,
		kind:                  input.Kind,
		status:                StatusPending,
		walletID:              input.WalletID,
		playerID:              input.PlayerID,
		money:                 input.Money,
		providerID:            input.ProviderID,
		externalTransactionID: input.ExternalTransactionID,
		idempotencyKey:        input.IdempotencyKey,
		payloadHash:           input.PayloadHash,
		roundID:               input.RoundID,
		gameID:                input.GameID,
		referenceExternalTxID: input.ReferenceExternalTransactionID,
		createdAt:             now,
		updatedAt:             now,
	}, nil

}

func validateMoneyForKind(kind WagerKind, money Money) error {
	if kind == KindLoss {
		if !money.IsZero() {
			return fmt.Errorf("%w: money must be zero", ErrWagerInvalidTransactionState)
		}
		return nil
	}

	if !money.IsPositive() {
		return fmt.Errorf("%w: money must be positive", ErrWagerInvalidTransactionState)
	}
	return nil
}

func NewOpeningTransaction(transactionID TransactionID, walletID WalletID, player PlayerID, money Money) (*WagerTransaction, error) {

	if player == "" {
		return nil, fmt.Errorf("%w: player id is required", ErrWagerInvalidTransactionState)
	}

	if money.IsNegative() {
		return nil, fmt.Errorf("%w: money must be positive", ErrWagerInvalidTransactionState)
	}

	now := time.Now().UTC()

	return &WagerTransaction{
		id:        transactionID,
		kind:      KindOpening,
		status:    StatusPending,
		walletID:  walletID,
		playerID:  player,
		money:     money,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RehydrateWagerTransaction(
	id TransactionID,
	kind WagerKind,
	status WagerStatus,
	walletID WalletID,
	playerID PlayerID,
	money Money,
	providerID ProviderID,
	externalTransactionID string,
	idempotencyKey string,
	payloadHash string,
	roundID RoundID,
	gameID GameID,
	referenceExternalTxID string,
	resolvedReferenceID TransactionID,
	failureCode string,
	createdAt, updatedAt time.Time,
) (*WagerTransaction, error) {

	return &WagerTransaction{
		id:                    id,
		kind:                  kind,
		status:                status,
		walletID:              walletID,
		playerID:              playerID,
		money:                 money,
		providerID:            providerID,
		externalTransactionID: externalTransactionID,
		idempotencyKey:        idempotencyKey,
		payloadHash:           payloadHash,
		roundID:               roundID,
		gameID:                gameID,
		referenceExternalTxID: referenceExternalTxID,
		resolvedReferenceID:   resolvedReferenceID,
		failureCode:           failureCode,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
	}, nil
}

func (t *WagerTransaction) ID() TransactionID                      { return t.id }
func (t *WagerTransaction) Kind() WagerKind                        { return t.kind }
func (t *WagerTransaction) Status() WagerStatus                    { return t.status }
func (t *WagerTransaction) WalletID() WalletID                     { return t.walletID }
func (t *WagerTransaction) PlayerID() PlayerID                     { return t.playerID }
func (t *WagerTransaction) Money() Money                           { return t.money }
func (t *WagerTransaction) ProviderID() ProviderID                 { return t.providerID }
func (t *WagerTransaction) ExternalTransactionID() string          { return t.externalTransactionID }
func (t *WagerTransaction) IdempotencyKey() string                 { return t.idempotencyKey }
func (t *WagerTransaction) PayloadHash() string                    { return t.payloadHash }
func (t *WagerTransaction) RoundID() RoundID                       { return t.roundID }
func (t *WagerTransaction) GameID() GameID                         { return t.gameID }
func (t *WagerTransaction) ReferenceExternalTransactionID() string { return t.referenceExternalTxID }
func (t *WagerTransaction) FailureCode() string                    { return t.failureCode }
func (t *WagerTransaction) UpdatedAt() time.Time                   { return t.updatedAt }
func (t *WagerTransaction) ToStringID() string                     { return uuid.UUID(t.id).String() }

func (t *WagerTransaction) transitionGuard() error {
	if t.status != StatusPending {
		return fmt.Errorf("%w: The transaction is already in a terminal state.", ErrWagerInvalidTransactionState)
	}
	return nil
}

func (t *WagerTransaction) MarkPendingReference() error {
	if t.status != StatusPending {
		return fmt.Errorf("%w: It is not possible to process from %s", ErrWagerInvalidTransactionState, t.status)
	}

	t.status = StatusPendingReference
	t.updatedAt = time.Now().UTC()

	return nil
}

func (t *WagerTransaction) MarkProcessed(resolvedReferenceID TransactionID) error {
	if err := t.transitionGuard(); err != nil {
		return err
	}

	if t.status != StatusPending && t.status != StatusPendingReference {
		return fmt.Errorf("%w: It is not possible to process from %s", ErrWagerInvalidTransactionState, t.status)
	}

	t.status = StatusProcessed
	t.resolvedReferenceID = resolvedReferenceID
	t.updatedAt = time.Now().UTC()

	return nil
}

func (t *WagerTransaction) MarkRejected(failureCode string) error {
	if err := t.transitionGuard(); err != nil {
		return err
	}

	if failureCode == "" {
		return fmt.Errorf("%w: failure code is required", ErrWagerInvalidTransactionState)
	}

	t.status = StatusRejected
	t.failureCode = failureCode
	t.updatedAt = time.Now().UTC()

	return nil
}
