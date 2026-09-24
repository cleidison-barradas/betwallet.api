package idgen

import (
	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"go.uber.org/fx"
)

var Module = fx.Module("idgen",
	fx.Provide(
		fx.Annotate(
			New,
			fx.As(new(app.IDGenerator)),
		),
	),
)
