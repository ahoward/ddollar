package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/drawohara/ddollar/src/tokens"
)

// Server represents the ddollar proxy server
type Server struct {
	tokenPool  *tokens.Pool
	httpServer *http.Server
	port       int
}

// NewServer creates a new proxy server
func NewServer(tokenPool *tokens.Pool, port int) *Server {
	return &Server{
		tokenPool: tokenPool,
		port:      port,
	}
}

// Start starts the HTTPS proxy server
func (s *Server) Start() error {
	// Generate or load certificate
	certPath, keyPath, err := GenerateCert()
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	// Load TLS certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Create reverse proxy handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleRequest(w, r)
	})

	// Configure HTTPS server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Starting ddollar proxy on port %d...", s.port)

	// Start server
	if err := s.httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Stop gracefully stops the proxy server
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Stopping ddollar proxy...")
	return s.httpServer.Shutdown(ctx)
}

// handleRequest handles incoming requests and injects tokens
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Get the target domain (hostname from the request)
	domain := r.Host

	// Log the request
	log.Printf("[%s] %s %s", r.Method, domain, r.URL.Path)

	// Get token for this domain
	token, provider, err := s.tokenPool.GetToken(domain)
	if err != nil {
		log.Printf("No tokens available for %s: %v", domain, err)
		http.Error(w, "No API tokens configured for this provider", http.StatusServiceUnavailable)
		return
	}

	// Create target URL (preserve original scheme and path)
	targetURL := &url.URL{
		Scheme:   "https",
		Host:     domain,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify request to inject token
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Set the correct Host header
		req.Host = domain
		req.URL.Scheme = "https"
		req.URL.Host = domain

		// Remove any existing auth headers (we'll replace with our token)
		req.Header.Del("Authorization")
		req.Header.Del("x-api-key")
		req.Header.Del("x-goog-api-key")

		// Inject the token using provider-specific auth header
		authValue := provider.AuthPrefix + token
		req.Header.Set(provider.AuthHeader, authValue)

		log.Printf("Injected token for %s (provider: %s)", domain, provider.Name)
	}

	// Proxy the request
	proxy.ServeHTTP(w, r)
}
