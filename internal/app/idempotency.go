package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type canonicalPayload struct {
	ExternalTransactionID          string `json:"externalTransactionId"`
	GameID                         string `json:"gameId"`
	Kind                           string `json:"kind"`
	MoneyAmount                    string `json:"moneyAmount"`
	MoneyCurrency                  string `json:"moneyCurrency"`
	PlayerID                       string `json:"playerId"`
	ProviderID                     string `json:"providerId"`
	ReferenceExternalTransactionID string `json:"referenceExternalTransactionId,omitempty"`
	RoundID                        string `json:"roundId"`
	WalletID                       string `json:"walletId"`
}

func HashPayload(cmd ProcessWagerTransactionCommand) (string, error) {
	canonical := canonicalPayload{
		ExternalTransactionID:          cmd.ExternalTransactionID,
		GameID:                         cmd.GameID,
		Kind:                           string(cmd.Kind),
		MoneyAmount:                    cmd.Money.String(),
		MoneyCurrency:                  string(cmd.Money.Currency()),
		PlayerID:                       cmd.PlayerID,
		ProviderID:                     cmd.ProviderID,
		ReferenceExternalTransactionID: cmd.ReferenceExternalTransactionID,
		RoundID:                        cmd.RoundID,
		WalletID:                       cmd.WalletID,
	}

	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
