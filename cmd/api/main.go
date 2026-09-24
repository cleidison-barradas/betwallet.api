package main

import (
	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/config"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/idgen"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/postgres"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/router"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/server"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		config.Module,
		idgen.Module,
		postgres.Module,
		fx.Provide(app.NewOpenWallet),
		server.Module,
		router.Module,
	).Run()
}
