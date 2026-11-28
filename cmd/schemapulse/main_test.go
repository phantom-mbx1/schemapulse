package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "schemapulse")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	// Use the actual executable to exercise argument parsing, both output
	// streams and os.Exit, without replacing process globals in the test.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}

	fixture := func(name string) string {
		t.Helper()
		path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		return path
	}
	safe, risky := fixture("safe"), fixture("risky")

	run := func(t *testing.T, wantExit int, args ...string) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = t.TempDir() // No user config can be discovered here.
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		exitCode := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || ctx.Err() != nil {
				t.Fatalf("%q did not finish: %v", args, err)
			}
			exitCode = exit.ExitCode()
		}
		if exitCode != wantExit {
			t.Fatalf("%q: want exit %d, got %d\nstdout: %s\nstderr: %s", args, wantExit, exitCode, &stdout, &stderr)
		}
		return stdout.String(), stderr.String()
	}

	for _, tc := range []struct {
		name string
		args []string
		code int
		out  string
		err  string
	}{
		{"help", []string{"--help"}, 0, "Usage:", ""},
		{"version", []string{"--version"}, 0, "schemapulse " + version, ""},
		{"rules", []string{"rules"}, 0, "PG001", ""},
		{"inspect", []string{"inspect", safe}, 0, "Sequence check passed", ""},
		{"no command", nil, 2, "Usage:", ""},
		{"unknown command", []string{"does-not-exist"}, 2, "Usage:", "Unknown command"},
		{"safe migration", []string{"check", safe}, 0, "Result: PASS", ""},
		{"threshold breach", []string{"check", risky}, 1, "Result: FAIL", ""},
		{"threshold override", []string{"check", "--fail-on", "none", risky}, 0, "Result: PASS", ""},
		{"markdown", []string{"check", "--format", "markdown", safe}, 0, "# SchemaPulse", ""},
		{"unsupported format", []string{"check", "--format", "other", safe}, 2, "", "unsupported output format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut := run(t, tc.code, tc.args...)
			assertOutput := func(stream, got, want string) {
				t.Helper()
				if (want == "" && got != "") || (want != "" && !strings.Contains(got, want)) {
					t.Errorf("%s: want %q, got %q", stream, want, got)
				}
			}
			assertOutput("stdout", out, tc.out)
			assertOutput("stderr", errOut, tc.err)
		})
	}

	t.Run("only and skip filter JSON findings", func(t *testing.T) {
		out, errOut := run(t, 1, "check", "--format", "json", "--only", "PG001, PG002", "--skip", "PG001", risky)
		if errOut != "" {
			t.Errorf("unexpected stderr: %q", errOut)
		}
		var report struct {
			Findings []struct {
				RuleID string `json:"ruleId"`
			} `json:"findings"`
			ThresholdBreach bool `json:"thresholdBreach"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("decode JSON report: %v\n%s", err, out)
		}
		if len(report.Findings) != 1 || report.Findings[0].RuleID != "PG002" || !report.ThresholdBreach {
			t.Fatalf("want a PG002-only report with a threshold breach, got %+v", report)
		}
	})

	t.Run("init preserves existing configuration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		run(t, 0, "init", "--output", path)
		first, err := os.ReadFile(path)
		if err != nil || !json.Valid(first) {
			t.Fatalf("init did not write valid JSON: %v\n%s", err, first)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		run(t, 2, "init", "--output", path)
		if got, err := os.ReadFile(path); err != nil || string(got) != "{}\n" {
			t.Fatalf("init changed existing config: %v\n%s", err, got)
		}
		run(t, 0, "init", "--force", "--output", path)
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, first) {
			t.Fatalf("forced init did not restore starter config: %v\n%s", err, got)
		}
	})
}
