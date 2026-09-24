package router

import "go.uber.org/fx"

var Module = fx.Module("router",
	fx.Provide(
		fx.Annotate(
			NewWalletHandler,
			fx.As(new(Route)),
			fx.ResultTags(`group:"routes"`),
		),
		fx.Annotate(
			NewWalletLedgerHandler,
			fx.As(new(Route)),
			fx.ResultTags(`group:"routes"`),
		),
		fx.Annotate(
			NewWagerTransactionHandler,
			fx.As(new(Route)),
			fx.ResultTags(`group:"routes"`),
		),
	),
	fx.Provide(New),
)
