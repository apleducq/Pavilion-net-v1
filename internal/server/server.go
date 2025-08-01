package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/handlers"
	"github.com/pavilion-trust/core-broker/internal/middleware"
)

// Server represents the HTTP server
type Server struct {
	*http.Server
	config *config.Config
}

// New creates a new HTTP server with all routes and middleware
func New(cfg *config.Config) *Server {
	// Create router
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.CORS)
	router.Use(middleware.Logging)
	router.Use(middleware.RequestID)
	router.Use(middleware.Recovery)

	// Create handlers
	verificationHandler := handlers.NewVerificationHandler(cfg)
	healthHandler := handlers.NewHealthHandler(cfg)

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.Authentication(cfg))

	// Verification endpoint (requires 'rp' role)
	verificationRouter := apiRouter.PathPrefix("/verify").Subrouter()
	verificationRouter.Use(middleware.RequireRole("rp"))
	verificationRouter.HandleFunc("", verificationHandler.HandleVerification).Methods("POST")

	// Health check endpoint (no authentication required)
	router.HandleFunc("/health", healthHandler.HandleHealth).Methods("GET")

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		Server: srv,
		config: cfg,
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
} 