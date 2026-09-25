package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

const maxAttempts = 5

type ProcessWagerTransactionCommand struct {
	ProviderID                     string
	ExternalTransactionID          string
	IdempotencyKey                 string
	PlayerID                       string
	WalletID                       string
	RoundID                        string
	GameID                         string
	Kind                           domain.WagerKind
	Money                          domain.Money
	ReferenceExternalTransactionID string
}

type ProcessWagerTransactionResult struct {
	TransactionID    string
	Status           string
	Balance          domain.Money
	IdempotentReplay bool
}

type ProcessWagerTransaction struct {
	uow              UnitOfWork
	walletRepo       WalletRepository
	transactionRepo  WagerTransactionRepository
	walletLedgerRepo WalletLedgerRepository
	outbox           OutboxRepository
	ids              IDGenerator
}

func NewProcessWagerTransaction(
	uow UnitOfWork,
	walletRepo WalletRepository,
	transactionRepo WagerTransactionRepository,
	walletLedgerRepo WalletLedgerRepository,
	outbox OutboxRepository,
	ids IDGenerator,
) *ProcessWagerTransaction {
	return &ProcessWagerTransaction{
		uow:              uow,
		walletRepo:       walletRepo,
		transactionRepo:  transactionRepo,
		walletLedgerRepo: walletLedgerRepo,
		outbox:           outbox,
		ids:              ids,
	}
}

func (uc *ProcessWagerTransaction) Execute(ctx context.Context, cmd ProcessWagerTransactionCommand) (*ProcessWagerTransactionResult, error) {

	if cmd.Kind != domain.KindBet {
		return nil, fmt.Errorf("process wager transaction: king %s not supported", cmd.Kind)
	}

	payloadHash, err := HashPayload(cmd)
	if err != nil {
		return nil, fmt.Errorf("process wager transaction: hash payload: %w", err)
	}

	existing, err := uc.transactionRepo.FindByIdempotencyKey(ctx, cmd.IdempotencyKey)
	if err != nil && !errors.Is(err, domain.ErrWagerTransactionNotFound) {
		return nil, fmt.Errorf("failed to find transaction by idempotency key: %w", err)
	}

	if existing != nil {
		if existing.PayloadHash() != nil && *existing.PayloadHash() != payloadHash {
			return nil, ErrIdempotencyKeyConflict
		}
		return uc.resultFromExisting(ctx, existing)
	}

	var result *ProcessWagerTransactionResult

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		txID := uc.ids.NewID().String()

		err := uc.uow.Execute(ctx, func(ctx context.Context) error {
			wallet, err := uc.walletRepo.FindByID(ctx, cmd.WalletID)
			if err != nil {
				return err
			}

			prevVersion := wallet.Version()

			wagerTx, err := domain.NewExternalWagerTransaction(domain.ExternalWagerInput{
				ID:                    txID,
				Kind:                  cmd.Kind,
				WalletID:              wallet.ID(),
				PlayerID:              wallet.PlayerID(),
				ProviderID:            cmd.ProviderID,
				ExternalTransactionID: cmd.ExternalTransactionID,
				IdempotencyKey:        cmd.IdempotencyKey,
				PayloadHash:           cmd.IdempotencyKey,
				RoundID:               cmd.RoundID,
				GameID:                cmd.GameID,
				Money:                 cmd.Money,
			})

			if err != nil {
				return err
			}

			movement, err := wallet.Debit(cmd.Money)
			if err != nil {
				if errors.Is(err, domain.ErrWalletBalanceNegative) {
					return uc.reject(ctx, wagerTx, wallet, "INSUFFICIENT_BALANCE", &result)
				}
				return err
			}

			if err := wagerTx.MarkProcessed(""); err != nil {
				return err
			}

			if err := uc.transactionRepo.Save(ctx, wagerTx); err != nil {
				if errors.Is(err, ErrIdempotencyKeyRaceLost) {
					return ErrIdempotencyKeyRaceLost
				}
				return err
			}

			entry, err := domain.NewWalletLedgerEntry(domain.NewWalletLedgerEntryParams{
				ID:            uc.ids.NewID().String(),
				WalletID:      wallet.ID(),
				TransactionID: txID,
				Amount:        cmd.Money,
				Direction:     domain.DirectionDebit,
				BalanceBefore: movement.BalanceBefore,
				BalanceAfter:  movement.BalanceAfter,
			})

			if err != nil {
				return err
			}

			if err := uc.walletLedgerRepo.Append(ctx, entry); err != nil {
				return err
			}

			if err := uc.walletRepo.Update(ctx, wallet, prevVersion); err != nil {
				return err
			}

			if err := uc.publishProcessedEvents(ctx, wallet, wagerTx, entry); err != nil {
				return err
			}

			result = &ProcessWagerTransactionResult{
				TransactionID: string(wagerTx.ID()),
				Status:        string(wagerTx.Status()),
				Balance:       wallet.Balance(),
			}
			return nil
		})

		if err == nil {
			return result, nil
		}

		if errors.Is(err, ErrWalletConcurrentUpdate) {
			continue
		}

		if errors.Is(err, ErrIdempotencyKeyRaceLost) {
			existing, findErr := uc.transactionRepo.FindByIdempotencyKey(ctx, cmd.IdempotencyKey)
			if findErr != nil {
				return nil, fmt.Errorf("process wager transaction: resolve race loser: %w", findErr)
			}
			if existing.PayloadHash() != nil && *existing.PayloadHash() != payloadHash {
				return nil, ErrIdempotencyKeyConflict
			}
			return uc.resultFromExisting(ctx, existing)
		}
		return nil, err
	}
	return nil, fmt.Errorf("process wager transaction: failed after %d attempts", maxAttempts)
}

