package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/handlers"
	"github.com/pavilion-trust/core-broker/internal/middleware"
	"github.com/pavilion-trust/core-broker/internal/services"
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

	// Create credential signing service
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate RSA key: %v", err))
	}
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate ECDSA key: %v", err))
	}
	signingService := services.NewCredentialSigningService(rsaKey, ecdsaKey, "key-1", cfg.Issuer)
	credentialHandler := handlers.NewCredentialHandler(cfg, signingService)

	// Create policy storage and handler
	policyStorage, err := services.NewPolicyStorage(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create policy storage: %v", err))
	}
	policyHandler := handlers.NewPolicyHandler(cfg, policyStorage)

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.Authentication(cfg))

	// Verification endpoint (requires 'rp' role)
	verificationRouter := apiRouter.PathPrefix("/verify").Subrouter()
	verificationRouter.Use(middleware.RequireRole("rp"))
	verificationRouter.Use(middleware.ValidationMiddleware)
	verificationRouter.HandleFunc("", verificationHandler.HandleVerification).Methods("POST")

	// Policy endpoints (requires 'admin' role)
	policyRouter := apiRouter.PathPrefix("/policies").Subrouter()
	policyRouter.Use(middleware.RequireRole("admin"))

	// Policy CRUD operations
	policyRouter.HandleFunc("", policyHandler.HandleCreatePolicy).Methods("POST")
	policyRouter.HandleFunc("", policyHandler.HandleListPolicies).Methods("GET")
	policyRouter.HandleFunc("/{id}", policyHandler.HandleGetPolicy).Methods("GET")
	policyRouter.HandleFunc("/{id}", policyHandler.HandleUpdatePolicy).Methods("PUT")
	policyRouter.HandleFunc("/{id}", policyHandler.HandleDeletePolicy).Methods("DELETE")

	// Policy evaluation
	policyRouter.HandleFunc("/evaluate", policyHandler.HandleEvaluatePolicy).Methods("POST")

	// Policy templates
	policyRouter.HandleFunc("/templates", policyHandler.HandleCreateTemplate).Methods("POST")
	policyRouter.HandleFunc("/templates", policyHandler.HandleListTemplates).Methods("GET")
	policyRouter.HandleFunc("/templates/{id}", policyHandler.HandleGetTemplate).Methods("GET")

	// Audit endpoints
	policyRouter.HandleFunc("/audit", policyHandler.HandleGetAuditLogs).Methods("GET")
	policyRouter.HandleFunc("/audit/{request_id}", policyHandler.HandleGetAuditLog).Methods("GET")
	policyRouter.HandleFunc("/audit/stats", policyHandler.HandleGetAuditStats).Methods("GET")

	// Health endpoint
	policyRouter.HandleFunc("/health", policyHandler.HandleHealth).Methods("GET")

	// Credential endpoints (requires 'admin' role)
	credentialRouter := apiRouter.PathPrefix("/credentials").Subrouter()
	credentialRouter.Use(middleware.RequireRole("admin"))

	// Credential CRUD operations
	credentialRouter.HandleFunc("", credentialHandler.HandleCreateCredential).Methods("POST")
	credentialRouter.HandleFunc("", credentialHandler.HandleListCredentials).Methods("GET")
	credentialRouter.HandleFunc("/{id}", credentialHandler.HandleGetCredential).Methods("GET")
	credentialRouter.HandleFunc("/{id}/revoke", credentialHandler.HandleRevokeCredential).Methods("POST")
	credentialRouter.HandleFunc("/{id}/status", credentialHandler.HandleGetCredentialStatus).Methods("GET")
	credentialRouter.HandleFunc("/{id}/verify", credentialHandler.HandleVerifyCredential).Methods("POST")

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

// NewAPIGateway creates a new API Gateway server with TLS termination and routing
func NewAPIGateway(cfg *config.Config) *Server {
	// Create router
	router := mux.NewRouter()

	// Add API Gateway middleware
	router.Use(middleware.CORS)
	router.Use(middleware.Logging)
	router.Use(middleware.RequestID)
	router.Use(middleware.Recovery)
	router.Use(middleware.HTTPSRedirect)
	router.Use(middleware.SecurityHeaders)

	// Create API Gateway handlers
	gatewayHandler := handlers.NewAPIGatewayHandler(cfg)
	healthHandler := handlers.NewHealthHandler(cfg)

	// API routes with authentication
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.Authentication(cfg))
	apiRouter.Use(middleware.RateLimiting(cfg))

	// Route all API requests to Core Broker
	apiRouter.PathPrefix("").HandlerFunc(gatewayHandler.HandleAPIRequest)

	// Health check endpoint (no authentication required)
	router.HandleFunc("/health", healthHandler.HandleHealth).Methods("GET")

	// Create HTTP server with TLS
	srv := &http.Server{
		Addr:         ":" + cfg.APIGatewayPort,
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
