// Package  main is main program of the golang version of mendel's accountant.
// It handles cmd line args, reads input files, handles restarts, and contains the main generation loop.
package main

import (
	"fmt"
	"flag"
	"log"
	"os"

	// "github.com/davecgh/go-spew/spew"

	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)

const DEFAULT_INPUT_FILE = "./mendel-defaults.ini"

func usage(exitCode int) {
	usageStr1 := `Usage:
  mendel {-c | -f} [filename]
  mendel -d

Performs a mendel run...

Options:
`

	usageStr2 := `
Examples:
  mendel -f /home/bob/mendel.in    # run with this input file
  mendel -d     # run with all default parameters from `+DEFAULT_INPUT_FILE+`
  mendel -c /home/bob/mendel.in    # create an input file primed with defaults, then you can edit it
`

	//if exitCode > 0 {
		fmt.Fprintln(os.Stderr, usageStr1)		// send it to stderr
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, usageStr2)		// send it to stderr
	//} else {
	//	fmt.Println(usageStr1)		// send it to stdout
	//	flag.PrintDefaults()		//todo: do not yet know how to get this to print to stdout
	//	fmt.Println(usageStr2)		// send it to stdout
	//}
	os.Exit(exitCode)
}

// Init initializes variables, objects, and settings for either an initial run or a restart.
func init() {
	log.SetOutput(os.Stdout)
}

// Main handles cmd line args, reads input files, handles restarts, and contains the main generation loop.
func main() {
	// Get and check cmd line options
	var inputFile, inputFileToCreate string
	var useDefaults bool
	flag.StringVar(&inputFile, "f", "", "Run mendel with this input file")
	flag.StringVar(&inputFileToCreate, "c", "", "Create a mendel input file (using default values) and then exit")
	flag.BoolVar(&useDefaults, "d", false, "Run mendel with all default parameters")
	flag.Usage = func() { usage(0) }
	flag.Parse()
	// can use this to get values anywhere in the program: flag.Lookup("name").Value.String()
	// spew.Dump(flag.Lookup("f").Value.String())
	// os.Exit(0)
	// if name == "" && !isStdinFile() { usage(0) }

	cfg := config.Config{}
	if inputFileToCreate != "" {
		if inputFile != "" || useDefaults { log.Println("Error: if you specify -c you can not specify either -f or -d"); usage(1) }
		if err := utils.CopyFile(DEFAULT_INPUT_FILE, inputFileToCreate); err != nil { log.Fatalln(err) }
		os.Exit(0)

	} else if useDefaults {
		if inputFile != "" || inputFileToCreate != "" { log.Println("Error: if you specify -d you can not specify either -f or -c"); usage(1) }
		if err := cfg.ReadFromFile(DEFAULT_INPUT_FILE); err != nil { log.Fatalln(err) }
		utils.Verbose(9, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else if inputFile != ""{
		// We already verified inputFileToCreate or useDefaults was not specified with this
		if err := cfg.ReadFromFile(inputFile); err != nil { log.Fatalln(err) }
		utils.Verbose(9, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else { usage(0) }


	log.Println("Running mendel...")

}
