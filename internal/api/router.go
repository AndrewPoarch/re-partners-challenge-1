package api

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter registers API routes and optional static UI; uiFS nil skips UI (tests).
func NewRouter(h *Handlers, uiFS fs.FS) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware)

	r.Get("/healthz", Health)

	r.Route("/api", func(r chi.Router) {
		r.Get("/pack-sizes", h.GetPackSizes)
		r.Put("/pack-sizes", h.PutPackSizes)
		r.Post("/calculate", h.Calculate)
	})

	if uiFS != nil {
		fileServer := http.FileServer(http.FS(uiFS))
		r.Handle("/*", fileServer)
	}

	return r
}

// corsMiddleware allows browser clients from other origins (useful for local dev tooling).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
