package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// GatewayServer forwards console requests to the local Web listener.
type GatewayServer struct {
	gateway          *Gateway
	server           *http.Server
	configurationErr error
}

func NewGatewayServer(address string) *GatewayServer {
	g := NewGateway()
	g.AddMiddleware(LoggingMiddleware())
	g.AddMiddleware(AuthMiddleware())
	g.AddMiddleware(RateLimitMiddleware(100))
	gs := &GatewayServer{
		gateway: g,
		server: &http.Server{
			Addr: address, Handler: g,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
	gs.configurationErr = gs.registerDefaultServices()
	return gs
}

func (gs *GatewayServer) registerDefaultServices() error {
	port := os.Getenv("WEB_UI_PORT")
	if port == "" {
		port = "8899"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("WEB_UI_PORT must be a number between 1 and 65535")
	}
	baseURL := "http://" + net.JoinHostPort("127.0.0.1", port)
	if err := gs.gateway.RegisterService("web-ui", baseURL, "", "1.0"); err != nil {
		return err
	}
	gs.gateway.AddRoute("/", "web-ui", nil)
	return nil
}

func (gs *GatewayServer) Start() error {
	return gs.Run(context.Background())
}

// Run serves until cancellation or a listener failure. The console owns its
// own routes and health response; no fictitious services are registered here.
func (gs *GatewayServer) Run(ctx context.Context) error {
	if gs.configurationErr != nil {
		return gs.configurationErr
	}
	if ctx.Err() != nil {
		return nil
	}
	finished := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			gs.Stop()
		case <-finished:
		}
	}()
	err := gs.server.ListenAndServe()
	close(finished)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("gateway listener failed: %w", err)
	}
	return nil
}

func (gs *GatewayServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return gs.server.Shutdown(ctx)
}

func (gs *GatewayServer) GetGateway() *Gateway { return gs.gateway }

func (gs *GatewayServer) RegisterService(name, address, health, version string) error {
	return gs.gateway.RegisterService(name, address, health, version)
}

func (gs *GatewayServer) AddRoute(path, serviceName string, methods []string) {
	gs.gateway.AddRoute(path, serviceName, methods)
}

func (gs *GatewayServer) GetServiceStatus() map[string]*ServiceInfo {
	return gs.gateway.ListServices()
}
