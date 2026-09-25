package main

import (
	"log"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/auth"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/config"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/idgen"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/postgres"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/router"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/server"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}

	fx.New(
		config.Module,
		idgen.Module,
		postgres.Module,
		auth.Module,
		fx.Provide(app.NewOpenWallet),
		fx.Provide(app.NewGetWalletByWalletID),
		fx.Provide(app.NewGetWalletLedgerByWalletID),
		fx.Provide(app.NewGetWagerTransaction),
		fx.Provide(app.NewGetWagerTransactionByProvider),
		fx.Provide(app.NewProcessWagerTransaction),
		server.Module,
		router.Module,
	).Run()
}
