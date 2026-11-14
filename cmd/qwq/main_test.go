package main

import (
	"context"
	"net"
	"testing"

	"github.com/spf13/cobra"
)

func TestWebModePropagatesBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PORT", port)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := &cobra.Command{}
	command.SetContext(ctx)
	if err := runWebMode(command, nil); err == nil {
		t.Fatal("web command swallowed the listener error")
	}
}
