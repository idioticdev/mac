package machine

import (
	"github.com/youruser/macsetup/internal/config"
	runexec "github.com/youruser/macsetup/internal/exec"
	"github.com/youruser/macsetup/internal/ui"
)

func Apply(cfg *config.Config) {
	if cfg.Machine.ComputerName == "" && cfg.Machine.LocalHostname == "" {
		return
	}

	ui.Banner("Machine Identity")

	if name := cfg.Machine.ComputerName; name != "" {
		ui.Info("Setting ComputerName → " + name)
		if _, err := runexec.RunSudo("scutil", "--set", "ComputerName", name); err != nil {
			ui.Fail("Could not set ComputerName: " + err.Error())
		}
		if _, err := runexec.RunSudo("scutil", "--set", "HostName", name); err != nil {
			ui.Fail("Could not set HostName: " + err.Error())
		}
		ui.Ok("ComputerName / HostName set")
	}

	if hostname := cfg.Machine.LocalHostname; hostname != "" {
		ui.Info("Setting LocalHostName → " + hostname)
		if _, err := runexec.RunSudo("scutil", "--set", "LocalHostName", hostname); err != nil {
			ui.Fail("Could not set LocalHostName: " + err.Error())
		}
		ui.Ok("LocalHostName set")
	}
}
