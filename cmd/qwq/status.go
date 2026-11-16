package main

import (
	"fmt"
	"qwq/internal/monitor"
	"slices"
	"strings"
)

func formatStatusReport(hostname, ip, uptime, timestamp string, stats monitor.SystemStats) string {
	display := func(group, value string) string {
		if value == "" || stats.Errors[group] != "" {
			return "N/A"
		}
		return value
	}
	memory := display("memory", stats.MemPct)
	if memory != "N/A" {
		memory = fmt.Sprintf("%s%% (已用 %sM / 总计 %sM)", memory, stats.MemUsed, stats.MemTotal)
	}
	disk := display("disk", stats.DiskPct)
	if disk != "N/A" {
		disk = fmt.Sprintf("%s%% (剩余 %s)", disk, stats.DiskAvail)
	}
	report := fmt.Sprintf(`### 📊 服务器状态日报 [%s]

> **IP**: %s
> **运行时间**: %s
> **报告时间**: %s

| 指标 | 状态 |
| :--- | :--- |
| **CPU负载** | %s |
| **内存使用** | %s |
| **系统磁盘** | %s |
| **TCP连接** | %s |
`, hostname, ip, uptime, timestamp, display("load", stats.Load), memory, disk, display("tcp", stats.TcpConn))
	if len(stats.Errors) > 0 {
		report += "\n### 采集失败\n\n"
		keys := make([]string, 0, len(stats.Errors))
		for key := range stats.Errors {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			report += fmt.Sprintf("- %s: %s\n", key, strings.TrimSpace(stats.Errors[key]))
		}
	}
	return report
}
