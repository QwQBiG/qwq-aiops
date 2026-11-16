package monitor

import (
	"context"
	"fmt"
	"qwq/internal/utils"
	"regexp"
	"strconv"
	"strings"
)

// SystemStats contains one Linux system snapshot. Empty fields are unavailable,
// while measured zero values remain numeric strings.
type SystemStats struct {
	Load      string            `json:"load"`
	MemPct    string            `json:"mem_pct"`
	MemUsed   string            `json:"mem_used"`
	MemTotal  string            `json:"mem_total"`
	DiskPct   string            `json:"disk_pct"`
	DiskAvail string            `json:"disk_avail"`
	TcpConn   string            `json:"tcp_conn"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// CollectSystemStats reports command and parsing failures alongside valid groups.
func CollectSystemStats(ctx context.Context) SystemStats {
	return collectSystemStats(ctx, utils.ExecuteShellResultContext)
}

var establishedConnections = regexp.MustCompile(`\bestab\s+(\d+)\b`)

func collectSystemStats(ctx context.Context, run func(context.Context, string) (string, error)) SystemStats {
	stats := SystemStats{Errors: map[string]string{}}
	read := func(key, command string, parse func(string) error) {
		output, err := run(ctx, command)
		if err == nil {
			err = parse(output)
		} else if detail := strings.TrimSpace(output); detail != "" {
			err = fmt.Errorf("%w: %s", err, detail)
		}
		if err != nil {
			stats.Errors[key] = err.Error()
		}
	}
	read("load", "LC_ALL=C uptime", func(output string) error {
		_, values, found := strings.Cut(output, "load average:")
		if !found {
			return fmt.Errorf("load averages were not present in uptime output")
		}
		parts := strings.Split(strings.TrimSpace(values), ",")
		if len(parts) != 3 {
			return fmt.Errorf("expected three load averages")
		}
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
			if _, err := nonnegativeNumber(parts[i]); err != nil {
				return err
			}
		}
		stats.Load = strings.Join(parts, ", ")
		return nil
	})
	read("memory", "LC_ALL=C free -m", func(output string) error {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 || fields[0] != "Mem:" {
				continue
			}
			total, err := nonnegativeNumber(fields[1])
			if err != nil || total == 0 {
				return fmt.Errorf("invalid total memory")
			}
			used, err := nonnegativeNumber(fields[2])
			if err != nil || used > total {
				return fmt.Errorf("invalid used memory")
			}
			stats.MemTotal, stats.MemUsed = fields[1], fields[2]
			stats.MemPct = fmt.Sprintf("%.1f", used/total*100)
			return nil
		}
		return fmt.Errorf("memory data was not present in free output")
	})
	read("disk", "LC_ALL=C df -hP /", func(output string) error {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 2 {
			return fmt.Errorf("expected one root filesystem in df output")
		}
		fields := strings.Fields(lines[1])
		if len(fields) < 6 || !strings.HasSuffix(fields[4], "%") {
			return fmt.Errorf("invalid root filesystem data")
		}
		percent := strings.TrimSuffix(fields[4], "%")
		if _, err := nonnegativeNumber(percent); err != nil {
			return err
		}
		stats.DiskPct, stats.DiskAvail = percent, fields[3]
		return nil
	})
	read("tcp", "LC_ALL=C ss -s", func(output string) error {
		matches := establishedConnections.FindStringSubmatch(output)
		if len(matches) != 2 {
			return fmt.Errorf("established connection count was not present in ss output")
		}
		stats.TcpConn = matches[1]
		return nil
	})
	return stats
}

func nonnegativeNumber(value string) (float64, error) {
	// Shell tools emit decimal values, not NaN, infinity, or exponent notation.
	if value == "" || strings.Trim(value, "0123456789.") != "" {
		return 0, fmt.Errorf("invalid numeric system value: %q", value)
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric system value: %q", value)
	}
	return number, nil
}
