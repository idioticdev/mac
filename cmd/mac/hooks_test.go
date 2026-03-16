package main

import (
	"errors"
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

func TestRunHooks_Empty(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	runHooks(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls for empty hooks, got %v", r.Calls())
	}
}

func TestRunHooks_SingleHook(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.WhenShell("echo hello").Returns("hello", nil)

	cfg := &Config{
		Hooks: HooksConfig{
			PostInstall: []string{"echo hello"},
		},
	}

	runHooks(cfg, r)

	if !r.CalledWithShell("echo hello") {
		t.Error("expected RunShell('echo hello') to be called")
	}
}

func TestRunHooks_MultipleHooksInOrder(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Hooks: HooksConfig{
			PostInstall: []string{"echo first", "echo second", "echo third"},
		},
	}

	runHooks(cfg, r)

	calls := r.Calls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(calls), calls)
	}
	if calls[0] != "shell:echo first" {
		t.Errorf("calls[0]: want shell:echo first, got %q", calls[0])
	}
	if calls[1] != "shell:echo second" {
		t.Errorf("calls[1]: want shell:echo second, got %q", calls[1])
	}
	if calls[2] != "shell:echo third" {
		t.Errorf("calls[2]: want shell:echo third, got %q", calls[2])
	}
}

func TestRunHooks_ContinuesOnFailure(t *testing.T) {
	r := testutil.NewFakeRunner()
	r.WhenShell("exit 1").Returns("", errors.New("exit status 1"))

	cfg := &Config{
		Hooks: HooksConfig{
			PostInstall: []string{"exit 1", "echo after"},
		},
	}

	// Should not panic or stop at the failing hook.
	runHooks(cfg, r)

	if !r.CalledWithShell("echo after") {
		t.Error("expected second hook to run even after first failed")
	}
}

func TestRunHooks_AllSucceed(t *testing.T) {
	r := testutil.NewFakeRunner()
	for _, cmd := range []string{"touch /tmp/a", "touch /tmp/b"} {
		r.WhenShell(cmd).Returns("", nil)
	}

	cfg := &Config{
		Hooks: HooksConfig{
			PostInstall: []string{"touch /tmp/a", "touch /tmp/b"},
		},
	}

	runHooks(cfg, r)

	for _, cmd := range []string{"touch /tmp/a", "touch /tmp/b"} {
		if !r.CalledWithShell(cmd) {
			t.Errorf("expected hook %q to be called", cmd)
		}
	}
}
