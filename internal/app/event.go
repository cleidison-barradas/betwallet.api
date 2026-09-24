package app

import (
	"encoding/json"
	"time"
)

type Event struct {
	EventID       string      `json:"eventId"`
	EventType     string      `json:"eventType"`
	AggregateID   string      `json:"aggregateId"`
	CorrelationID string      `json:"correlationId"`
	CausationID   string      `json:"causationId,omitempty"`
	OccurredAt    string      `json:"occurredAt"`
	Version       int         `json:"version"`
	Data          interface{} `json:"data"`
}

func NewEvent(
	eventID string,
	eventType string,
	aggregateID string,
	correlationID string,
	version int,
	data interface{},
) ([]byte, error) {
	event := Event{
		EventID:       eventID,
		EventType:     eventType,
		AggregateID:   aggregateID,
		CorrelationID: correlationID,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
		Version:       version,
		Data:          data,
	}
	return json.Marshal(event)
}

type WagerTransactionProcessedData struct {
	TransactionID string `json:"transactionId"`
	Kind          string `json:"kind"`
	WalletID      string `json:"walletId"`
	Money         struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"money"`
}

type WalletBalanceChangedData struct {
	WalletID      string `json:"walletId"`
	TransactionID string `json:"transactionId"`
	Direction     string `json:"direction"`
	Money         struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"money"`
	BalanceBefore string `json:"balanceBefore"`
	BalanceAfter  string `json:"balanceAfter"`
	WalletVersion int64  `json:"walletVersion"`
}
