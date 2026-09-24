package bootstrap

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadDotEnv loads the first .env file found, preferring the project root,
// then the current working directory, then the executable directory.
// Values from the file override matching process environment variables.
// If no file is found, the existing process environment is left unchanged.
func LoadDotEnv() (string, error) {
	for _, path := range dotEnvCandidates() {
		if _, err := os.Stat(path); err == nil || !errors.Is(err, os.ErrNotExist) {
			return path, LoadDotEnvFile(path)
		}
	}
	return "", nil
}

// LoadDotEnvFile parses a dotenv file and exports its values to the process.
// It parses the whole file before changing the environment, so a syntax error
// never leaves a partially applied configuration.
func LoadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open environment file: %w", err)
	}
	defer file.Close()

	values, err := parseDotEnv(file)
	if err != nil {
		return fmt.Errorf("parse environment file: %w", err)
	}
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set environment variable %s: %w", key, err)
		}
	}
	return nil
}

func dotEnvCandidates() []string {
	var candidates []string
	if root := strings.TrimSpace(os.Getenv("PROJECT_ROOT")); root != "" {
		candidates = append(candidates, filepath.Join(root, ".env"))
	}

	workingDir, _ := os.Getwd()
	executable, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	executableDir := filepath.Dir(executable)

	if root := findProjectRoot(workingDir); root != "" {
		candidates = append(candidates, filepath.Join(root, ".env"))
	}
	if root := findProjectRoot(executableDir); root != "" {
		candidates = append(candidates, filepath.Join(root, ".env"))
	}
	if workingDir != "" {
		candidates = append(candidates, filepath.Join(workingDir, ".env"))
	}
	if executableDir != "." && executableDir != "" {
		candidates = append(candidates, filepath.Join(executableDir, ".env"))
	}

	seen := make(map[string]struct{}, len(candidates))
	unique := candidates[:0]
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			absolute = filepath.Clean(candidate)
		}
		if _, ok := seen[absolute]; ok {
			continue
		}
		seen[absolute] = struct{}{}
		unique = append(unique, absolute)
	}
	return unique
}

func findProjectRoot(start string) string {
	if start == "" {
		return ""
	}
	current, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		_, serverErr := os.Stat(filepath.Join(current, "admin-server"))
		_, scriptsErr := os.Stat(filepath.Join(current, "scripts"))
		if serverErr == nil && scriptsErr == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func parseDotEnv(reader io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if lineNumber == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, rawValue, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !validEnvKey(key) {
			return nil, fmt.Errorf("invalid assignment on line %d", lineNumber)
		}
		value, err := parseDotEnvValue(strings.TrimSpace(rawValue))
		if err != nil {
			return nil, fmt.Errorf("invalid value for %s on line %d: %w", key, lineNumber, err)
		}
		if strings.IndexByte(value, 0) >= 0 {
			return nil, fmt.Errorf("invalid NUL byte for %s on line %d", key, lineNumber)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read environment file: %w", err)
	}
	return values, nil
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, char := range key {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '_' || (i > 0 && char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}

func parseDotEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	quote := raw[0]
	if quote == '\'' || quote == '"' {
		end := -1
		escaped := false
		for i := 1; i < len(raw); i++ {
			if quote == '"' && raw[i] == '\\' && !escaped {
				escaped = true
				continue
			}
			if raw[i] == quote && !escaped {
				end = i
				break
			}
			escaped = false
		}
		if end < 0 {
			return "", errors.New("unterminated quoted value")
		}
		remainder := strings.TrimSpace(raw[end+1:])
		if remainder != "" && !strings.HasPrefix(remainder, "#") {
			return "", errors.New("unexpected text after quoted value")
		}
		if quote == '\'' {
			return raw[1:end], nil
		}
		value, err := strconv.Unquote(raw[:end+1])
		if err != nil {
			return "", err
		}
		return value, nil
	}

	for i := 0; i < len(raw); i++ {
		if raw[i] == '#' && (i == 0 || raw[i-1] == ' ' || raw[i-1] == '\t') {
			return strings.TrimSpace(raw[:i]), nil
		}
	}
	return strings.TrimSpace(raw), nil
}
