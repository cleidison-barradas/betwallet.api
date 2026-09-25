package app

import (
	"context"

	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type GetWalletByWalletIDCommand struct {
	WalletID string
}

type GetWalletByWalletIDResult struct {
	WalletID string
	PlayerID string
	Balance  domain.Money
	Version  int64
}

type GetWalletByWalletID struct {
	wallets WalletRepository
}

func NewGetWalletByWalletID(wallets WalletRepository) *GetWalletByWalletID {
	return &GetWalletByWalletID{
		wallets: wallets,
	}
}

func (uc *GetWalletByWalletID) Execute(ctx context.Context, cmd GetWalletByWalletIDCommand) (*GetWalletByWalletIDResult, error) {
	wallet, err := uc.wallets.FindByID(ctx, cmd.WalletID)
	if err != nil {
		return nil, err
	}

	return &GetWalletByWalletIDResult{
		WalletID: wallet.ID(),
		PlayerID: wallet.PlayerID(),
		Balance:  wallet.Balance(),
		Version:  wallet.Version(),
	}, nil
}
