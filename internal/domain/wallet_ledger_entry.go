package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidWalletLedgerEntry = errors.New("wallet_ledger_entry: invalid wallet ledger entry")

type Direction string
type LedgerEntryID uuid.UUID

const (
	DirectionCredit Direction = "CREDIT"
	DirectionDebit  Direction = "DEBIT"
)

func (d Direction) valid() bool {
	return d == DirectionCredit || d == DirectionDebit
}

type WalletLedgerEntry struct {
	id            LedgerEntryID
	walletID      WalletID
	transactionID TransactionID
	direction     Direction
	amount        Money
	balanceBefore Money
	balanceAfter  Money
	createdAt     time.Time
}

func NewWalletLedgerEntry(
	id LedgerEntryID,
	walletID WalletID,
	transactionID TransactionID,
	direction Direction,
	amount Money,
	balanceBefore Money,
	balanceAfter Money,
	createdAt time.Time,
) (*WalletLedgerEntry, error) {

	if !direction.valid() {
		return nil, fmt.Errorf("%w: invalid direction %s", ErrInvalidWalletLedgerEntry, direction)
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("%w: amount must be positive", ErrInvalidWalletLedgerEntry)
	}

	expectedAfter, err := applyDirection(direction, balanceBefore, amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWalletLedgerEntry, err)
	}

	cmp, err := expectedAfter.Cmp(balanceAfter)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWalletLedgerEntry, err)
	}

	if cmp != 0 {
		return nil, fmt.Errorf("%w: expected balance after %s, got %s", ErrInvalidWalletLedgerEntry, expectedAfter.String(), balanceAfter.String())
	}

	return &WalletLedgerEntry{
		id:            id,
		walletID:      walletID,
		transactionID: transactionID,
		direction:     direction,
		amount:        amount,
		balanceBefore: balanceBefore,
		balanceAfter:  balanceAfter,
		createdAt:     time.Now().UTC(),
	}, nil

}

func applyDirection(direction Direction, balanceBefore, amount Money) (Money, error) {
	if direction == DirectionDebit {
		return balanceBefore.Sub(amount)
	}
	return balanceBefore.Add(amount)
}

func RehydrateWalletLedgerEntry(
	id LedgerEntryID,
	walletID WalletID,
	transactionID TransactionID,
	direction Direction,
	amount Money,
	balanceBefore Money,
	balanceAfter Money,
	createdAt time.Time,
) (*WalletLedgerEntry, error) {
	entry, err := NewWalletLedgerEntry(id, walletID, transactionID, direction, amount, balanceBefore, balanceAfter, createdAt)
	if err != nil {
		return nil, err
	}
	entry.createdAt = createdAt
	return entry, nil
}

func (e *WalletLedgerEntry) ID() LedgerEntryID            { return e.id }
func (e *WalletLedgerEntry) WalletID() WalletID           { return e.walletID }
func (e *WalletLedgerEntry) TransactionID() TransactionID { return e.transactionID }
func (e *WalletLedgerEntry) Direction() Direction         { return e.direction }
func (e *WalletLedgerEntry) Amount() Money                { return e.amount }
func (e *WalletLedgerEntry) BalanceBefore() Money         { return e.balanceBefore }
func (e *WalletLedgerEntry) BalanceAfter() Money          { return e.balanceAfter }
func (e *WalletLedgerEntry) CreatedAt() time.Time         { return e.createdAt }
