package main

import (
	"context"
	"errors"
	"qwq/internal/config"
	"strings"
	"testing"
)

func successfulPatrolOutput(command string) string {
	switch command {
	case "LC_ALL=C df -hP -x tmpfs -x devtmpfs":
		return "Filesystem Size Used Avail Use% Mounted on\n/dev/root 10G 2G 8G 20% /\n"
	case "LC_ALL=C uptime":
		return "up 1 hour, load average: 0.00, 0.10, 0.20"
	default:
		return ""
	}
}

func TestPatrolSeparatesSuccessfulSilenceFromFailures(t *testing.T) {
	ctx := context.Background()
	result := collectPatrolChecks(ctx, nil, func(_ context.Context, command string) (string, error) {
		return successfulPatrolOutput(command), nil
	})
	if len(result.anomalies) != 0 || len(result.failures) != 0 {
		t.Fatalf("successful checks were not quiet: %+v", result)
	}
	result = collectPatrolChecks(ctx, []config.PatrolRule{{Name: "custom", Command: "check-custom"}}, func(_ context.Context, command string) (string, error) {
		return "tool unavailable", errors.New("exit status 127")
	})
	if len(result.failures) != 5 || len(result.anomalies) != 0 || !strings.Contains(strings.Join(result.failures, " "), "tool unavailable") {
		t.Fatalf("execution failures became apparent checks or anomalies: %+v", result)
	}
}

func TestPatrolReportsRootOverlayAndCustomFindings(t *testing.T) {
	result := collectPatrolChecks(context.Background(), []config.PatrolRule{{Name: "custom", Command: "check-custom"}}, func(_ context.Context, command string) (string, error) {
		switch command {
		case "LC_ALL=C df -hP -x tmpfs -x devtmpfs":
			return "Filesystem Size Used Avail Use% Mounted on\noverlay 10G 9G 1G 90% /\n/dev/loop0 1G 1G 0G 100% /snap/pkg/1\n", nil
		case "LC_ALL=C uptime":
			return "up 1 hour, load average: 5.00, 2.00, 1.00", nil
		case "check-custom":
			return "service needs attention", nil
		default:
			return "", nil
		}
	})
	if len(result.anomalies) != 3 || len(result.failures) != 0 || strings.Contains(strings.Join(result.anomalies, " "), "/snap/pkg") {
		t.Fatalf("incorrect findings: %+v", result)
	}
}

func TestPatrolRejectsMalformedMeasurementsAndStopsOnCancellation(t *testing.T) {
	result := collectPatrolChecks(context.Background(), nil, func(_ context.Context, command string) (string, error) {
		if command == "LC_ALL=C uptime" {
			return "load average: NaN, 0, 0", nil
		}
		if command == "LC_ALL=C df -hP -x tmpfs -x devtmpfs" {
			return "unrecognized output", nil
		}
		return "", nil
	})
	if len(result.failures) != 2 || len(result.anomalies) != 0 {
		t.Fatalf("malformed measurements were accepted: %+v", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	collectPatrolChecks(ctx, nil, func(context.Context, string) (string, error) {
		t.Fatal("a cancelled patrol executed another command")
		return "", nil
	})
}
