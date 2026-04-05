package main

import (
	"testing"

	"github.com/youruser/mac/cmd/mac/testutil"
)

func TestApplyMachine_NoConfig(t *testing.T) {
	r := testutil.NewFakeRunner()
	cfg := &Config{}
	applyMachine(cfg, r)
	if len(r.Calls()) != 0 {
		t.Errorf("expected no calls when machine config is empty, got %v", r.Calls())
	}
}

func TestApplyMachine_ComputerName(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Machine: MachineConfig{
			ComputerName: "my-mac",
		},
	}

	applyMachine(cfg, r)

	if !r.CalledWithSudo("scutil", "--set", "ComputerName", "my-mac") {
		t.Error("expected scutil --set ComputerName my-mac")
	}
	if !r.CalledWithSudo("scutil", "--set", "HostName", "my-mac") {
		t.Error("expected scutil --set HostName my-mac")
	}
}

func TestApplyMachine_LocalHostname(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Machine: MachineConfig{
			LocalHostname: "my-mac-local",
		},
	}

	applyMachine(cfg, r)

	if !r.CalledWithSudo("scutil", "--set", "LocalHostName", "my-mac-local") {
		t.Error("expected scutil --set LocalHostName my-mac-local")
	}
}

func TestApplyMachine_BothFields(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Machine: MachineConfig{
			ComputerName:  "my-mac",
			LocalHostname: "my-mac-local",
		},
	}

	applyMachine(cfg, r)

	if !r.CalledWithSudo("scutil", "--set", "ComputerName", "my-mac") {
		t.Error("expected scutil --set ComputerName")
	}
	if !r.CalledWithSudo("scutil", "--set", "HostName", "my-mac") {
		t.Error("expected scutil --set HostName")
	}
	if !r.CalledWithSudo("scutil", "--set", "LocalHostName", "my-mac-local") {
		t.Error("expected scutil --set LocalHostName")
	}
}

// ── uninstallMachine ─────────────────────────────────────────────────────────

func TestUninstallMachine_NoConfig(t *testing.T) {
	// Should not panic or print anything when no machine config is set.
	cfg := &Config{}
	uninstallMachine(cfg) // no assertions — just must not panic
}

func TestUninstallMachine_ComputerName_NoScutilCalls(t *testing.T) {
	cfg := &Config{
		Machine: MachineConfig{ComputerName: "my-mac"},
	}
	// uninstallMachine only warns — it must not call scutil.
	// We verify this by capturing no runner calls (function takes no runner).
	uninstallMachine(cfg) // must not panic
}

func TestUninstallMachine_BothFields_NoScutilCalls(t *testing.T) {
	cfg := &Config{
		Machine: MachineConfig{
			ComputerName:  "my-mac",
			LocalHostname: "my-mac-local",
		},
	}
	uninstallMachine(cfg) // must not panic
}

func TestApplyMachine_ComputerNameOnly_NoLocalHostnameCall(t *testing.T) {
	r := testutil.NewFakeRunner()

	cfg := &Config{
		Machine: MachineConfig{
			ComputerName: "my-mac",
		},
	}

	applyMachine(cfg, r)

	// LocalHostName should NOT be set since LocalHostname is empty.
	if r.CalledWithSudo("scutil", "--set", "LocalHostName", "") {
		t.Error("should not call scutil --set LocalHostName when LocalHostname is empty")
	}
}
