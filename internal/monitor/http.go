package monitor

import (
	"context"
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
	return RunChecksContext(context.Background())
}

func RunChecksContext(ctx context.Context) []CheckResult {
	var results []CheckResult
	client := http.Client{
		Timeout: 10 * time.Second, // 10秒超时
	}

	for _, rule := range config.GlobalConfig.HTTPRules {
		if ctx.Err() != nil {
			break
		}
		start := time.Now()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, rule.URL, nil)
		var resp *http.Response
		if err == nil {
			resp, err = client.Do(request)
		}
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
			resp.Body.Close()
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
