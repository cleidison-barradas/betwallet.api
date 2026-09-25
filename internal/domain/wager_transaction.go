package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrWagerInvalidTransactionState = errors.New("wager_transaction: invalid state")
	ErrWagerInvalidTransactionType  = errors.New("wager_transaction: invalid transaction type")
	ErrWagerTransactionNotFound     = errors.New("wager_transaction: transaction not found")
	ErrWagerIdempotencyKeyMissing   = errors.New("wager_transaction: idempotency key is missing")
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

type WagerTransaction struct {
	id                    string
	kind                  WagerKind
	status                WagerStatus
	walletID              string
	playerID              string
	money                 Money
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
}

type ExternalWagerInput struct {
	ID                             string
	Kind                           WagerKind
	WalletID                       string
	PlayerID                       string
	Money                          Money
	ProviderID                     string
	ExternalTransactionID          string
	IdempotencyKey                 string
	PayloadHash                    string
	RoundID                        string
	GameID                         string
	ReferenceExternalTransactionID string
}

func NewExternalWagerTransaction(input ExternalWagerInput) (*WagerTransaction, error) {
	if !input.Kind.isExternal() {
		return nil, fmt.Errorf("%w: invalid transaction type", ErrWagerInvalidTransactionType)
	}

	if input.ID == "" || input.WalletID == "" || input.PlayerID == "" {
		return nil, fmt.Errorf("%w: id, wallet id and player id are required", ErrWagerInvalidTransactionState)
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
		providerID:            &input.ProviderID,
		externalTransactionID: &input.ExternalTransactionID,
		idempotencyKey:        &input.IdempotencyKey,
		payloadHash:           &input.PayloadHash,
		roundID:               &input.RoundID,
		gameID:                &input.GameID,
		referenceExternalTxID: &input.ReferenceExternalTransactionID,
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

type NewOpeningTransactionParams struct {
	TransactionID string
	WalletID      string
	PlayerID      string
	Money         Money
}

func NewOpeningTransaction(p NewOpeningTransactionParams) (*WagerTransaction, error) {

	if p.TransactionID == "" || p.WalletID == "" || p.PlayerID == "" {
		return nil, fmt.Errorf(" %w: transactionID, walletID and playerID are required", ErrWagerInvalidTransactionState)
	}

	if p.Money.IsNegative() {
		return nil, fmt.Errorf("%w: money must be positive", ErrWagerInvalidTransactionState)
	}

	now := time.Now().UTC()

	return &WagerTransaction{
		id:        p.TransactionID,
		kind:      KindOpening,
		status:    StatusPending,
		walletID:  p.WalletID,
		playerID:  p.PlayerID,
		money:     p.Money,
		createdAt: now,
		updatedAt: now,
	}, nil
}

type RehydrateWagerTransactionParams struct {
	ID                    string
	Kind                  WagerKind
	Status                WagerStatus
	WalletID              string
	PlayerID              string
	Money                 Money
	ProviderID            *string
	ExternalTransactionID *string
	IdempotencyKey        *string
	PayloadHash           *string
	RoundID               *string
	GameID                *string
	ReferenceExternalTxID *string
	ResolvedReferenceID   *string
	FailureCode           *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func RehydrateWagerTransaction(p RehydrateWagerTransactionParams) (*WagerTransaction, error) {

	return &WagerTransaction{
		id:                    p.ID,
		kind:                  p.Kind,
		status:                p.Status,
		walletID:              p.WalletID,
		playerID:              p.PlayerID,
		money:                 p.Money,
		providerID:            p.ProviderID,
		externalTransactionID: p.ExternalTransactionID,
		idempotencyKey:        p.IdempotencyKey,
		payloadHash:           p.PayloadHash,
		roundID:               p.RoundID,
		gameID:                p.GameID,
		referenceExternalTxID: p.ReferenceExternalTxID,
		resolvedReferenceID:   p.ResolvedReferenceID,
		failureCode:           p.FailureCode,
		createdAt:             p.CreatedAt,
		updatedAt:             p.UpdatedAt,
	}, nil
}

func (t *WagerTransaction) ID() string                              { return t.id }
func (t *WagerTransaction) Kind() WagerKind                         { return t.kind }
func (t *WagerTransaction) Status() WagerStatus                     { return t.status }
func (t *WagerTransaction) WalletID() string                        { return t.walletID }
func (t *WagerTransaction) PlayerID() string                        { return t.playerID }
func (t *WagerTransaction) Money() Money                            { return t.money }
func (t *WagerTransaction) ProviderID() *string                     { return t.providerID }
func (t *WagerTransaction) ExternalTransactionID() *string          { return t.externalTransactionID }
func (t *WagerTransaction) IdempotencyKey() *string                 { return t.idempotencyKey }
func (t *WagerTransaction) PayloadHash() *string                    { return t.payloadHash }
func (t *WagerTransaction) RoundID() *string                        { return t.roundID }
func (t *WagerTransaction) GameID() *string                         { return t.gameID }
func (t *WagerTransaction) ReferenceExternalTransactionID() *string { return t.referenceExternalTxID }
func (t *WagerTransaction) FailureCode() *string                    { return t.failureCode }
func (t *WagerTransaction) UpdatedAt() time.Time                    { return t.updatedAt }
func (t *WagerTransaction) CreatedAt() time.Time                    { return t.createdAt }
func (t *WagerTransaction) ResolvedReferenceID() *string            { return t.resolvedReferenceID }

func (t *WagerTransaction) transitionGuard() error {
	if t.status.isTerminal() {
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

func (t *WagerTransaction) MarkProcessed(resolvedReferenceID string) error {
	if err := t.transitionGuard(); err != nil {
		return err
	}

	if t.status != StatusPending && t.status != StatusPendingReference {
		return fmt.Errorf("%w: It is not possible to process from %s", ErrWagerInvalidTransactionState, t.status)
	}

	t.status = StatusProcessed
	t.resolvedReferenceID = &resolvedReferenceID
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
	t.failureCode = &failureCode
	t.updatedAt = time.Now().UTC()

	return nil
}
