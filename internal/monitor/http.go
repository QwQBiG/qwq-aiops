package monitor

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	redirectClient := client
	redirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	for _, rule := range config.GlobalConfig.HTTPRules {
		if ctx.Err() != nil {
			break
		}
		expectedCode := rule.Code
		if expectedCode == 0 {
			expectedCode = http.StatusOK
		}
		start := time.Now()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, rule.URL, nil)
		res := CheckResult{Name: rule.Name}
		if err != nil {
			res.Error = "无效的检查地址"
			results = append(results, res)
			continue
		}
		displayURL := *request.URL
		displayURL.User = nil
		displayURL.RawQuery, displayURL.Fragment, displayURL.RawFragment = "", "", ""
		displayURL.ForceQuery = false
		res.URL = displayURL.String()

		checkClient := &client
		if expectedCode >= 300 && expectedCode < 400 {
			checkClient = &redirectClient
		}
		resp, err := checkClient.Do(request)
		res.Latency = fmt.Sprintf("%dms", time.Since(start).Milliseconds())
		if err != nil {
			res.Error = checkRequestError(err)
		} else {
			resp.Body.Close()
			if resp.StatusCode != expectedCode {
				res.Error = fmt.Sprintf("状态码异常: %d (期望 %d)", resp.StatusCode, expectedCode)
			} else {
				res.Success = true
			}
		}
		results = append(results, res)
	}
	return results
}

// Client errors can include both the original URL and redirect locations.
func checkRequestError(err error) string {
	if errors.Is(err, context.Canceled) {
		return "检查已取消"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "连接超时"
	}
	return "连接失败"
}
