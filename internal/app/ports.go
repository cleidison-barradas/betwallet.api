package app

import (
	"context"
	"errors"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

var (
	ErrWalletNotFound         = errors.New("app_wallet_not_found")
	ErrWalletAlreadyExists    = errors.New("app_already_exists_wallet_to_this_player_and_currency")
	ErrWalletConcurrentUpdate = errors.New("app_wallet_is_being_updated_concurrently")
	ErrIdempotencyKeyConflict = errors.New("app_idempotency_key_conflict")
	ErrLedgerNotFound         = errors.New("app_ledger_not_found")
	ErrIdempotencyKeyRaceLost = errors.New("app_idempotency_key_race_lost")
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
