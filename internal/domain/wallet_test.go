package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewWallet(t *testing.T) {
	t.Run("Create wallet with version 1 sucessfully", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		balance, _ := Parse("100.00", BRL)

		w, err := NewWallet(walletID, "player-1", balance)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if w.Version() != 1 {
			t.Errorf("expected version 1, got %d", w.Version())
		}

		if w.Balance().String() != "100.00" {
			t.Errorf("expected balance 100.00, got %s", w.Balance().String())
		}

		if w.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", w.Currency())
		}
	})

	t.Run("Accept balance 0", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())

		zero, _ := Zero(BRL)
		w, err := NewWallet(walletID, "player-1", zero)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !w.Balance().IsZero() {
			t.Error("balance must be zero")
		}
	})

	t.Run("Reject inital balance negative", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())

		balance, _ := Parse("-10.00", BRL)
		_, err := NewWallet(walletID, "player-1", balance)

		if !errors.Is(err, ErrWalletNotEnoughBalance) {
			t.Errorf("expected error %v, got %v", ErrWalletNotEnoughBalance, err)
		}
	})

	t.Run("Reject if playerID is empty", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())

		balance, _ := Parse("10.00", BRL)
		_, err := NewWallet(walletID, "", balance)

		if !errors.Is(err, ErrWalletStateInvalid) {
			t.Errorf("expected error %v, got %v", ErrWalletStateInvalid, err)
		}
	})
}

func TestRehydrateWallet(t *testing.T) {
	t.Run("Reconstructs the state without altering the version", func(t *testing.T) {
		balance, _ := Parse("100.00", BRL)
		createdAt := time.Now().UTC().Add(-24 * time.Hour)
		updatedAt := time.Now().UTC()
		walletID := WalletID(uuid.New().String())

		w, err := RehydrateWallet(walletID, "player-1", balance, 7, createdAt, updatedAt)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if w.Version() != 7 {
			t.Errorf("expected version 7, got %d", w.Version())
		}

		if w.Balance().String() != "100.00" {
			t.Errorf("expected balance 100.00, got %s", w.Balance().String())
		}
	})

	t.Run("Reject version less than 1", func(t *testing.T) {
		balance, _ := Parse("100.00", BRL)
		walletID := WalletID(uuid.New().String())

		_, err := RehydrateWallet(walletID, "player-1", balance, 0, time.Now(), time.Now())

		if !errors.Is(err, ErrWalletStateInvalid) {
			t.Errorf("expected error %v, got %v", ErrWalletStateInvalid, err)
		}
	})

	t.Run("Reject balance negative", func(t *testing.T) {
		balance, _ := Parse("-10.00", BRL)
		walletID := WalletID(uuid.New().String())

		_, err := RehydrateWallet(walletID, "player-1", balance, 1, time.Now(), time.Now())

		if !errors.Is(err, ErrWalletNotEnoughBalance) {
			t.Errorf("expected error %v, got %v", ErrWalletNotEnoughBalance, err)
		}
	})
}

