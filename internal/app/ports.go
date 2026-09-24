package app

import (
	"context"
	"errors"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

var (
	ErrWalletNotFound      = errors.New("app: wallet not found")
	ErrWalletAlreadyExists = errors.New("app: already exists wallet to this player and currency")
)

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(context.Context) error) error
}

type WalletRepository interface {
	Create(ctx context.Context, wallet *domain.Wallet) error
	FindByID(ctx context.Context, walletID domain.WalletID) (*domain.Wallet, error)
}

type WagerTransactionRepository interface {
	Save(ctx context.Context, wagerTransaction *domain.WagerTransaction) error
}

type LedgerRepository interface {
	Append(ctx context.Context, ledgerEntry *domain.WalletLedgerEntry) error
}

type OutboxRepository interface {
	Save(ctx context.Context, outboxEntry *domain.OutboxEntry) error
}
