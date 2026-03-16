package main

func runHooks(cfg *Config, r Runner) {
	if len(cfg.Hooks.PostInstall) == 0 {
		return
	}

	Banner("Post-Install Hooks")

	for _, cmd := range cfg.Hooks.PostInstall {
		Info("Running: " + cmd)
		if _, err := r.RunShell(cmd); err != nil {
			Warn("Non-zero exit (continuing): " + err.Error())
		} else {
			Ok("Done")
		}
	}
}
