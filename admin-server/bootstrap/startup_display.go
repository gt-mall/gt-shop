package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
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
	build, commit := strings.TrimSpace(info.BuildTime), strings.TrimSpace(info.GitCommit)
	if build != "" && build != "unknown" && commit != "" && commit != "unknown" {
		version += " (" + build + "-" + commit + ")"
	}
	environment := strings.TrimSpace(info.Environment)
	if environment == "" {
		environment = "production"
	}

	row := func(label, value string) string { return pad(label, 16) + value }
	binary := filepath.Base(info.ExecutablePath)
	if binary == "." || binary == "" {
		binary = "-"
	}
	size := binary + "  " + formatBytes(info.ExecutableSize) + "   Static  " + formatBytes(info.StaticSize)
	rows := []string{
		pad("HTTP", 10) + pad("● Running", 15) + displayValue(info.Listen),
		pad("MySQL", 10) + pad(connectionState(info.DatabaseStatus), 15) + displayValue(info.DatabaseTarget),
		pad("Redis", 10) + pad(connectionState(info.RedisStatus), 15) + displayValue(info.RedisTarget),
		row("➜ Version", version),
		row("➜ Size", size),
		row("➜ Running Path", displayValue(info.ExecutablePath)),
		row("➜ Network", displayValue(info.NetworkURL)),
	}
	width := 56
	for _, value := range append(rows, "GT-Shop  "+environment) {
		width = max(width, displayWidth(value))
	}
	separator := strings.Repeat("─", width)
	status := "✔ GT-Shop started successfully"
	statusColor := colorGreen
	if info.DatabaseStatus == "failed" || info.RedisStatus == "failed" || info.SchemaStatus == "failed" || info.SchemaStatus == "missing" || info.LogsStatus == "failed" {
		status = "✘ GT-Shop started with unavailable dependencies"
		statusColor = colorRed
	}

	var out strings.Builder
	out.WriteString(paint("GT-Shop", colorPurple, color))
	out.WriteString(strings.Repeat(" ", width-displayWidth("GT-Shop")-displayWidth(environment)))
	out.WriteString(environment + "\n")
	out.WriteString(separator + "\n")
	out.WriteString(pad("HTTP", 10) + paint(pad("● Running", 15), colorGreen, color) + displayValue(info.Listen) + "\n")
	out.WriteString(pad("MySQL", 10) + paint(pad(connectionState(info.DatabaseStatus), 15), connectionColor(info.DatabaseStatus), color) + displayValue(info.DatabaseTarget) + "\n")
	out.WriteString(pad("Redis", 10) + paint(pad(connectionState(info.RedisStatus), 15), connectionColor(info.RedisStatus), color) + displayValue(info.RedisTarget) + "\n")
	out.WriteString(separator + "\n")
	out.WriteString(row("➜ Version", version) + "\n")
	out.WriteString(row("➜ Size", size) + "\n")
	out.WriteString(row("➜ Running Path", displayValue(info.ExecutablePath)) + "\n")
	out.WriteString(row("➜ Network", displayValue(info.NetworkURL)) + "\n")
	out.WriteString(separator + "\n")
	out.WriteString(paint(status, statusColor, color) + "\n")
	return out.String()
}

func formatBytes(size int64) string {
	if size <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f MB", float64(size)/1024/1024)
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

func connectionColor(status string) string {
	switch status {
	case "connected":
		return colorGreen
	case "failed":
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
