package commands

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_MakefileBuildsRenamedTrio is a hermetic end-to-end check of
// the clone flow: copy the module's Go sources into a temp dir, re-identify the
// binary trio through the real plan/apply path, then run the renamed `make
// build` and confirm all three binaries land in bin/. It is skipped under
// -short and when make or go is unavailable.
func TestIntegration_MakefileBuildsRenamedTrio(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skipf("make not available: %v", err)
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go not available: %v", err)
	}

	repo := findRepoRoot(t)
	tmp := t.TempDir()

	// Copy only the paths the build needs, never the real repo tree.
	for _, name := range []string{"go.mod", "go.sum", "Makefile"} {
		copyFile(t, filepath.Join(repo, name), filepath.Join(tmp, name))
	}
	for _, dir := range []string{"cmd", "internal"} {
		copyTree(t, filepath.Join(repo, dir), filepath.Join(tmp, dir))
	}

	cur := defaultIdentity()
	next := newIdentityVals("myapp", "myapp", ".myapp", "myapp")
	p := mustPlan(t, tmp, cur, next)

	if err := apply(p, false); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	mk, err := os.ReadFile(filepath.Join(tmp, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	if !strings.Contains(string(mk), "BIN_TRIO ?= myapp") {
		t.Fatalf("Makefile not re-identified:\n%s", mk)
	}

	cmd := exec.Command("make", "build")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build failed: %v\n%s", err, out)
	}

	for _, name := range []string{"myapp", "myapp-config", "myapp-migration"} {
		bin := filepath.Join(tmp, "bin", name)
		if _, err := os.Stat(bin); err != nil {
			t.Errorf("expected binary %s: %v\nmake output:\n%s", bin, err, out)
		}
	}
}

// findRepoRoot walks up from the test's working directory to the first
// directory containing go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}

// copyTree recursively copies src into dst, skipping any directory named .git
// or bin so the copy stays hermetic and free of build artifacts.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "bin") {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy tree %s: %v", src, err)
	}
}

// copyFile copies a single regular file.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}
