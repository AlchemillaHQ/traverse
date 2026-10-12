package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--not-a-real-flag"},
		{"-T", "not-a-number"},
		{"-b=not-a-bool"},
		{"-s"},
		{"unexpected-argument"},
	} {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("parseFlags(%q) accepted invalid arguments", args)
		}
	}
	cfg, err := parseFlags([]string{"-s", "stun.example.com", "-t", "turn.example.com", "-P", "TCP", "-u", "user", "-p", "password", "-o", "JSON"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.stunServer != "stun.example.com:3478" || cfg.turnServer != "turn.example.com:3478" || cfg.protocol != "tcp" || cfg.outputFormat != "json" {
		t.Fatalf("unexpected parsed config: %+v", cfg)
	}
}

func TestEnsurePort(t *testing.T) {
	for _, tt := range []struct{ address, want string }{
		{"", ""},
		{"stun.example.com", "stun.example.com:3478"},
		{"stun.example.com:19302", "stun.example.com:19302"},
		{"127.0.0.1", "127.0.0.1:3478"},
		{"::1", "[::1]:3478"},
		{"[::1]", "[::1]:3478"},
		{"[::1]:19302", "[::1]:19302"},
	} {
		if got := ensurePort(tt.address, "3478"); got != tt.want {
			t.Errorf("ensurePort(%q) = %q, want %q", tt.address, got, tt.want)
		}
	}
}

func TestInvalidLocalIP(t *testing.T) {
	cfg, err := parseFlags([]string{"-s", "127.0.0.1", "-l", "not-an-ip"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.validate(); err == nil {
		t.Fatal("invalid local IP was accepted")
	}
}

// Exercise main in a subprocess so these tests verify actual exit codes.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("TRAVERSE_TEST_CLI_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"traverse"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(2)
}

func TestCLIExitStatus(t *testing.T) {
	for _, tt := range []struct {
		name   string
		args   []string
		status int
		output string
	}{
		{"unknown flag", []string{"--not-a-real-flag"}, 1, "flag provided but not defined"},
		{"invalid number", []string{"-T", "not-a-number"}, 1, "parse error"},
		{"invalid boolean", []string{"-b=not-a-bool"}, 1, "parse error"},
		{"missing value", []string{"-s"}, 1, "flag needs an argument"},
		{"unexpected argument", []string{"unexpected-argument"}, 1, "unexpected argument"},
		{"help", []string{"--help"}, 0, "USAGE"},
		{"short help", []string{"-h"}, 0, "USAGE"},
		{"version", []string{"--version"}, 0, "traverse v"},
		{"welcome", nil, 0, "Quick start"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			args := append([]string{"-test.run=^TestCLIProcess$", "--"}, tt.args...)
			cmd := exec.CommandContext(ctx, os.Args[0], args...)
			cmd.Env = append(os.Environ(), "TRAVERSE_TEST_CLI_PROCESS=1", "GORACE=atexit_sleep_ms=0")
			out, err := cmd.CombinedOutput()
			if cmd.ProcessState == nil {
				t.Fatalf("could not run CLI: %v", err)
			}
			if status := cmd.ProcessState.ExitCode(); status != tt.status {
				t.Fatalf("exit status = %d, want %d; output: %s", status, tt.status, out)
			}
			if !strings.Contains(string(out), tt.output) {
				t.Errorf("output %q does not contain %q", out, tt.output)
			}
		})
	}
}
