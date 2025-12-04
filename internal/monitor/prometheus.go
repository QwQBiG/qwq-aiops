package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CpuLoad = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "qwq_system_cpu_load_1min",
		Help: "System CPU Load (1 min)",
	})
	MemUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "qwq_system_mem_usage_percent",
		Help: "System Memory Usage Percent",
	})
	DiskUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "qwq_system_disk_usage_percent",
		Help: "Root Disk Usage Percent",
	})
	TcpConn = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "qwq_system_tcp_connections",
		Help: "Total TCP Established Connections",
	})
	AppStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "qwq_app_health_status",
		Help: "Application Health Status (1=UP, 0=DOWN)",
	}, []string{"name", "url"})
)

func UpdatePrometheusMetrics(load, memPct, diskPct, tcpConn float64) {
	CpuLoad.Set(load)
	MemUsage.Set(memPct)
	DiskUsage.Set(diskPct)
	TcpConn.Set(tcpConn)
}

func UpdateAppMetrics(results []CheckResult) {
	for _, res := range results {
		val := 0.0
		if res.Success {
			val = 1.0
		}
		AppStatus.WithLabelValues(res.Name, res.URL).Set(val)
	}
}