package router

import (
	"encoding/json"
	"net/http"

	"go.uber.org/fx"
)

type routerParams struct {
	fx.In
	Routes []Route `group:"routes"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

func New(p routerParams) http.Handler {
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

	for _, route := range p.Routes {
		route.Register(mux)
	}

	var h http.Handler = mux

	return h
}
