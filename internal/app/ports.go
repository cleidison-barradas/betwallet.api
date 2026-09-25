package app

import (
	"context"
	"errors"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

var (
	ErrWalletNotFound         = errors.New("app: wallet not found")
	ErrWalletAlreadyExists    = errors.New("app: already exists wallet to this player and currency")
	ErrWalletConcurrentUpdate = errors.New("app: wallet is being updated concurrently")
	ErrIdempotencyKeyConflict = errors.New("app: idempotency key conflict")
	ErrWalletLedgerNotFound   = errors.New("app: wallet ledger not found")
	ErrIdempotencyKeyRaceLost = errors.New("app: idempotency key race lost")
)

type WalletLedgerCursor struct {
	CreatedAt time.Time
	ID        string
}

type ListWalletLedgerParams struct {
	WalletID string
	Cursor   *WalletLedgerCursor
	Limit    int
}

type WalletLedgerPagination struct {
	Entries []domain.WalletLedgerEntry
	Cursor  *WalletLedgerCursor
	HasNext bool
}

type FindWagerTransactionByProviderParams struct {
	ProviderID   string
	ExternalTxID string
}

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(context.Context) error) error
}

type WalletRepository interface {
	Create(ctx context.Context, wallet *domain.Wallet) error
	FindByID(ctx context.Context, walletID string) (*domain.Wallet, error)
	Update(ctx context.Context, wallet *domain.Wallet, prevVersion int64) error
}

type WagerTransactionRepository interface {
	Save(ctx context.Context, wagerTransaction *domain.WagerTransaction) error
	FindByID(ctx context.Context, transactionID string, providerID string) (*domain.WagerTransaction, error)
	FindByProvider(ctx context.Context, params FindWagerTransactionByProviderParams) (*domain.WagerTransaction, error)
	FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.WagerTransaction, error)
}

type WalletLedgerRepository interface {
	Append(ctx context.Context, ledgerEntry *domain.WalletLedgerEntry) error
	ListWalletLedger(ctx context.Context, params ListWalletLedgerParams) (*WalletLedgerPagination, error)
	FindByTransactionID(ctx context.Context, transactionID string) (*domain.WalletLedgerEntry, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, outboxEntry *domain.OutboxEntry) error
}
