package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"qwq/internal/config"
	"testing"

	"github.com/spf13/cobra"
)

func TestExplicitFlagsOverrideFileAndEnvironment(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	config.GlobalConfig = config.Config{}
	for _, key := range []string{"AI_PROVIDER", "TELEGRAM_TOKEN", "TELEGRAM_CHAT_ID"} {
		t.Setenv(key, "")
	}
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("WEB_USER", "env-user")
	t.Setenv("WEB_PASSWORD", "env-password")
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"web_user":"file-user","web_password":"file-password"}`), 0600); err != nil {
		t.Fatal(err)
	}
	root := newRootCommand()
	root.AddCommand(&cobra.Command{Use: "check-config", Run: func(cmd *cobra.Command, args []string) {}})
	root.SetArgs([]string{"check-config", "--config", path, "--user", "flag-user", "--password", "flag-password"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if config.GlobalConfig.WebUser != "flag-user" || config.GlobalConfig.WebPassword != "flag-password" {
		t.Fatal("CLI flags were overwritten")
	}
}

func TestStatusWithoutNotificationFails(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = previous })
	config.GlobalConfig = config.Config{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runStatusMode(cmd, nil); err == nil {
		t.Fatal("missing notification configuration reported success")
	}
}

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
