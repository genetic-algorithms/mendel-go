// Package  main is main program of the golang version of mendel's accountant.
// It handles cmd line args, reads input files, handles restarts, and contains the main generation loop.

/* Order of todos:
- add global function ptr struct, based in input params
- support fixed mutation effect
- combine mutation effects according to Multiplicative_weighting
- review and merge jon's random number generator PR
- output a simple file such as the .hst file with generation number, number of del., neu., fav. mutations, and pop size
- (jon) apply correct Weibull fitness factor distribution to mutation - see wes' comments in slack
- (jon) the num mutations should be a poisson distribution with the mean being Mutn_rate
- combine fitness factors appropriately
- figure out how chromosomes are used
- run with more linkage blocks
 */
package main

import (
	"log"
	"os"

	// "github.com/davecgh/go-spew/spew"

	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"math"
	"bitbucket.org/geneticentropy/mendel-go/pop"
	"bitbucket.org/geneticentropy/mendel-go/random"
	"math/rand"
)

// Initialize initializes variables, objects, and settings for either an initial run or a restart.
func initialize() {
	utils.Verbose(5, "Initializing...\n")

	//todo: support parallel
	//todo: open intermediate files if necessary

	// Adjust certain config values
	config.Cfg.Selection.Heritability = math.Max(1.e-20, config.Cfg.Selection.Heritability)   // Limit the minimum value of heritability to be 10**-20
	if config.Cfg.Mutations.Fraction_neutral == 0 { config.Cfg.Computation.Track_neutrals = false }   // no neutrals to track
	if config.Cfg.Computation.Track_neutrals { config.Cfg.Computation.Tracking_threshold = 0 }
	if config.Cfg.Mutations.Allow_back_mutn { config.Cfg.Computation.Tracking_threshold = 0 }  // If back mutations are allowed, set the tracking threshold to zero so that all mutations are tracked

	// Read restart file, if specified
	config.ReadRestartFile("")

	if config.Cfg.Computation.Random_number_seed != 0 {
		random.Rnd = rand.New(rand.NewSource(config.Cfg.Computation.Random_number_seed))
	} else {
		random.Rnd = rand.New(rand.NewSource(random.GetSeed()))
	}

	//todo: complete this function
}

// Shutdown does all the stuff necessary at the end of the run.
func shutdown(population *pop.Population) {
	utils.Verbose(2, "Shutting down...\n")
	population.Report(true)
}

// Main handles cmd line args, reads input files, handles restarts, and contains the main generation loop.
func main() {
	log.SetOutput(os.Stdout) 	// needs to be done very early

	config.ReadCmdArgs()    // Get/check cmd line options and load specified input file - flags are accessible in config.CmdArgs, config values in config.Cfg

	// Handle the different input file choices
	if config.CmdArgs.InputFileToCreate != "" {
		if err := utils.CopyFile(config.DEFAULT_INPUT_FILE, config.CmdArgs.InputFileToCreate); err != nil { log.Fatalln(err) }
		os.Exit(0)

	} else if config.CmdArgs.InputFile != ""{
		if err := config.ReadFromFile(config.CmdArgs.InputFile); err != nil { log.Fatalln(err) }
		utils.Verbose(3, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else { config.Usage(0) }

	// Initialize
	log.Println("Running mendel simulation...")
	initialize()
	population := pop.PopulationFactory(config.Cfg.Basic.Pop_size) 		// time 0 population
	population.Initialize()

	// Main generation loop. config.Restart.Gen_0 allows us to restart some number of generations into the simulation.
	for gen := config.Restart.Gen_0+1; gen <= config.Restart.Gen_0+config.Cfg.Basic.Num_generations; gen++ {
		utils.Verbose(1, "Generation %d\n", gen)
		population = population.Mate()
		population.Select()
		population.Report(false)
	}

	// Finish up
	shutdown(population)
}
