package main

import (
	"github.com/cleidison-barradas/betwallet.api/internal/infra/config"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/router"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/server"
	"go.uber.org/fx"
)

func main() {

	fx.New(
		config.Module,
		server.Module,
		router.Module,
	).Run()
}
