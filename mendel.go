// Package  main is main program of the golang version of mendel's accountant.
// It handles cmd line args, reads input files, and contains the main generation loop.

/* Order of todos:
- (bruce) Multiple threads (go routines)
- (bruce) Add a new key `populationSize` to alleles-count.json, and stop supporting alleles.json
- (bruce) Upgrade go on ec2 machines, run large runs, and compare results with mendel-f90
- (bruce) Improve initial alleles: 1) don't count twice in same individual, 2) imitate diagnostics.f90:1351 ff which uses variable MNP to limit the number of alleles to 100000 for statistical sampling and normalizing that graph to be so we don't report hard numbers but ratios- (bruce) compare run results with mendel-f90
- (jon) Population growth
- (jon) Bottleneck
- (jon) Tribes
- add stats for length of time mutations have been in population (for both eliminated indivs and current pop)
- support num offspring proportional to fitness (fitness_dependent_fertility in mendel-f90)
- stop execution when any of these are reached: extinction_threshold, max_del_mutn_per_indiv, max_neu_mutn_per_indiv, max_fav_mutn_per_indiv
 */
package main

import (
	"log"
	"os"
	// "github.com/davecgh/go-spew/spew"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"bitbucket.org/geneticentropy/mendel-go/pop"
	"bitbucket.org/geneticentropy/mendel-go/random"
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/dna"
	"github.com/pkg/profile"
	"strings"
)

// Initialize initializes variables, objects, and settings.
func initialize() *rand.Rand {
	config.Verbose(5, "Initializing...\n")

	utils.MeasurerFactory(config.Cfg.Computation.Verbosity)

	// Set all of the function ptrs for the algorithms we want to use.
	dna.SetModels(config.Cfg)
	pop.SetModels(config.Cfg)

	// Initialize random number generator
	var uniformRandom *rand.Rand
	if config.Cfg.Computation.Random_number_seed != 0 {
		uniformRandom = rand.New(rand.NewSource(config.Cfg.Computation.Random_number_seed))
	} else {
		uniformRandom = rand.New(rand.NewSource(random.GetSeed()))
	}

	return uniformRandom
}

// Shutdown does all the stuff necessary at the end of the run.
func shutdown() {
	config.Verbose(5, "Shutting down...\n")
	config.FMgr.CloseFiles()
}

// Main handles cmd line args, reads input files, and contains the main generation loop.
func main() {
	log.SetOutput(os.Stdout) 	// needs to be done very early

	config.ReadCmdArgs()    // Get/check cmd line options and load specified input file - flags are accessible in config.CmdArgs, config values in config.Cfg

	// Handle the different input file choices
	if config.CmdArgs.InputFileToCreate != "" {
		if err := utils.CopyFile(config.FindDefaultFile(), config.CmdArgs.InputFileToCreate); err != nil { log.Fatalln(err) }
		os.Exit(0)

	} else if config.CmdArgs.InputFile != ""{
		if err := config.ReadFromFile(config.CmdArgs.InputFile); err != nil { log.Fatalln(err) }
		config.Verbose(3, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else { config.Usage(0) }

	// Initialize
	switch strings.ToLower(config.Cfg.Computation.Performance_profile) {
	case "":
		// no profiling, do nothing
	case "cpu":
		defer profile.Start(profile.CPUProfile, profile.ProfilePath("./pprof")).Stop()
	case "mem":
		defer profile.Start(profile.MemProfile, profile.ProfilePath("./pprof")).Stop()
	case "block":
		defer profile.Start(profile.BlockProfile, profile.ProfilePath("./pprof")).Stop()
	default:
		log.Fatalf("Error: unrecognized value for performance_profile: %v", config.Cfg.Computation.Performance_profile)
	}
	uniformRandom := initialize()
	population := pop.PopulationFactory(config.Cfg.Basic.Pop_size, true) 		// time 0 population
	population.GenerateInitialAlleles(uniformRandom)
	population.ReportInitial(config.Cfg.Basic.Num_generations)

	// Main generation loop.
	for gen := uint32(1); gen <= config.Cfg.Basic.Num_generations; gen++ {
		population = population.Mate(uniformRandom)
		population.Select(uniformRandom)

		if (pop.RecombinationType(config.Cfg.Population.Recombination_model) == pop.FULL_SEXUAL && population.GetCurrentSize() < 2) || population.GetCurrentSize() == 0 {
			log.Println("Population is extinct. Stopping simulation.")
			break
		}

		population.ReportEachGen(gen)
	}

	// Finish up
	//population.ReportFinal(config.Cfg.Basic.Num_generations)
	shutdown()
}
