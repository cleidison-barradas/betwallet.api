package app

import (
	"context"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type GetWagerTransactionByProviderCommand struct {
	ProviderID   domain.ProviderID
	ExternalTxID string
}

type GetWagerTransactionByProvider struct {
	repo WagerTransactionRepository
}

func NewGetWagerTransactionByProvider(repo WagerTransactionRepository) *GetWagerTransactionByProvider {
	return &GetWagerTransactionByProvider{repo: repo}
}

func (uc *GetWagerTransactionByProvider) Execute(ctx context.Context, command GetWagerTransactionByProviderCommand) (*WagerTransactionResult, error) {
	transaction, err := uc.repo.FindByProvider(ctx, FindWagerTransactionByProviderParams{
		ProviderID:   command.ProviderID,
		ExternalTxID: command.ExternalTxID,
	})
	if err != nil {
		return nil, err
	}

	return &WagerTransactionResult{
		Id:                    string(transaction.ID()),
		Kind:                  string(transaction.Kind()),
		Status:                string(transaction.Status()),
		WalletID:              string(transaction.WalletID()),
		PlayerID:              string(transaction.PlayerID()),
		Money:                 transaction.Money(),
		ProviderID:            transaction.ProviderID(),
		ExternalTransactionID: transaction.ExternalTransactionID(),
		IdempotencyKey:        transaction.IdempotencyKey(),
		PayloadHash:           transaction.PayloadHash(),
		RoundID:               transaction.RoundID(),
		GameID:                transaction.GameID(),
		ReferenceExternalTxID: transaction.ReferenceExternalTransactionID(),
		ResolvedReferenceID:   transaction.ResolvedReferenceID(),
		FailureCode:           transaction.FailureCode(),
		CreatedAt:             transaction.CreatedAt(),
		UpdatedAt:             transaction.UpdatedAt(),
	}, nil
}