func (uc *ProcessWagerTransaction) reject(ctx context.Context, wagerTx *domain.WagerTransaction, wallet *domain.Wallet, failureCode string, result **ProcessWagerTransactionResult) error {
	if err := wagerTx.MarkRejected(failureCode); err != nil {
		return err
	}

	if err := uc.transactionRepo.Save(ctx, wagerTx); err != nil {
		return err
	}

	payload, err := NewEvent(uc.ids.NewID().String(), "WagerTransactionRejected", string(wallet.ID()), uc.ids.NewID().String(), 1, WagerTransactionRejectedData{
		TransactionID: string(wagerTx.ID()),
		Kind:          string(wagerTx.Kind()),
		WalletID:      string(wallet.ID()),
		FailureCode:   failureCode,
	})
	if err != nil {
		return err
	}

	event, err := domain.NewOutboxEntry(domain.OutboxEventID(uc.ids.NewID().String()), string(wallet.ID()), "WagerTransactionRejected", payload)
	if err != nil {
		return err
	}

	if err := uc.outbox.Save(ctx, event); err != nil {
		return err
	}

	*result = &ProcessWagerTransactionResult{
		TransactionID: wagerTx.ID(),
		Status:        string(wagerTx.Status()),
		Balance:       wallet.Balance(),
	}
	return nil
}

func (uc *ProcessWagerTransaction) publishProcessedEvents(ctx context.Context, wallet *domain.Wallet, wagerTx *domain.WagerTransaction, entry *domain.WalletLedgerEntry) error {
	correlationID := uc.ids.NewID().String()

	processedData := WagerTransactionProcessedData{
		WalletID:      wallet.ID(),
		TransactionID: wagerTx.ID(),
		Kind:          string(wagerTx.Kind()),
	}
	processedData.Money.Amount = wagerTx.Money().String()
	processedData.Money.Currency = string(wagerTx.Money().Currency())

	processedPayload, err := NewEvent(uc.ids.NewID().String(), "WagerTransactionProcessed", string(wallet.ID()), correlationID, 1, processedData)
	if err != nil {
		return err
	}

	processedEvent, err := domain.NewOutboxEntry(domain.OutboxEventID(uc.ids.NewID().String()), string(wallet.ID()), "WagerTransactionProcessed", processedPayload)
	if err != nil {
		return err
	}

	if err := uc.outbox.Save(ctx, processedEvent); err != nil {
		return err
	}

	balanceData := WalletBalanceChangedData{
		WalletID:      wallet.ID(),
		TransactionID: wagerTx.ID(),
		Direction:     string(domain.DirectionDebit),
		BalanceBefore: entry.BalanceBefore().String(),
		BalanceAfter:  entry.BalanceAfter().String(),
		WalletVersion: wallet.Version(),
	}
	balanceData.Money.Amount = wagerTx.Money().String()
	balanceData.Money.Currency = string(wagerTx.Money().Currency())

	balancePayload, err := NewEvent(uc.ids.NewID().String(), "WalletBalanceChanged", string(wallet.ID()), correlationID, 1, balanceData)
	if err != nil {
		return err
	}

	balanceEvent, err := domain.NewOutboxEntry(domain.OutboxEventID(uc.ids.NewID().String()), string(wallet.ID()), "WalletBalanceChanged", balancePayload)
	if err != nil {
		return err
	}

	return uc.outbox.Save(ctx, balanceEvent)
}

func (uc *ProcessWagerTransaction) resultFromExisting(ctx context.Context, existing *domain.WagerTransaction) (*ProcessWagerTransactionResult, error) {
	entry, err := uc.walletLedgerRepo.FindByTransactionID(ctx, existing.ID())
	if err != nil && !errors.Is(err, ErrLedgerNotFound) {
		return nil, fmt.Errorf("process wager transaction: find ledger entry for replay: %w", err)
	}

	var balance domain.Money
	if entry != nil {
		balance = entry.BalanceAfter()
	} else {
		wallet, err := uc.walletRepo.FindByID(ctx, existing.WalletID())
		if err != nil {
			return nil, fmt.Errorf("process wager transaction: find wallet for replay: %w", err)
		}
		balance = wallet.Balance()
	}

	return &ProcessWagerTransactionResult{
		TransactionID:    existing.ID(),
		Status:           string(existing.Status()),
		Balance:          balance,
		IdempotentReplay: true,
	}, nil
}
