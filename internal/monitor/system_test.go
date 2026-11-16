package monitor

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func systemOutputs() map[string]string {
	return map[string]string{
		"LC_ALL=C uptime":   " 10:20:30 up 3 days, 1 user, load average: 0.00, 1.25, 2.50\n",
		"LC_ALL=C free -m":  "               total used free shared buff/cache available\nMem: 2048 1024 256 0 768 1024\nSwap: 0 0 0\n",
		"LC_ALL=C df -hP /": "Filesystem Size Used Avail Use% Mounted on\n/dev/root 10G 2G 8G 20% /\n",
		"LC_ALL=C ss -s":    "Total: 12\nTCP: 3 (estab 0, closed 0, orphaned 0, timewait 0)\n",
	}
}

func TestCollectSystemStatsRetainsZeroAndParsesGroups(t *testing.T) {
	outputs := systemOutputs()
	stats := collectSystemStats(context.Background(), func(ctx context.Context, command string) (string, error) {
		output, ok := outputs[command]
		if !ok {
			t.Fatalf("unexpected system command: %s", command)
		}
		return output, nil
	})
	if len(stats.Errors) != 0 || stats.Load != "0.00, 1.25, 2.50" || stats.MemPct != "50.0" || stats.MemUsed != "1024" || stats.MemTotal != "2048" || stats.DiskPct != "20" || stats.DiskAvail != "8G" || stats.TcpConn != "0" {
		t.Fatalf("incorrect system snapshot: %#v", stats)
	}
}

func TestCollectSystemStatsSeparatesUnavailableGroups(t *testing.T) {
	outputs := systemOutputs()
	stats := collectSystemStats(context.Background(), func(ctx context.Context, command string) (string, error) {
		if strings.Contains(command, "free") {
			return "free unavailable", errors.New("exit status 127")
		}
		if strings.Contains(command, "df") {
			return "invalid disk output", nil
		}
		return outputs[command], nil
	})
	if len(stats.Errors) != 2 || !strings.Contains(stats.Errors["memory"], "free unavailable") || stats.Errors["disk"] == "" {
		t.Fatalf("missing failure details: %#v", stats.Errors)
	}
	if stats.MemPct != "" || stats.MemUsed != "" || stats.MemTotal != "" || stats.DiskPct != "" || stats.DiskAvail != "" {
		t.Fatalf("failed collection produced apparent measurements: %#v", stats)
	}
	if stats.Load == "" || stats.TcpConn != "0" {
		t.Fatal("valid groups were lost with an unrelated failure")
	}
}

func TestCollectSystemStatsRejectsMalformedMeasurements(t *testing.T) {
	for name, malformed := range map[string]map[string]string{
		"invalid numbers": {"LC_ALL=C uptime": "load average: NaN, 0, 0", "LC_ALL=C free -m": "Mem: 0 0 0", "LC_ALL=C df -hP /": "head\n/dev/root 10G 2G 8G NaN% /", "LC_ALL=C ss -s": "no connection summary"},
		"missing output":  {},
	} {
		t.Run(name, func(t *testing.T) {
			stats := collectSystemStats(context.Background(), func(ctx context.Context, command string) (string, error) { return malformed[command], nil })
			if len(stats.Errors) != 4 || stats.Load != "" || stats.MemPct != "" || stats.DiskPct != "" || stats.TcpConn != "" {
				t.Fatalf("malformed data accepted: %#v", stats)
			}
		})
	}
}
