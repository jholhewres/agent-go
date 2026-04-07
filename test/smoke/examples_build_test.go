package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// TestExamplesBuild verifies that every example under cmd/examples/ compiles
// successfully. It does not run any example (no API keys needed). Build failures
// are reported via t.Errorf so all broken examples are listed in a single run.
func TestExamplesBuild(t *testing.T) {
	// Resolve the repo root relative to this test file's location.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: <repo>/test/smoke/examples_build_test.go  → go up two dirs
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	examplesDir := filepath.Join(repoRoot, "cmd", "examples")

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("cmd/examples/ directory not found, skipping")
		}
		t.Fatalf("reading cmd/examples/: %v", err)
	}

	// Collect example names that have a main.go.
	type example struct {
		name    string
		pkgPath string // module-relative import path
	}
	var examples []example
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mainFile := filepath.Join(examplesDir, e.Name(), "main.go")
		if _, err := os.Stat(mainFile); err != nil {
			continue // no main.go, skip
		}
		examples = append(examples, example{
			name:    e.Name(),
			pkgPath: "./cmd/examples/" + e.Name() + "/",
		})
	}

	if len(examples) == 0 {
		t.Skip("no examples with main.go found")
	}

	t.Logf("building %d examples", len(examples))

	// Semaphore: limit concurrency to 4 parallel builds.
	const maxConcurrent = 4
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for _, ex := range examples {
		ex := ex // capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			t.Run(ex.name, func(t *testing.T) {
				outDir := t.TempDir()
				outBin := filepath.Join(outDir, ex.name)

				// logfire_observability uses build tag "logfire"; skip it in
				// the default build tag set so it doesn't cause spurious failure.
				if ex.name == "logfire_observability" {
					t.Skip("requires build tag 'logfire' and external OTEL dependencies")
				}

				cmd := exec.Command("go", "build", "-o", outBin, ex.pkgPath)
				cmd.Dir = repoRoot
				// Strip any live API keys from the environment — build must
				// succeed without them.
				cmd.Env = cleanEnv()

				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Errorf("build failed for %s:\n%s", ex.name, string(out))
				}
			})
		}()
	}
	wg.Wait()
}

// cleanEnv returns os.Environ() minus any API key variables so builds are
// verified without external credentials.
func cleanEnv() []string {
	drop := map[string]bool{
		"OPENAI_API_KEY":      true,
		"ANTHROPIC_API_KEY":   true,
		"GROQ_API_KEY":        true,
		"ZHIPUAI_API_KEY":     true,
		"GEMINI_API_KEY":      true,
		"DEEPSEEK_API_KEY":    true,
		"DASHSCOPE_API_KEY":   true,
		"LOGFIRE_WRITE_TOKEN": true,
	}
	var env []string
	for _, kv := range os.Environ() {
		key := kv
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				key = kv[:i]
				break
			}
		}
		if !drop[key] {
			env = append(env, kv)
		}
	}
	return env
}
