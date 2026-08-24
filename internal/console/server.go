package console

import (
	"context"
	"net"
	"net/http"
	"time"
)

// HTTPServer wraps the chi router with graceful shutdown support.
type HTTPServer struct {
	server *http.Server
}

// NewHTTPServer builds an HTTP server on the given address.
func NewHTTPServer(addr string, api *Server) *HTTPServer {
	server := &http.Server{
		Addr:              addr,
		Handler:           api.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return &HTTPServer{server: server}
}

// Addr returns the bound listener address.
func (h *HTTPServer) Addr() string {
	return h.server.Addr
}

// Listen starts the HTTP server and blocks until shutdown.
func (h *HTTPServer) Listen(ctx context.Context) error {
	listener, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.server.Shutdown(shutdownCtx)
	}()
	err = h.server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// URL returns the base URL of the server.
func (h *HTTPServer) URL() string {
	return "http://" + h.server.Addr
}
