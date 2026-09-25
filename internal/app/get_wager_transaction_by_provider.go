package app

import (
	"context"
)

type GetWagerTransactionByProviderCommand struct {
	ProviderID   string
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
		Id:                    transaction.ID(),
		Kind:                  string(transaction.Kind()),
		Status:                string(transaction.Status()),
		WalletID:              transaction.WalletID(),
		PlayerID:              transaction.PlayerID(),
		Money:                 transaction.Money(),
		ProviderID:            *transaction.ProviderID(),
		ExternalTransactionID: *transaction.ExternalTransactionID(),
		IdempotencyKey:        *transaction.IdempotencyKey(),
		PayloadHash:           *transaction.PayloadHash(),
		RoundID:               *transaction.RoundID(),
		GameID:                *transaction.GameID(),
		ReferenceExternalTxID: *transaction.ReferenceExternalTransactionID(),
		ResolvedReferenceID:   *transaction.ResolvedReferenceID(),
		FailureCode:           *transaction.FailureCode(),
		CreatedAt:             transaction.CreatedAt(),
		UpdatedAt:             transaction.UpdatedAt(),
	}, nil
}
