package domain

import (
	"fmt"
	"time"
)

var (
	ErrWalletNotEnoughBalance = fmt.Errorf("insufficient balance")
	ErrWalletStateInvalid     = fmt.Errorf("invalid wallet state")
)

type Wallet struct {
	id        string
	playerID  string
	balance   Money
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

type NewWalletParams struct {
	WalletID       string
	PlayerID       string
	InitialBalance Money
}

type RehydrateWalletParams struct {
	WalletID  string
	PlayerID  string
	Balance   Money
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewWallet(p NewWalletParams) (*Wallet, error) {

	if p.WalletID == "" || p.PlayerID == "" {
		return nil, fmt.Errorf("%w: walletID and playerID are required", ErrWalletStateInvalid)
	}

	if p.InitialBalance.IsNegative() {
		return nil, fmt.Errorf("%w: insufficient balance", ErrWalletNotEnoughBalance)
	}

	now := time.Now().UTC()

	return &Wallet{
		id:        p.WalletID,
		playerID:  p.PlayerID,
		balance:   p.InitialBalance,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RehydrateWallet(p RehydrateWalletParams) (*Wallet, error) {

	if p.Version < 1 {
		return nil, fmt.Errorf("%w: invalid version (%d)", ErrWalletStateInvalid, p.Version)
	}

	if p.Balance.IsNegative() {
		return nil, fmt.Errorf("%w: insufficient balance", ErrWalletNotEnoughBalance)
	}

	return &Wallet{
		id:        p.WalletID,
		playerID:  p.PlayerID,
		balance:   p.Balance,
		version:   p.Version,
		createdAt: p.CreatedAt,
		updatedAt: p.UpdatedAt,
	}, nil
}

type Movement struct {
	BalanceBefore Money
	BalanceAfter  Money
}

func (w *Wallet) ID() string           { return w.id }
func (w *Wallet) PlayerID() string     { return w.playerID }
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
