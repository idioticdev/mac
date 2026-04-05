package main

func uninstallMachine(cfg *Config) {
	if cfg.Machine.ComputerName == "" && cfg.Machine.LocalHostname == "" {
		return
	}

	Banner("Machine Identity")
	Warn("Machine identity was set by mac apply — no automatic revert.")
	if cfg.Machine.ComputerName != "" {
		Warn("  ComputerName/HostName: run 'sudo scutil --set ComputerName <name>' to change")
	}
	if cfg.Machine.LocalHostname != "" {
		Warn("  LocalHostName: run 'sudo scutil --set LocalHostName <name>' to change")
	}
}

func applyMachine(cfg *Config, r Runner) {
	if cfg.Machine.ComputerName == "" && cfg.Machine.LocalHostname == "" {
		return
	}

	Banner("Machine Identity")

	if name := cfg.Machine.ComputerName; name != "" {
		Info("Setting ComputerName → " + name)
		if _, err := r.RunSudo("scutil", "--set", "ComputerName", name); err != nil {
			Fail("Could not set ComputerName: " + err.Error())
		}
		if _, err := r.RunSudo("scutil", "--set", "HostName", name); err != nil {
			Fail("Could not set HostName: " + err.Error())
		}
		Ok("ComputerName / HostName set")
	}

	if hostname := cfg.Machine.LocalHostname; hostname != "" {
		Info("Setting LocalHostName → " + hostname)
		if _, err := r.RunSudo("scutil", "--set", "LocalHostName", hostname); err != nil {
			Fail("Could not set LocalHostName: " + err.Error())
		}
		Ok("LocalHostName set")
	}
}
