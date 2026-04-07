package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPostgresStorageExampleBuilds verifies that the postgres_storage example
// compiles successfully. It does not connect to any real Postgres instance.
func TestPostgresStorageExampleBuilds(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: <repo>/test/smoke/postgres_storage_smoke_test.go → up two dirs
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	pkgPath := "./cmd/examples/postgres_storage/"

	outDir := t.TempDir()
	outBin := filepath.Join(outDir, "postgres_storage")

	cmd := exec.Command("go", "build", "-o", outBin, pkgPath)
	cmd.Dir = repoRoot
	cmd.Env = cleanEnv()

	// Ensure DATABASE_URL is absent so the binary is built without any live DB.
	var filtered []string
	for _, kv := range cmd.Env {
		if len(kv) >= 12 && kv[:12] == "DATABASE_URL" {
			continue
		}
		filtered = append(filtered, kv)
	}
	cmd.Env = filtered

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("postgres_storage example build failed:\n%s", string(out))
		return
	}

	if _, err := os.Stat(outBin); err != nil {
		t.Errorf("binary not produced at %s: %v", outBin, err)
	}
}
