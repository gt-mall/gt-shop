package bootstrap

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	colorReset  = "\x1b[0m"
	colorPurple = "\x1b[38;2;171;121;255m"
	colorAmber  = "\x1b[38;2;255;197;90m"
	colorGreen  = "\x1b[38;2;111;232;160m"
	colorRed    = "\x1b[38;2;255;112;112m"
)

// FormatStartupInfo returns plain text suitable for redirected log files.
func FormatStartupInfo(info StartupInfo) string {
	return renderStartupInfo(info, false)
}

// PrintStartupInfo uses color for terminals. STARTUP_COLOR=always or never overrides auto detection.
func PrintStartupInfo(info StartupInfo) {
	fmt.Fprint(os.Stdout, renderStartupInfo(info, startupColorEnabled()))
}

func startupColorEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("STARTUP_COLOR"))) {
	case "always", "true", "1":
		return true
	case "never", "false", "0":
		return false
	}
	stat, err := os.Stdout.Stat()
	return err == nil && stat.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb"
}

func renderStartupInfo(info StartupInfo, color bool) string {
	version := strings.TrimSpace(info.Version)
	if version == "" {
		version = "unknown"
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	environment := strings.ToUpper(strings.TrimSpace(info.Environment))
	if environment == "" {
		environment = "PRODUCTION"
	}
	title := "  GT-Shop " + version
	right := environment + "  "
	width := max(54, displayWidth(title)+displayWidth(right)+1)
	header := "│" + title + strings.Repeat(" ", width-displayWidth(title)-displayWidth(right)) + right + "│"

	var out strings.Builder
	out.WriteString(paint("╭"+strings.Repeat("─", width)+"╮", colorPurple, color) + "\n")
	out.WriteString(paint(header, colorPurple, color) + "\n")
	out.WriteString(paint("╰"+strings.Repeat("─", width)+"╯", colorPurple, color) + "\n\n")

	writeConnection := func(label, status, value string) {
		state := connectionState(status)
		if label == "HTTP" {
			state = "● Running"
		}
		out.WriteString("  " + pad(label, 10) + paint(pad(state, 15), statusColor(status), color) + displayValue(value) + "\n")
	}
	writeConnection("HTTP", "connected", info.Listen)
	writeConnection("MySQL", info.DatabaseStatus, info.DatabaseTarget)
	writeConnection("Redis", info.RedisStatus, info.RedisTarget)
	out.WriteByte('\n')

	schema := displayValue(info.SchemaVersion)
	if info.SchemaStatus != "ok" {
		schema = generalState(info.SchemaStatus)
	}
	out.WriteString("  " + pad("Schema", 10) + paint(schema, statusColor(info.SchemaStatus), color) + "\n")
	logs := displayValue(info.LogsValue)
	if info.LogsStatus != "ok" {
		logs = generalState(info.LogsStatus) + " · " + logs
	}
	out.WriteString("  " + pad("Logs", 10) + paint(logs, statusColor(info.LogsStatus), color) + "\n\n")

	ready := "✔ Ready"
	readyColor := colorGreen
	if info.DatabaseStatus == "failed" || info.RedisStatus == "failed" || info.SchemaStatus == "failed" || info.SchemaStatus == "missing" || info.LogsStatus == "failed" {
		ready = "✘ Startup incomplete"
		readyColor = colorRed
	}
	out.WriteString("  " + paint(ready, readyColor, color) + "\n")
	return out.String()
}

func pad(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-displayWidth(value)))
}

func displayWidth(value string) int {
	return utf8.RuneCountInString(value)
}

func paint(value, code string, enabled bool) string {
	if !enabled {
		return value
	}
	return code + value + colorReset
}

func connectionState(status string) string {
	switch status {
	case "connected":
		return "● Connected"
	case "disabled":
		return "Disabled"
	case "not configured":
		return "Not configured"
	default:
		return "● Failed"
	}
}

func generalState(status string) string {
	switch status {
	case "ok":
		return "OK"
	case "disabled":
		return "Disabled"
	case "missing":
		return "Missing"
	case "empty":
		return "Empty"
	case "unavailable":
		return "Unavailable"
	default:
		return "Failed"
	}
}

func statusColor(status string) string {
	switch status {
	case "connected", "ok":
		return colorGreen
	case "failed", "missing":
		return colorRed
	default:
		return colorAmber
	}
}

func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
