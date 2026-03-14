package main

func applyMachine(cfg *Config) {
	if cfg.Machine.ComputerName == "" && cfg.Machine.LocalHostname == "" {
		return
	}

	Banner("Machine Identity")

	if name := cfg.Machine.ComputerName; name != "" {
		Info("Setting ComputerName → " + name)
		if _, err := RunSudo("scutil", "--set", "ComputerName", name); err != nil {
			Fail("Could not set ComputerName: " + err.Error())
		}
		if _, err := RunSudo("scutil", "--set", "HostName", name); err != nil {
			Fail("Could not set HostName: " + err.Error())
		}
		Ok("ComputerName / HostName set")
	}

	if hostname := cfg.Machine.LocalHostname; hostname != "" {
		Info("Setting LocalHostName → " + hostname)
		if _, err := RunSudo("scutil", "--set", "LocalHostName", hostname); err != nil {
			Fail("Could not set LocalHostName: " + err.Error())
		}
		Ok("LocalHostName set")
	}
}
