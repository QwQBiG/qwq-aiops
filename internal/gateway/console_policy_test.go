package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"testing"
	"time"
)

func TestGatewayAuthenticationConfiguration(t *testing.T) {
	old := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = old })
	handler := AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	for _, tc := range []struct {
		user, password string
		want           int
	}{{"", "", 204}, {"operator", "", 503}, {"", "password", 503}, {"operator", "password", 401}} {
		config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = tc.user, tc.password
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/test", nil))
		if w.Code != tc.want {
			t.Fatalf("credential configuration returned %d, want %d", w.Code, tc.want)
		}
	}
}

func TestGatewayUsesLongestRoutePrefix(t *testing.T) {
	g := NewGateway()
	g.AddRoute("/", "console", nil)
	g.AddRoute("/api/", "api", nil)
	g.AddRoute("/api/special/", "special", nil)
	for i := 0; i < 100; i++ {
		if got := g.findRoute("/api/special/resource"); got == nil || got.ServiceName != "special" {
			t.Fatalf("longest route lost to another prefix: %+v", got)
		}
	}
}

func TestGatewayListenerStopsOnCancellationAndReportsBindFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer upstream.Close()
	_, upstreamPort, _ := net.SplitHostPort(upstream.Listener.Addr().String())
	t.Setenv("WEB_UI_PORT", upstreamPort)
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewGatewayServer(occupied.Addr().String()).Run(context.Background()); err == nil {
		t.Fatal("occupied listener address was reported as a successful start")
	}
	address := occupied.Addr().String()
	occupied.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- NewGatewayServer(address).Run(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		response, err := http.Get("http://" + address + "/health")
		if err == nil {
			response.Body.Close()
			ready = response.StatusCode == 200
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("gateway listener did not start")
	}
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("gateway did not stop after cancellation")
	}
}
