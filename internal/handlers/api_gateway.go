package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

// APIGatewayHandler handles API Gateway requests
type APIGatewayHandler struct {
	config *config.Config
	proxy  *httputil.ReverseProxy
}

// NewAPIGatewayHandler creates a new API Gateway handler
func NewAPIGatewayHandler(cfg *config.Config) *APIGatewayHandler {
	// Parse the Core Broker URL
	targetURL, err := url.Parse(cfg.CoreBrokerURL)
	if err != nil {
		panic("Invalid Core Broker URL: " + err.Error())
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the proxy director
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Add any additional headers or modifications here
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", req.Host)
	}

	// Customize the proxy transport
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}

	return &APIGatewayHandler{
		config: cfg,
		proxy:  proxy,
	}
}

// HandleAPIRequest handles all API requests by proxying them to the Core Broker
func (h *APIGatewayHandler) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Add request ID to context if not present
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		ctx = context.WithValue(ctx, "requestID", requestID)
	}

	// Create a new request with the updated context
	r = r.WithContext(ctx)

	// Proxy the request to Core Broker
	h.proxy.ServeHTTP(w, r)
}

// HandleHealth handles health check requests
func (h *APIGatewayHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// Check if Core Broker is reachable
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	coreBrokerHealthURL := h.config.CoreBrokerURL + "/health"
	resp, err := client.Get(coreBrokerHealthURL)
	if err != nil {
		http.Error(w, "Core Broker unreachable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Read and forward the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read Core Broker response", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
} 