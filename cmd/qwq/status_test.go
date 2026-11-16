package main

import (
	"qwq/internal/monitor"
	"strings"
	"testing"
)

func TestStatusReportPreservesMeasuredZeroAndUnavailableMetrics(t *testing.T) {
	stats := monitor.SystemStats{
		Load: "0.00, 0.00, 0.00", MemPct: "0.0", MemUsed: "0", MemTotal: "1024",
		DiskPct: "0", DiskAvail: "10G", TcpConn: "0",
	}
	report := formatStatusReport("host", "127.0.0.1", "up 1 hour", "timestamp", stats)
	for _, want := range []string{"0.0% (已用 0M / 总计 1024M)", "0% (剩余 10G)", "**TCP连接** | 0"} {
		if !strings.Contains(report, want) {
			t.Fatalf("measured zero was lost: %q", want)
		}
	}
	stats.Errors = map[string]string{"memory": "free unavailable", "tcp": "ss unavailable"}
	report = formatStatusReport("host", "127.0.0.1", "up 1 hour", "timestamp", stats)
	for _, want := range []string{"**内存使用** | N/A", "**TCP连接** | N/A", "free unavailable", "ss unavailable"} {
		if !strings.Contains(report, want) {
			t.Fatalf("collection failure was hidden: %q", want)
		}
	}
	if strings.Contains(report, "0.0%") {
		t.Fatal("failed memory collection was shown as a measurement")
	}
}
