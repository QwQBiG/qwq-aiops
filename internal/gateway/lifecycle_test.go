package gateway

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestGatewayCancellationStopsActiveRequests(t *testing.T) {
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reserved.Addr().String()
	reserved.Close()
	server := NewGatewayServer(address)
	started, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(cancelled)
		case <-release:
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		cancelRequest()
		close(release)
		server.server.Close()
	})
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("gateway did not listen")
		}
		time.Sleep(10 * time.Millisecond)
	}
	go func() {
		request, _ := http.NewRequestWithContext(requestCtx, http.MethodGet, "http://"+address, nil)
		response, err := client.Do(request)
		if err == nil {
			response.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach the gateway")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("gateway cancellation left the active request running")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("gateway did not finish shutdown")
	}
}
