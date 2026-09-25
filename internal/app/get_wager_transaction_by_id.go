package app

import (
	"context"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type GetWagerTransactionByIDCommand struct {
	TransactionID string
	ProviderID    string
}

type GetWagerTransaction struct {
	repo WagerTransactionRepository
}

type WagerTransactionResult struct {
	Id                    string       `json:"id"`
	Kind                  string       `json:"kind"`
	Status                string       `json:"status"`
	WalletID              string       `json:"walletId"`
	PlayerID              string       `json:"playerId"`
	Money                 domain.Money `json:"money"`
	ProviderID            string       `json:"providerId"`
	ExternalTransactionID string       `json:"externalTransactionId"`
	IdempotencyKey        string       `json:"idempotencyKey"`
	PayloadHash           string       `json:"payloadHash"`
	RoundID               string       `json:"roundId"`
	GameID                string       `json:"gameId"`
	ReferenceExternalTxID string       `json:"referenceExternalTransactionId"`
	ResolvedReferenceID   string       `json:"resolvedReferenceId"`
	FailureCode           string       `json:"failureCode"`
	CreatedAt             time.Time    `json:"createdAt"`
	UpdatedAt             time.Time    `json:"updatedAt"`
}

func NewGetWagerTransaction(repo WagerTransactionRepository) *GetWagerTransaction {
	return &GetWagerTransaction{repo: repo}
}

func (uc *GetWagerTransaction) Execute(ctx context.Context, cmd GetWagerTransactionByIDCommand) (*WagerTransactionResult, error) {
	transaction, err := uc.repo.FindByID(ctx, cmd.TransactionID, cmd.ProviderID)
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
