package monitor

import (
	"fmt"
	"net/http"
	"qwq/internal/config"
	"time"
)

// CheckResult 检查结果
type CheckResult struct {
	Name    string
	URL     string
	Success bool
	Latency string
	Error   string
}

// RunChecks 执行所有 HTTP 检查
func RunChecks() []CheckResult {
	var results []CheckResult
	client := http.Client{
		Timeout: 10 * time.Second, // 10秒超时
	}

	for _, rule := range config.GlobalConfig.HTTPRules {
		start := time.Now()
		resp, err := client.Get(rule.URL)
		latency := time.Since(start).Milliseconds()
		
		res := CheckResult{
			Name:    rule.Name,
			URL:     rule.URL,
			Latency: fmt.Sprintf("%dms", latency),
		}

		expectedCode := rule.Code
		if expectedCode == 0 {
			expectedCode = 200
		}

		if err != nil {
			res.Success = false
			res.Error = fmt.Sprintf("连接失败: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != expectedCode {
				res.Success = false
				res.Error = fmt.Sprintf("状态码异常: %d (期望 %d)", resp.StatusCode, expectedCode)
			} else {
				res.Success = true
			}
		}
		results = append(results, res)
	}
	return results
}