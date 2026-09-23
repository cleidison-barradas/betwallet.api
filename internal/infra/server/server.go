package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/infra/config"
	"go.uber.org/fx"
)

type Server struct {
	httpServer *http.Server
}

func New(cfg config.Config, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			IdleTimeout:  cfg.IdleTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
}

func (s *Server) Run() error {
	slog.Info("Http server starting", "Addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Http server shutting down gracefully")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(shutdownCtx)
}

var Module = fx.Module("server",
	fx.Provide(New),
	fx.Invoke(registerHooks),
)

func registerHooks(lifecycle fx.Lifecycle, server *Server) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.Run(); err != nil && err != http.ErrServerClosed {
					slog.Error("Failed to start server", "err", err)
				}
			}()
			return nil
		},
		OnStop: server.Shutdown,
	})
}
