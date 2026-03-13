package hooks

import (
	"github.com/youruser/macsetup/internal/config"
	runexec "github.com/youruser/macsetup/internal/exec"
	"github.com/youruser/macsetup/internal/ui"
)

func Run(cfg *config.Config) {
	if len(cfg.Hooks.PostInstall) == 0 {
		return
	}

	ui.Banner("Post-Install Hooks")

	for _, cmd := range cfg.Hooks.PostInstall {
		ui.Info("Running: " + cmd)
		if _, err := runexec.RunShell(cmd); err != nil {
			ui.Warn("Non-zero exit (continuing): " + err.Error())
		} else {
			ui.Ok("Done")
		}
	}
}