func TestWallet_Debit(t *testing.T) {
	t.Run("Successfully debits and increments version", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		balance, _ := Parse("100.00", BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		expectAmount := "70.00"

		amount, _ := Parse("30.00", BRL)
		mov, err := w.Debit(amount)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if w.Balance().String() != expectAmount {
			t.Errorf("expected balance %s, got %s", expectAmount, w.Balance().String())
		}

		if w.Version() != 2 {
			t.Errorf("expected version 2, got %d", w.Version())
		}

		if mov.BalanceBefore.String() != "100.00" || mov.BalanceAfter.String() != expectAmount {
			t.Errorf("expected balance before %s and after %s, got %s and %s", "100.00", expectAmount, mov.BalanceBefore.String(), mov.BalanceAfter.String())
		}
	})

	t.Run("Rejects a debit that would result in a negative balance", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		initialAmount := "50.00"
		debitAmount := "80.00"

		balance, _ := Parse(initialAmount, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		amount, _ := Parse(debitAmount, BRL)
		_, err := w.Debit(amount)

		if !errors.Is(err, ErrWalletNotEnoughBalance) {
			t.Errorf("expected error %v, got %v", ErrWalletNotEnoughBalance, err)
		}

		if w.Balance().String() != initialAmount {
			t.Errorf("expected balance %s, got %s", initialAmount, w.Balance().String())
		}

		if w.Version() != 1 {
			t.Errorf("expected version 1, got %d", w.Version())
		}
	})

	t.Run("Allows a debit that brings the balance exactly to zero", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		var initialAmount = "50.00"
		var debitAmount = "50.00"

		balance, _ := Parse(initialAmount, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		amount, _ := Parse(debitAmount, BRL)
		_, err := w.Debit(amount)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !w.Balance().IsZero() {
			t.Errorf("expected balance to be zero, got %s", w.Balance().String())
		}
	})

	t.Run("Rejects balance zero", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		var initialBalance = "50.00"

		balance, _ := Parse(initialBalance, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		zero, _ := Zero(BRL)
		_, err := w.Debit(zero)

		if !errors.Is(err, ErrWalletNotEnoughBalance) {
			t.Errorf("expected error %v, got %v", ErrWalletNotEnoughBalance, err)
		}
	})

	t.Run("Rejects currency different from the wallet's", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		var initialBalance = "50.00"

		balance, _ := Parse(initialBalance, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		amount, _ := Parse("50.00", USD)
		_, err := w.Debit(amount)

		if !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("expected error %v, got %v", ErrInvalidCurrency, err)
		}
	})
}

func TestWallet_Credit(t *testing.T) {
	t.Run("Successfully credits and increments version", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		var initialBalance = "50.00"
		var creditAmount = "30.00"

		balance, _ := Parse(initialBalance, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		amount, _ := Parse(creditAmount, BRL)
		mov, err := w.Credit(amount)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if w.Balance().String() != "80.00" {
			t.Errorf("expected balance 80.00, got %s", w.Balance().String())
		}

		if w.Version() != 2 {
			t.Errorf("expected version 2, got %d", w.Version())
		}

		if mov.BalanceBefore.String() != initialBalance || mov.BalanceAfter.String() != "80.00" {
			t.Errorf("expected balance before %s and after %s, got %s and %s", initialBalance, "80.00", mov.BalanceBefore.String(), mov.BalanceAfter.String())
		}
	})

	t.Run("Rejects balance zero", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		var initialBalance = "50.00"

		balance, _ := Parse(initialBalance, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		zero, _ := Zero(BRL)
		_, err := w.Credit(zero)

		if !errors.Is(err, ErrWalletNotEnoughBalance) {
			t.Errorf("expected error %v, got %v", ErrWalletNotEnoughBalance, err)
		}
	})

	t.Run("Rejects currency different from the wallet's", func(t *testing.T) {
		walletID := WalletID(uuid.New().String())
		var initialBalance = "50.00"

		balance, _ := Parse(initialBalance, BRL)
		w, _ := NewWallet(walletID, "player-1", balance)

		usd, _ := Parse("50.00", USD)
		_, err := w.Credit(usd)

		if !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("expected error %v, got %v", ErrInvalidCurrency, err)
		}

	})
}

func TestWallet_DebitCredit_Sequence(t *testing.T) {
	walletID := WalletID(uuid.New().String())
	balance, _ := Parse("100.00", BRL)
	w, _ := NewWallet(walletID, "player-1", balance)

	bet1, _ := Parse("80.00", BRL)
	bet2, _ := Parse("80.00", BRL)

	_, err1 := w.Debit(bet1)
	_, err2 := w.Debit(bet2)

	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}

	if !errors.Is(err2, ErrWalletNotEnoughBalance) {
		t.Errorf("expected error %v, got %v", ErrWalletNotEnoughBalance, err2)
	}

	if w.Balance().String() != "20.00" {
		t.Errorf("expected balance 20.00, got %s", w.Balance().String())
	}

	if w.Version() != 2 {
		t.Errorf("expected version 2, got %d", w.Version())
	}
}
