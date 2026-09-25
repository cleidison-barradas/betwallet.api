package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrLedgerMissingRequiredFields = errors.New("ledger_missing_required_fields")
	ErrLedgerInvalidDirection      = errors.New("ledger_invalid_direction")
	ErrLedgerNegativeAmount        = errors.New("ledger_negative_amount")
	ErrLedgerInvalidBalance        = errors.New("ledger_invalid_balance")
)

type Direction string

const (
	DirectionCredit Direction = "CREDIT"
	DirectionDebit  Direction = "DEBIT"
)

func (d Direction) valid() bool {
	return d == DirectionCredit || d == DirectionDebit
}

type WalletLedgerEntry struct {
	id            string
	walletID      string
	transactionID string
	direction     Direction
	amount        Money
	balanceBefore Money
	balanceAfter  Money
	createdAt     time.Time
}

type NewWalletLedgerEntryParams struct {
	ID            string
	WalletID      string
	TransactionID string
	Direction     Direction
	Amount        Money
	BalanceBefore Money
	BalanceAfter  Money
}

func NewWalletLedgerEntry(p NewWalletLedgerEntryParams) (*WalletLedgerEntry, error) {

	if p.ID == "" || p.WalletID == "" || p.TransactionID == "" {
		return nil, fmt.Errorf("%w: id, walletID and transactionID are required", ErrLedgerMissingRequiredFields)
	}

	if !p.Direction.valid() {
		return nil, fmt.Errorf("%w: invalid direction %s", ErrLedgerInvalidDirection, p.Direction)
	}

	if !p.Amount.IsPositive() {
		return nil, fmt.Errorf("%w: amount must be positive", ErrLedgerNegativeAmount)
	}

	expectedAfter, err := applyDirection(p.Direction, p.BalanceBefore, p.Amount)
	if err != nil {
		return nil, fmt.Errorf("occurred error applying direction: %v", err)
	}

	cmp, err := expectedAfter.Cmp(p.BalanceAfter)
	if err != nil {
		return nil, fmt.Errorf("occurred error comparing balances: %v", err)
	}

	if cmp != 0 {
		return nil, fmt.Errorf("%w: expected balance after %s, got %s", ErrLedgerInvalidBalance, expectedAfter.String(), p.BalanceAfter.String())
	}

	return &WalletLedgerEntry{
		id:            p.ID,
		walletID:      p.WalletID,
		transactionID: p.TransactionID,
		direction:     p.Direction,
		amount:        p.Amount,
		balanceBefore: p.BalanceBefore,
		balanceAfter:  p.BalanceAfter,
		createdAt:     time.Now().UTC(),
	}, nil

}

func applyDirection(direction Direction, balanceBefore, amount Money) (Money, error) {
	if direction == DirectionDebit {
		return balanceBefore.Sub(amount)
	}
	return balanceBefore.Add(amount)
}

type RehydrateWalletLedgerEntryParams struct {
	ID            string
	WalletID      string
	TransactionID string
	Direction     Direction
	Amount        Money
	BalanceBefore Money
	BalanceAfter  Money
	CreatedAt     time.Time
}

func RehydrateWalletLedgerEntry(p RehydrateWalletLedgerEntryParams) (*WalletLedgerEntry, error) {
	entry, err := NewWalletLedgerEntry(NewWalletLedgerEntryParams{
		ID:            p.ID,
		WalletID:      p.WalletID,
		TransactionID: p.TransactionID,
		Direction:     p.Direction,
		Amount:        p.Amount,
		BalanceBefore: p.BalanceBefore,
		BalanceAfter:  p.BalanceAfter,
	})
	if err != nil {
		return nil, err
	}
	entry.createdAt = p.CreatedAt
	return entry, nil
}

func (e *WalletLedgerEntry) ID() string            { return e.id }
func (e *WalletLedgerEntry) WalletID() string      { return e.walletID }
func (e *WalletLedgerEntry) TransactionID() string { return e.transactionID }
func (e *WalletLedgerEntry) Direction() Direction  { return e.direction }
func (e *WalletLedgerEntry) Amount() Money         { return e.amount }
func (e *WalletLedgerEntry) BalanceBefore() Money  { return e.balanceBefore }
func (e *WalletLedgerEntry) BalanceAfter() Money   { return e.balanceAfter }
func (e *WalletLedgerEntry) CreatedAt() time.Time  { return e.createdAt }
