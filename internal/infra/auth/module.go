package auth

import (
	"context"
	"fmt"

	"github.com/cleidison-barradas/betwallet.api/internal/infra/config"
	"go.uber.org/fx"
)

func newMiddleware(cfg config.Config) (*Middleware, error) {
	m, err := New(context.Background(), cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to init oidc provider: %w", err)
	}

	return m, nil
}

var Module = fx.Module("auth", fx.Provide(newMiddleware))
