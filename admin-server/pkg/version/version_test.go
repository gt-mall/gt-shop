package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	info := Get()
	if info.Version == "" {
		t.Fatalf("expected non-empty Version, got empty string")
	}
	if info.BuildTime == "" {
		t.Fatalf("expected non-empty BuildTime, got empty string")
	}
	if info.GitCommit == "" {
		t.Fatalf("expected non-empty GitCommit, got empty string")
	}
}

func TestString(t *testing.T) {
	str := String()
	if !strings.HasPrefix(str, "GT-Shop v") {
		t.Fatalf("expected prefix 'GT-Shop v', got %q", str)
	}
	if !strings.Contains(str, "(") || !strings.Contains(str, ")") {
		t.Fatalf("expected build info enclosed in parentheses, got %q", str)
	}
}
