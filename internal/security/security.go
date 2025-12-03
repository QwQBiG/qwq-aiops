package security

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

var (
	ipRegex    = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	keyRegex   = regexp.MustCompile(`(sk-[a-zA-Z0-9]{20,}|AKIA[0-9A-Z]{16})`)
)

func Redact(input string) string {
	masked := input
	masked = ipRegex.ReplaceAllString(masked, "<IP_REDACTED>")
	masked = emailRegex.ReplaceAllString(masked, "<EMAIL_REDACTED>")
	masked = keyRegex.ReplaceAllString(masked, "<SECRET_KEY_REDACTED>")
	return masked
}

type RiskLevel int

const (
	RiskLow      RiskLevel = iota
	RiskMedium
	RiskHigh
	RiskCritical
)

func CheckRisk(cmd string) RiskLevel {
	c := strings.ToLower(cmd)
	if strings.Contains(c, "rm -rf /") || strings.Contains(c, "mkfs") || strings.Contains(c, "> /dev/sda") {
		return RiskCritical
	}
	highKeywords := []string{"rm ", "kill", "fdisk", "mount", "umount", "kubectl delete", "drop table"}
	for _, kw := range highKeywords {
		if strings.Contains(c, kw) {
			return RiskHigh
		}
	}
	mediumKeywords := []string{"docker", "systemctl", "service", "iptables", "chmod", "chown", "wget", "curl"}
	for _, kw := range mediumKeywords {
		if strings.Contains(c, kw) {
			return RiskMedium
		}
	}
	return RiskLow
}

func GenerateVerifyCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}