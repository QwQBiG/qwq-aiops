package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestRunReturnsListenerErrors(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := Run(context.Background(), listener.Addr().String()); err == nil {
		t.Fatal("occupied port was reported as a successful startup")
	}
}

func TestServerCancellationStopsActiveRequestsAndCollector(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, requestCancelled := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-r.Context().Done()
			close(requestCancelled)
			w.WriteHeader(http.StatusNoContent)
		}))
	}()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, err := http.Get("http://" + listener.Addr().String() + "/wait")
		if err == nil {
			response.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not accept a request")
	}
	cancel()
	select {
	case <-requestCancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("request context was not cancelled")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server or collector did not stop")
	}
	select {
	case <-requestDone:
	case <-time.After(2 * time.Second):
		t.Fatal("client request remained active")
	}
}
