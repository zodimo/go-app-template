package commands

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/exitcode"
)

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// everything fn wrote to stderr. ExecuteCmd writes directly to os.Stderr, so
// this (rather than cmd.SetErr) is the only way to observe its output.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stderr = w

	out := make(chan string, 1)
	go func() {
		defer r.Close()
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		out <- buf.String()
	}()

	fn()

	os.Stderr = orig
	if err := w.Close(); err != nil {
		t.Logf("closing pipe writer: %v", err)
	}
	return <-out
}

// guardJSONOff pins the package-global viper "json" key to false for the
// duration of the test and restores the previous value afterwards, so JSON
// output mode never leaks across tests.
func guardJSONOff(t *testing.T) {
	t.Helper()
	prev := viper.Get("json")
	viper.Set("json", false)
	t.Cleanup(func() {
		viper.Set("json", prev)
	})
}

// newExecuteTestCmd builds a throwaway cobra command whose RunE is fully
// controlled by the test. SilenceUsage/SilenceErrors keep cobra itself from
// writing anything, so ExecuteCmd is the sole writer under test.
func newExecuteTestCmd(runE func(cmd *cobra.Command, args []string) error) *cobra.Command {
	return &cobra.Command{
		Use:           "test",
		RunE:          runE,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func TestExecuteCmd_Success(t *testing.T) {
	guardJSONOff(t)

	cmd := newExecuteTestCmd(func(cmd *cobra.Command, args []string) error {
		return nil
	})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	var got uint8
	stderr := captureStderr(t, func() {
		got = ExecuteCmd(cmd)
	})

	if got != 0 {
		t.Errorf("ExecuteCmd() = %d, want 0", got)
	}
	if stderr != "" {
		t.Errorf("ExecuteCmd wrote to stderr on success: %q", stderr)
	}
}

func TestExecuteCmd_OrdinaryError(t *testing.T) {
	guardJSONOff(t)

	cmd := newExecuteTestCmd(func(cmd *cobra.Command, args []string) error {
		return errors.New("boom")
	})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	var got uint8
	stderr := captureStderr(t, func() {
		got = ExecuteCmd(cmd)
	})

	if got != 1 {
		t.Errorf("ExecuteCmd() = %d, want 1", got)
	}
	if n := strings.Count(stderr, "boom"); n != 1 {
		t.Errorf("stderr printed the error %d times (want exactly 1): %q", n, stderr)
	}
}

func TestExecuteCmd_ExitStatus(t *testing.T) {
	guardJSONOff(t)

	cmd := newExecuteTestCmd(func(cmd *cobra.Command, args []string) error {
		return exitcode.NewExitStatus(3)
	})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	var got uint8
	stderr := captureStderr(t, func() {
		got = ExecuteCmd(cmd)
	})

	if got != 3 {
		t.Errorf("ExecuteCmd() = %d, want 3", got)
	}
	if n := strings.Count(stderr, "exit status 3"); n != 1 {
		t.Errorf("stderr printed the exit status %d times (want exactly 1): %q", n, stderr)
	}
}

func TestExecuteCmd_ExitStatusOK(t *testing.T) {
	guardJSONOff(t)

	cmd := newExecuteTestCmd(func(cmd *cobra.Command, args []string) error {
		return exitcode.ExitStatusOK
	})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	var got uint8
	stderr := captureStderr(t, func() {
		got = ExecuteCmd(cmd)
	})

	if got != 0 {
		t.Errorf("ExecuteCmd() = %d, want 0", got)
	}
	if stderr != "" {
		t.Errorf("ExecuteCmd wrote to stderr for ExitStatusOK (regression for the exit status 0 bug): %q", stderr)
	}
}
