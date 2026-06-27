package router

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mohit838/learn-go-with-project/internal/constants"
)

// NewRouter builds and returns the HTTP handler for the migrator test service.
// It only exposes health/readiness probes — no auth, no business logic.
// This service exists solely to exercise the libs/migrator library.
func NewRouter(db *sql.DB) http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	registerHealthRoutes(r, healthDependencies{
		db:           db,
		serviceName:  constants.ServiceName,
		serviceTitle: constants.ServiceTitle,
	})

	return r
}
