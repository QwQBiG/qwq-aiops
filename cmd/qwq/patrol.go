package main

import (
	"context"
	"fmt"
	"math"
	"qwq/internal/config"
	"strconv"
	"strings"
)

type patrolResult struct {
	anomalies []string
	failures  []string
}

// Failed checks remain separate from observed anomalies and successful silence.
func collectPatrolChecks(ctx context.Context, rules []config.PatrolRule, run func(context.Context, string) (string, error)) patrolResult {
	result := patrolResult{}
	read := func(name, command string) (string, bool) {
		if ctx.Err() != nil {
			return "", false
		}
		output, err := run(ctx, command)
		if err != nil {
			result.failures = append(result.failures, fmt.Sprintf("%s: %v\n%s", name, err, strings.TrimSpace(output)))
			return "", false
		}
		return strings.TrimSpace(output), true
	}
	add := func(name, output string) {
		if output != "" {
			result.anomalies = append(result.anomalies, fmt.Sprintf("**%s**:\n```\n%s\n```", name, output))
		}
	}
	if output, ok := read("磁盘检查", "LC_ALL=C df -hP -x tmpfs -x devtmpfs"); ok {
		alerts, err := diskPatrolAlerts(output)
		if err != nil {
			result.failures = append(result.failures, "磁盘检查: "+err.Error())
		}
		add("磁盘告警", strings.Join(alerts, "\n"))
	}
	if output, ok := read("负载检查", "LC_ALL=C uptime"); ok {
		_, averages, found := strings.Cut(output, "load average:")
		parts := strings.Split(strings.TrimSpace(averages), ",")
		valid := found && len(parts) == 3
		var load float64
		for i, part := range parts {
			value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			valid = valid && err == nil && value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
			if i == 0 {
				load = value
			}
		}
		if !valid {
			result.failures = append(result.failures, "负载检查: 无法解析 uptime 输出")
		} else if load > 4 {
			add("高负载", strings.TrimSpace(averages))
		}
	}
	// awk succeeds when there are no matches; pipefail still detects dmesg errors.
	if output, ok := read("OOM日志检查", `LC_ALL=C dmesg | awk 'tolower($0) ~ /out of memory/' | tail -n 5`); ok {
		add("OOM日志", output)
	}
	if output, ok := read("僵尸进程检查", `LC_ALL=C ps -A -o stat,ppid,pid,cmd | awk '$1 ~ /^[Zz]/'`); ok {
		add("僵尸进程", output)
	}
	for _, rule := range rules {
		if output, ok := read(rule.Name, rule.Command); ok {
			add(rule.Name, output)
		}
	}
	return result
}

func diskPatrolAlerts(output string) ([]string, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "Filesystem") {
		return nil, fmt.Errorf("无法解析 df 输出")
	}
	var alerts []string
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 6 || !strings.HasSuffix(fields[4], "%") {
			return alerts, fmt.Errorf("无法解析文件系统使用率")
		}
		device, mount := fields[0], strings.Join(fields[5:], " ")
		if strings.HasPrefix(device, "/dev/loop") || strings.HasPrefix(mount, "/snap/") || strings.Contains(mount, "/hostfs") ||
			device == "tmpfs" || device == "devtmpfs" || device == "efivarfs" || strings.Contains(device, "cdrom") ||
			(device == "overlay" && mount != "/") {
			continue
		}
		percent, err := strconv.Atoi(strings.TrimSuffix(fields[4], "%"))
		if err != nil || percent < 0 {
			return alerts, fmt.Errorf("无效的文件系统使用率")
		}
		if percent > 85 {
			alerts = append(alerts, line)
		}
	}
	return alerts, nil
}
