package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything fn wrote to stdout. printVersion writes via fmt.Printf (os.Stdout),
// so this is the only way to observe its output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	out := make(chan string, 1)
	go func() {
		defer r.Close()
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		out <- buf.String()
	}()

	fn()

	os.Stdout = orig
	if err := w.Close(); err != nil {
		t.Logf("closing pipe writer: %v", err)
	}
	return <-out
}

// findVersionChild returns the direct child of cmd named "version", or nil.
func findVersionChild(cmd *cobra.Command) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == "version" {
			return c
		}
	}
	return nil
}

func TestNewVersionCmd_Use(t *testing.T) {
	cmd := newVersionCmd()
	if cmd.Use != "version" {
		t.Errorf("newVersionCmd().Use = %q, want %q", cmd.Use, "version")
	}
}

func TestNewVersionCmd_ReturnsDistinctPointer(t *testing.T) {
	a := newVersionCmd()
	b := newVersionCmd()
	if a == b {
		t.Error("newVersionCmd() returned the same *cobra.Command for two calls")
	}
}

func TestVersionChildrenDistinctAcrossRoots(t *testing.T) {
	root := findVersionChild(RootCmd)
	config := findVersionChild(ConfigCmd)
	database := findVersionChild(DatabaseCmd)

	if root == nil {
		t.Fatal("RootCmd has no version child")
	}
	if config == nil {
		t.Fatal("ConfigCmd has no version child")
	}
	if database == nil {
		t.Fatal("DatabaseCmd has no version child")
	}

	if root == config || root == database || config == database {
		t.Error("version children share a *cobra.Command pointer across roots")
	}
}

func TestPrintVersion_WritesBuildInfo(t *testing.T) {
	got := captureStdout(t, func() {
		printVersion()
	})

	for _, want := range []string{"Version:", "Build Date:", "Commit ID:"} {
		if !strings.Contains(got, want) {
			t.Errorf("printVersion output missing %q: %q", want, got)
		}
	}
}

func TestVersionCmd_RunE_ReturnsNil(t *testing.T) {
	cmd := newVersionCmd()

	var err error
	captureStdout(t, func() {
		err = cmd.RunE(cmd, nil)
	})

	if err != nil {
		t.Fatalf("version command RunE returned error: %v", err)
	}
}

func TestNewVersionCmd_HelpNamesBinaryAndFlags(t *testing.T) {
	root := &cobra.Command{
		Use:           "cli",
		Short:         "main app",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "config file")
	root.AddCommand(newVersionCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("executing version --help: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "cli version") {
		t.Errorf("help output does not name the binary and command %q: %q", "cli version", got)
	}
	if !strings.Contains(got, "--config") {
		t.Errorf("help output does not list inherited global flag --config: %q", got)
	}
}

func TestConfigAndDatabaseVersionHelpNameTheirBinary(t *testing.T) {
	tests := []struct {
		name    string
		root    *cobra.Command
		wantUse string
	}{
		{name: "config", root: ConfigCmd, wantUse: "cli-config version"},
		{name: "database", root: DatabaseCmd, wantUse: "cli-migrations version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := findVersionChild(tt.root)
			if child == nil {
				t.Fatalf("%s: no version child", tt.name)
			}

			var buf bytes.Buffer
			child.SetOut(&buf)
			child.SetErr(&buf)
			t.Cleanup(func() { child.SetArgs(nil) })

			child.HelpFunc()(child, nil)

			got := buf.String()
			if !strings.Contains(got, tt.wantUse) {
				t.Errorf("help output does not name %q: %q", tt.wantUse, got)
			}
			if !strings.Contains(got, "--config") {
				t.Errorf("help output missing --config: %q", got)
			}
			if !strings.Contains(got, "--data-dir") {
				t.Errorf("help output missing --data-dir: %q", got)
			}
		})
	}
}
