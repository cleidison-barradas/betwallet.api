package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrWalletNotEnoughBalance = fmt.Errorf("insufficient balance")
	ErrWalletStateInvalid     = fmt.Errorf("invalid wallet state")
)

type PlayerID string
type WalletID uuid.UUID

type Wallet struct {
	id        WalletID
	playerID  PlayerID
	balance   Money
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

func NewWallet(playerID PlayerID, initialBalance Money) (*Wallet, error) {

	if playerID == "" {
		return nil, fmt.Errorf("%w: player id is required", ErrWalletStateInvalid)
	}

	if initialBalance.IsNegative() {
		return nil, fmt.Errorf("%w: insufficient balance", ErrWalletNotEnoughBalance)
	}

	now := time.Now().UTC()

	return &Wallet{
		playerID:  playerID,
		balance:   initialBalance,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RehydrateWallet(id WalletID, playerID PlayerID, balance Money, version int64, createdAt, updatedAt time.Time) (*Wallet, error) {

	if version < 1 {
		return nil, fmt.Errorf("%w: invalid version (%d)", ErrWalletStateInvalid, version)
	}

	if balance.IsNegative() {
		return nil, fmt.Errorf("%w: insufficient balance", ErrWalletNotEnoughBalance)
	}

	return &Wallet{
		id:        id,
		playerID:  playerID,
		balance:   balance,
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

type Movement struct {
	BalanceBefore Money
	BalanceAfter  Money
}

func (w *Wallet) ID() WalletID         { return w.id }
func (w *Wallet) PlayerID() PlayerID   { return w.playerID }
func (w *Wallet) Balance() Money       { return w.balance }
func (w *Wallet) Currency() Currency   { return w.balance.Currency() }
func (w *Wallet) Version() int64       { return w.version }
func (w *Wallet) CreatedAt() time.Time { return w.createdAt }
func (w *Wallet) UpdatedAt() time.Time { return w.updatedAt }

func (w *Wallet) Debit(amount Money) (Movement, error) {
	if !amount.IsPositive() {
		return Movement{}, fmt.Errorf("%w: amount must be positive", ErrWalletNotEnoughBalance)
	}

	if amount.Currency() != w.balance.Currency() {
		return Movement{}, fmt.Errorf("%w: %s vs %s", ErrInvalidCurrency, amount.Currency(), w.balance.Currency())
	}

	newBalance, err := w.balance.Sub(amount)
	if err != nil {
		return Movement{}, err
	}

	if newBalance.IsNegative() {
		return Movement{}, ErrWalletNotEnoughBalance
	}

	before := w.balance
	w.balance = newBalance
	w.version++
	w.updatedAt = time.Now().UTC()

	return Movement{
		BalanceBefore: before,
		BalanceAfter:  w.balance,
	}, nil
}
func (w *Wallet) Credit(amount Money) (Movement, error) {
	if !amount.IsPositive() {
		return Movement{}, fmt.Errorf("%w: amount must be positive", ErrWalletNotEnoughBalance)
	}

	if amount.Currency() != w.balance.Currency() {
		return Movement{}, fmt.Errorf("%w: %s vs %s", ErrInvalidCurrency, amount.Currency(), w.balance.Currency())
	}

	newBalance, err := w.balance.Add(amount)
	if err != nil {
		return Movement{}, err
	}

	before := w.balance
	w.balance = newBalance
	w.version++
	w.updatedAt = time.Now().UTC()

	return Movement{
		BalanceBefore: before,
		BalanceAfter:  w.balance,
	}, nil
}
