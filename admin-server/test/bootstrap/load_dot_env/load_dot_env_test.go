package load_dot_env_test

import (
	"os"
	"path/filepath"
	"testing"

	"mall/admin_core/bootstrap"
)

func TestLoadDotEnvPrefersProjectRootAndOverridesSystem(t *testing.T) {
	root := t.TempDir()
	workingDir := filepath.Join(root, "release", "run")
	for _, dir := range []string{
		filepath.Join(root, "admin-server"),
		filepath.Join(root, "scripts"),
		workingDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rootEnv := filepath.Join(root, ".env")
	workingEnv := filepath.Join(workingDir, ".env")
	if err := os.WriteFile(rootEnv, []byte("DOTENV_PRIORITY_TEST=from-root\nDOTENV_QUOTED_TEST=\"tcp(127.0.0.1:3306)/shop?parseTime=true\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workingEnv, []byte("DOTENV_PRIORITY_TEST=from-working-directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Chdir(workingDir)
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("DOTENV_PRIORITY_TEST", "from-system")

	loadedPath, err := bootstrap.LoadDotEnv()
	if err != nil {
		t.Fatal(err)
	}
	if loadedPath != rootEnv {
		t.Fatalf("expected project root .env %q, got %q", rootEnv, loadedPath)
	}
	if got := os.Getenv("DOTENV_PRIORITY_TEST"); got != "from-root" {
		t.Fatalf("project root .env should override system and working directory values, got %q", got)
	}
	if got := os.Getenv("DOTENV_QUOTED_TEST"); got != "tcp(127.0.0.1:3306)/shop?parseTime=true" {
		t.Fatalf("unexpected quoted DSN value: %q", got)
	}
}

func TestLoadDotEnvFallsBackToSystemWhenFileIsMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("DOTENV_SYSTEM_FALLBACK_TEST", "from-system")

	loadedPath, err := bootstrap.LoadDotEnv()
	if err != nil {
		t.Fatal(err)
	}
	if loadedPath != "" {
		t.Fatalf("expected no dotenv file, got %q", loadedPath)
	}
	if got := os.Getenv("DOTENV_SYSTEM_FALLBACK_TEST"); got != "from-system" {
		t.Fatalf("expected system environment value to remain available, got %q", got)
	}
}

func TestLoadDotEnvFileDoesNotApplyPartialValuesOnSyntaxError(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DOTENV_ATOMIC_TEST=from-file\ninvalid assignment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTENV_ATOMIC_TEST", "from-system")

	if err := bootstrap.LoadDotEnvFile(path); err == nil {
		t.Fatal("expected malformed dotenv assignment to fail")
	}
	if got := os.Getenv("DOTENV_ATOMIC_TEST"); got != "from-system" {
		t.Fatalf("invalid file must not partially override process environment, got %q", got)
	}
}
