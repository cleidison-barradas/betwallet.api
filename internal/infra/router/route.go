package router

import "net/http"

type Route interface {
	Register(mux *http.ServeMux)
}
