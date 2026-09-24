package app

import (
	"context"
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type OpenWalletCommand struct {
	PlayerID       string
	InitialBalance domain.Money
}

type OpenWalletResult struct {
	WalletID string
	PlayerID string
	Balance  domain.Money
	Version  int64
}

type OpenWallet struct {
	uow          UnitOfWork
	wallets      WalletRepository
	transactions WagerTransactionRepository
	ledger       LedgerRepository
	outbox       OutboxRepository
	ids          IDGenerator
}

func NewOpenWallet(
	uow UnitOfWork,
	wallets WalletRepository,
	transactions WagerTransactionRepository,
	ledger LedgerRepository,
	outbox OutboxRepository,
	ids IDGenerator,
) *OpenWallet {
	return &OpenWallet{
		uow:          uow,
		wallets:      wallets,
		transactions: transactions,
		ledger:       ledger,
		outbox:       outbox,
		ids:          ids,
	}
}

func (uc *OpenWallet) Execute(ctx context.Context, cmd OpenWalletCommand) (*OpenWalletResult, error) {
	walletID := domain.WalletID(uc.ids.NewID().String())
	playerID := domain.PlayerID(cmd.PlayerID)

	wallet, err := domain.NewWallet(walletID, playerID, cmd.InitialBalance)
	if err != nil {
		return nil, fmt.Errorf("erro on open wallet: %w", err)
	}

	err = uc.uow.Execute(ctx, func(ctx context.Context) error {
		if err := uc.wallets.Create(ctx, wallet); err != nil {
			return err
		}

		if cmd.InitialBalance.IsZero() {
			return nil
		}

		return uc.recordOpeningCredit(ctx, wallet)
	})

	if err != nil {
		return nil, err
	}

	return &OpenWalletResult{
		WalletID: string(walletID),
		PlayerID: string(playerID),
		Balance:  wallet.Balance(),
		Version:  wallet.Version(),
	}, nil
}

func (uc *OpenWallet) recordOpeningCredit(ctx context.Context, wallet *domain.Wallet) error {
	transactionID := domain.TransactionID(uc.ids.NewID().String())

	openingTransaction, err := domain.NewOpeningTransaction(transactionID, wallet.ID(), wallet.PlayerID(), wallet.Balance())
	if err != nil {
		return err
	}

	if err := openingTransaction.MarkProcessed(""); err != nil {
		return err
	}

	if err := uc.transactions.Save(ctx, openingTransaction); err != nil {
		return err
	}

	zeroBalance, err := domain.Zero(wallet.Currency())
	if err != nil {
		return err
	}

	entryID := domain.LedgerEntryID(uc.ids.NewID().String())
	entry, err := domain.NewWalletLedgerEntry(
		entryID,
		wallet.ID(),
		transactionID,
		domain.DirectionCredit,
		wallet.Balance(),
		zeroBalance,
		wallet.Balance(),
	)
	if err != nil {
		return err
	}

	if err := uc.ledger.Append(ctx, entry); err != nil {
		return err
	}

	return uc.publishOpeningEvents(ctx, wallet, transactionID, entry)
}

func (uc *OpenWallet) publishOpeningEvents(ctx context.Context, wallet *domain.Wallet, transactionID domain.TransactionID, entry *domain.WalletLedgerEntry) error {
	correlationID := uc.ids.NewID().String()

	processedData := WagerTransactionProcessedData{
		TransactionID: string(transactionID),
		Kind:          string(domain.KindOpening),
		WalletID:      wallet.ToStringID(),
	}

	processedData.Money.Amount = wallet.Balance().String()
	processedData.Money.Currency = string(wallet.Currency())

	processedPayload, err := NewEvent(uc.ids.NewID().String(), "WagerTransactionProcessed", wallet.ToStringID(), correlationID, 1, processedData)
	if err != nil {
		return err
	}

	processedEvent, err := domain.NewOutboxEntry(domain.OutboxEventID(uc.ids.NewID().String()), wallet.ToStringID(), "WagerTransactionProcessed", processedPayload)
	if err != nil {
		return err
	}

	if err := uc.outbox.Save(ctx, processedEvent); err != nil {
		return err
	}

	balanceChangedData := WalletBalanceChangedData{
		WalletID:      wallet.ToStringID(),
		TransactionID: string(transactionID),
		Direction:     string(domain.DirectionCredit),
		BalanceBefore: entry.BalanceBefore().String(),
		BalanceAfter:  entry.BalanceAfter().String(),
		WalletVersion: wallet.Version(),
	}

	balanceChangedData.Money.Amount = wallet.Balance().String()
	balanceChangedData.Money.Currency = string(wallet.Currency())

	balancePayload, err := NewEvent(uc.ids.NewID().String(), "WalletBalanceChanged", wallet.ToStringID(), correlationID, 1, balanceChangedData)
	if err != nil {
		return err
	}

	balanceEvent, err := domain.NewOutboxEntry(domain.OutboxEventID(uc.ids.NewID().String()), wallet.ToStringID(), "WalletBalanceChanged", balancePayload)
	if err != nil {
		return err
	}
	return uc.outbox.Save(ctx, balanceEvent)
}
