package router

import (
	"database/sql"
	"net/http"

	"github.com/mohit838/mtz-go-migrator/test/internal/response"
)

type healthDependencies struct {
	db           *sql.DB
	serviceName  string
	serviceTitle string
}

func registerHealthRoutes(r interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}, deps healthDependencies) {
	// GET / — basic liveness probe
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, http.StatusOK, deps.serviceTitle+" is running", map[string]string{
			"service": deps.serviceName,
			"status":  "ok",
		})
	})

	// GET /health — service health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, http.StatusOK, "service is healthy", map[string]string{
			"service": deps.serviceName,
			"status":  "ok",
		})
	})

	// GET /ready — readiness probe (checks Postgres)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.db.PingContext(r.Context()); err != nil {
			response.JSON(w, http.StatusServiceUnavailable, response.Body{
				Success: false,
				Message: "postgres not ready",
				Data:    map[string]string{"error": err.Error()},
			})
			return
		}
		response.Success(w, http.StatusOK, "service is ready", map[string]string{
			"service":  deps.serviceName,
			"postgres": "ok",
		})
	})

	// GET /health/postgres — explicit Postgres health check
	r.Get("/health/postgres", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.db.PingContext(r.Context()); err != nil {
			response.Error(w, http.StatusServiceUnavailable, "postgres health check failed", err.Error())
			return
		}
		response.Success(w, http.StatusOK, "postgres connected", map[string]string{"status": "ok"})
	})
}
