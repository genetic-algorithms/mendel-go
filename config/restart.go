package config

// Note: this file is not currently used

import "log"

type RestartValues struct {
	Gen_0 uint32
}

var Restart *RestartValues

// ReadRestartFile loads values from a restart file. If filename == "", it creates values appropriate for the non-restart case.
// This is also the factory method for the RestartValues class and will store the created instance in this packages Restart var.
func ReadRestartFile(filename string) error {
	if filename == "" {
		Restart = &RestartValues{Gen_0: 0}
	} else {
		// Load the restart file
		log.Fatalln("Error: loading a restart file is not yet supported.")
	}
	return nil
}