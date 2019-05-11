package config

import (
	"log"
	"flag"
	"os"
	"fmt"
	"strings"
)

const DEFAULTS_INPUT_FILE = "mendel-defaults.ini"
var DEFAULTS_INPUT_DIRS = []string{"/usr/local/share/mendel-go"}

func Usage(exitCode int) {
	usageStr1 := `Usage:
  mendel-go -f <filename> [-D <defaults-path>] [-O <data-path>] [-z] [-u <SPC-username>]
  mendel-go -d [-D <defaults-path>] [-O <data-path>] [-z] [-u <SPC-username>]
  mendel-go -c <filename> [-D <defaults-path>] [-O <data-path>]
  mendel-go -V

Performs a mendel run...

Options:
`

	usageStr2 := `
Examples:
  mendel-go -f /home/bob/mendel.in    # run with this input file
  mendel-go -d     # run with all default parameters from `+ DEFAULTS_INPUT_FILE +`
  mendel-go -c /home/bob/mendel.in    # create an input file primed with defaults, then you can edit it
`

	//if exitCode > 0 {
	fmt.Fprintf(os.Stderr, usageStr1)		// send it to stderr
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, usageStr2)		// send it to stderr
	//} else {
	//	fmt.Println(usageStr1)		// send it to stdout
	//	flag.PrintDefaults()		// do not yet know how to get this to print to stdout
	//	fmt.Println(usageStr2)		// send it to stdout
	//}
	os.Exit(exitCode)
}

type CommandArgs struct {
	InputFile, InputFileToCreate, DefaultFile, DataPath, SPCusername string
	CreateZip, Version bool
}

// CmdArgs is the singleton instance of CommandArgs that can be accessed throughout the mendel code.
// It gets set in ReadCmdArgs().
var CmdArgs *CommandArgs

// ReadCmdArgs reads the command line args/flag, checks them, and puts them into the Config struct. Will exit if user input error.
// This is also the factory method for the CommandArgs class and will store the created instance in this packages CmdArgs var.
func ReadCmdArgs() {
	//log.Println("Reading command line arguments and flags...") 	// can not use verbosity here because we have not read the config file yet
	CmdArgs = &CommandArgs{} 		// create and set the singleton config
	var useDefaults bool
	flag.StringVar(&CmdArgs.InputFile, "f", "", "Run mendel-go with this input file (backed by the defaults file)")
	flag.StringVar(&CmdArgs.DefaultFile, "D", "", "Path to the defaults file. If not set, looks for "+DEFAULTS_INPUT_FILE+" in the current directory, the directory of the executable, and "+strings.Join(DEFAULTS_INPUT_DIRS,", "))
	flag.StringVar(&CmdArgs.DataPath, "O", "", "Path to put the output data files in. If not set, the data_file_path in the input config file or defaults file is used.")
	flag.StringVar(&CmdArgs.SPCusername, "u", "", "Create a zip of the output for this SPC username, suitable for importing into SPC for data visualization.")
	flag.StringVar(&CmdArgs.InputFileToCreate, "c", "", "Create a mendel input file (using default values) and then exit")
	flag.BoolVar(&useDefaults, "d", false, "Run mendel with all default parameters")
	flag.BoolVar(&CmdArgs.CreateZip, "z", false, "Create a zip of the output, suitable for importing into the Mendel web UI for data visualization")
	flag.BoolVar(&CmdArgs.Version, "V", false, "Display version and exit")
	flag.Usage = func() { Usage(0) }
	flag.Parse()
	// can use this to get values anywhere in the program: flag.Lookup("name").Value.String()
	// spew.Dump(flag.Lookup("f").Value.String())

	if CmdArgs.InputFileToCreate != "" {
		if CmdArgs.InputFile != "" || useDefaults { log.Println("Error: if you specify -c you can not specify either -f or -d"); Usage(1) }

	} else if useDefaults {
		if CmdArgs.InputFile != "" || CmdArgs.InputFileToCreate != "" { log.Println("Error: if you specify -d you can not specify either -f or -c"); Usage(1) }
		CmdArgs.InputFile = FindDefaultFile()

	} else if CmdArgs.InputFile != ""{
		// We already verified inputFileToCreate or useDefaults was not specified with this

	} else if !CmdArgs.Version { Usage(0) }
}
