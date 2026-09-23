package router

import (
	"encoding/json"
	"net/http"

	"github.com/cleidison-barradas/betwallet.api/internal/infra/config"
	"go.uber.org/fx"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func New(cfg config.Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{Status: "OK"})
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{Status: "OK"})
	})

	var h http.Handler = mux

	return h
}

var Module = fx.Module("router", fx.Provide(New))
