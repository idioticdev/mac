package main

func runHooks(cfg *Config) {
	if len(cfg.Hooks.PostInstall) == 0 {
		return
	}

	Banner("Post-Install Hooks")

	for _, cmd := range cfg.Hooks.PostInstall {
		Info("Running: " + cmd)
		if _, err := RunShell(cmd); err != nil {
			Warn("Non-zero exit (continuing): " + err.Error())
		} else {
			Ok("Done")
		}
	}
}
