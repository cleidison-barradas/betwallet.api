package app

import (
	"context"
	"errors"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

var (
	ErrWalletNotFound      = errors.New("app: wallet not found")
	ErrWalletAlreadyExists = errors.New("app: already exists wallet to this player and currency")
)

type WalletLedgerCursor struct {
	CreatedAt time.Time
	ID        string
}

type ListWalletLedgerParams struct {
	WalletID domain.WalletID
	Cursor   *WalletLedgerCursor
	Limit    int
}

type WalletLedgerPagination struct {
	Entries []domain.WalletLedgerEntry
	Cursor  *WalletLedgerCursor
	HasNext bool
}

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

type WalletLedgerRepository interface {
	Append(ctx context.Context, ledgerEntry *domain.WalletLedgerEntry) error
	ListWalletLedger(ctx context.Context, params ListWalletLedgerParams) (*WalletLedgerPagination, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, outboxEntry *domain.OutboxEntry) error
}
