package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey string

const (
	providerIDKey contextKey = "auth.providerId"
	internalKey   contextKey = "auth.internal"
)

type claims struct {
	ProviderID string `json:"provider_id"`
	Internal   bool   `json:"internal"`
}

type Middleware struct {
	verifier *oidc.IDTokenVerifier
}

func New(ctx context.Context, issuerURL string) (*Middleware, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}
	return &Middleware{verifier: provider.Verifier(&oidc.Config{SkipClientIDCheck: true})}, nil
}

func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		idToken, err := m.verifier.Verify(r.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		var c claims
		if err := idToken.Claims(&c); err != nil {
			http.Error(w, "invalid claims", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), providerIDKey, c.ProviderID)
		ctx = context.WithValue(ctx, internalKey, c.Internal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) RequireProvider(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ProviderIDFromContext(r.Context()) == "" {
			http.Error(w, "token não possui provider_id", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequireInternal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !InternalFromContext(r.Context()) {
			http.Error(w, "operação restrita ao serviço interno", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ProviderIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(providerIDKey).(string)
	return v
}

func InternalFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(internalKey).(bool)
	return v
}
