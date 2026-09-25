package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type GetWalletLedgerByWalletIDCommand struct {
	WalletID string
	Limit    int
	Cursor   string
}

type WalletLedgerEntryResponse struct {
	ID            string    `json:"id"`
	WalletID      string    `json:"wallet_id"`
	TransactionID string    `json:"transaction_id"`
	Direction     string    `json:"direction"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	BalanceBefore int64     `json:"balance_before_minor_units"`
	BalanceAfter  int64     `json:"balance_after_minor_units"`
	CreatedAt     time.Time `json:"created_at"`
}

type WalletLedgerResponse struct {
	Data       []WalletLedgerEntryResponse `json:"data"`
	NextCursor string                      `json:"next_cursor,omitempty"`
	HasNext    bool                        `json:"has_next"`
}

type GetWalletLedgerByWalletID struct {
	ledger WalletLedgerRepository
}

func NewWalletLedgerEntryResponse(e domain.WalletLedgerEntry) WalletLedgerEntryResponse {
	return WalletLedgerEntryResponse{
		ID:            string(e.ID()),
		WalletID:      string(e.WalletID()),
		TransactionID: string(e.TransactionID()),
		Direction:     string(e.Direction()),
		Amount:        e.Amount().MinorUnits(),
		Currency:      string(e.Amount().Currency()),
		BalanceBefore: e.BalanceBefore().MinorUnits(),
		BalanceAfter:  e.BalanceAfter().MinorUnits(),
		CreatedAt:     e.CreatedAt(),
	}
}

func NewGetWalletLedgerByWalletID(ledger WalletLedgerRepository) *GetWalletLedgerByWalletID {
	return &GetWalletLedgerByWalletID{
		ledger: ledger,
	}
}

func (uc *GetWalletLedgerByWalletID) Execute(ctx context.Context, cmd GetWalletLedgerByWalletIDCommand) (*WalletLedgerResponse, error) {

	cursorDecoded, err := decodeCursor(cmd.Cursor)
	if err != nil {
		return nil, fmt.Errorf("error on decode cursor: %w", err)
	}

	result, err := uc.ledger.ListWalletLedger(ctx, ListWalletLedgerParams{
		WalletID: cmd.WalletID,
		Cursor:   cursorDecoded,
		Limit:    cmd.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("error on listing ledgers: %w", err)
	}

	response := make([]WalletLedgerEntryResponse, len(result.Entries))

	for i, e := range result.Entries {
		response[i] = NewWalletLedgerEntryResponse(e)
	}

	nextEncodedCursor, err := encodeCursor(result.Cursor)
	if err != nil {
		return nil, fmt.Errorf("error on encode cursor: %w", err)
	}

	return &WalletLedgerResponse{
		Data:       response,
		HasNext:    result.HasNext,
		NextCursor: nextEncodedCursor,
	}, nil
}

func decodeCursor(value string) (*WalletLedgerCursor, error) {

	if value == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	var cursor WalletLedgerCursor

	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}

func encodeCursor(cursor *WalletLedgerCursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}
