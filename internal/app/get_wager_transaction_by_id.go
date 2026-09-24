package app

import (
	"context"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type GetWagerTransactionByIDCommand struct {
	TransactionID domain.TransactionID
}

type GetWagerTransaction struct {
	repo WagerTransactionRepository
}

type WagerTransactionResult struct {
	Id                    string                `json:"id"`
	Kind                  string                `json:"kind"`
	Status                string                `json:"status"`
	WalletID              string                `json:"walletId"`
	PlayerID              string                `json:"playerId"`
	Money                 domain.Money          `json:"money"`
	ProviderID            *domain.ProviderID    `json:"providerId"`
	ExternalTransactionID string                `json:"externalTransactionId"`
	IdempotencyKey        string                `json:"idempotencyKey"`
	PayloadHash           string                `json:"payloadHash"`
	RoundID               *domain.RoundID       `json:"roundId"`
	GameID                *domain.GameID        `json:"gameId"`
	ReferenceExternalTxID string                `json:"referenceExternalTransactionId"`
	ResolvedReferenceID   *domain.TransactionID `json:"resolvedReferenceId"`
	FailureCode           string                `json:"failureCode"`
	CreatedAt             time.Time             `json:"createdAt"`
	UpdatedAt             time.Time             `json:"updatedAt"`
}

func NewGetWagerTransaction(repo WagerTransactionRepository) *GetWagerTransaction {
	return &GetWagerTransaction{repo: repo}
}

func (uc *GetWagerTransaction) Execute(ctx context.Context, command GetWagerTransactionByIDCommand) (*WagerTransactionResult, error) {
	transaction, err := uc.repo.FindByID(ctx, command.TransactionID)
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
