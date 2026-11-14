package utils

import "strings"

// IsReadOnlyCommand accepts a single diagnostic command. Shell composition,
// expansion and redirection require an operator and are never run automatically.
func IsReadOnlyCommand(command string) bool {
	if strings.ContainsAny(command, ";&|<>$`\\\r\n\x00(){}") {
		return false
	}
	args := strings.Fields(command)
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "ls", "cat", "head", "tail", "grep", "pwd", "echo", "whoami", "id", "ps", "uptime", "free", "df", "du", "netstat", "lsof", "uname", "w":
		return true
	case "date":
		for _, arg := range args[1:] {
			if arg != "-u" && arg != "--utc" && !strings.HasPrefix(arg, "+") {
				return false
			}
		}
		return true
	case "hostname":
		for _, arg := range args[1:] {
			if !oneOf(arg, "-I", "-i", "-f", "-s", "-d", "--fqdn", "--short", "--domain", "--ip-address", "--all-ip-addresses") {
				return false
			}
		}
		return true
	case "ss":
		for _, arg := range args[1:] {
			if !strings.HasPrefix(arg, "-") || len(arg) < 2 || strings.Trim(arg[1:], "stuanlop") != "" {
				return false
			}
		}
		return true
	case "top":
		return len(args) > 1 && args[1] == "-b"
	case "docker":
		return len(args) > 1 && oneOf(args[1], "ps", "images", "logs", "stats", "inspect", "version", "info")
	case "kubectl":
		if len(args) == 2 && args[1] == "cluster-info" {
			return true
		}
		return len(args) > 1 && oneOf(args[1], "get", "describe", "logs", "top", "version")
	case "systemctl":
		return len(args) > 1 && oneOf(args[1], "status", "show", "is-active", "is-enabled", "list-units", "list-unit-files")
	case "ip":
		return len(args) > 1 && oneOf(args[1], "address", "addr", "route", "link") && (len(args) == 2 || args[2] == "show")
	default:
		return false
	}
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
