/*
Package main is the main program of the golang version of mendel's accountant. It handles cmd line args, reads input files,
and contains the main generation loop of mating and selection.

The species and genome are modelled with this hierarchy of classes:

- Species
 - Populations (tribes)
  - PopulationPart (to enable parallel writes)
   - Individuals
    - Chromosomes
     - LinkageBlocks
      - Mutations
*/

/* Order of todos:
- Compare more runs with mendel-f90
- Tribes
- Figure out the delay at the beginning on r4 (and to a lesser extent on m4)
- (Consider) Add ability for user to specify a change to any input parameter at any generation (need to use reflection)
- Improve initial alleles: imitate diagnostics.f90:1351 ff which uses variable MNP to limit the number of alleles to 100000 for statistical sampling and normalizing that graph to be so we don't report hard numbers but ratios- (bruce) compare run results with mendel-f90
- add stats for length of time mutations have been in population (for both eliminated indivs and current pop)
- (When needed) support num offspring proportional to fitness (fitness_dependent_fertility in mendel-f90)
 */
package main

import (
	"log"
	"os"
	// "github.com/davecgh/go-spew/spew"
	"github.com/genetic-algorithms/mendel-go/config"
	"github.com/genetic-algorithms/mendel-go/utils"
	"github.com/genetic-algorithms/mendel-go/pop"
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/dna"
	"github.com/pkg/profile"
	"strings"
	"runtime/debug"
	"github.com/genetic-algorithms/mendel-go/random"
	"fmt"
	"path/filepath"
	"io"
)

// Initialize initializes variables, objects, and settings.
func initialize() *rand.Rand {
	config.Verbose(5, "Initializing...\n")

	if config.Cfg.Computation.Force_gc {
		debug.SetGCPercent(-1)
	}

	utils.MeasurerFactory(config.Cfg.Computation.Verbosity)
	utils.Measure.Start("Total")

	utils.GlobalUniqueIntFactory()

	// Set all of the function ptrs for the algorithms we want to use.
	dna.SetModels(config.Cfg)
	pop.SetModels(config.Cfg)

	random.NextSeed = config.Cfg.Computation.Random_number_seed
	return random.RandFactory()
}

// CreateSpcZip zips up the output in a form suitable for importing into SPC for data visualization
func CreateSpcZip(spcUsername, randomSlug string) {
	// Write input params to the output dir
	// when running in the spc context we don't want to overwrite the toml file it has already written
	if isEqual, err := utils.CanonicalPathsEqual(config.CmdArgs.InputFile, config.TOML_FILENAME); err != nil && !isEqual {
		if tomlWriter := config.FMgr.GetFile(config.TOML_FILENAME, 0); tomlWriter != nil {
			//if err := config.Cfg.WriteToFile(tomlWriter); err != nil { log.Fatalf("Error writing %s: %v", config.TOML_FILENAME, err) }
			if config.CmdArgs.InputFile != "" {
				if err := utils.CopyFromFileName2Writer(config.CmdArgs.InputFile, tomlWriter); err != nil { log.Fatalln(err) }
			} else {
				if err := utils.CopyFromFileName2Writer(config.FindDefaultFile(), tomlWriter); err != nil { log.Fatalln(err) }
			}
		}
	}

	//todo: write real run output to OUTPUT_FILENAME
	if outputWriter := config.FMgr.GetFile(config.OUTPUT_FILENAME, 0); outputWriter != nil {
		outputStr := "The run log entries are not available in this job.\nThe plot files and inputs ARE available (click PLOT or FILES below).\n"
		if _, err := io.WriteString(outputWriter, outputStr); err != nil { log.Fatalf("Error writing %s: %v", config.OUTPUT_FILENAME, err) }
	}
	config.FMgr.CloseAllFiles()	// explicitly close all files so the zip file contains all data

	// The files in the zip need to have a path like: user_data/<spcUsername>/mendel_go/<randomSlug>/...
	pathToZipUp := config.Cfg.Computation.Data_file_path	// e.g. test/output/short
	zipFilePath := config.Cfg.Computation.Data_file_path+"/../"+config.Cfg.Basic.Case_id+".zip"	// e.g. test/output/short.zip
	prefixToReplace := config.Cfg.Computation.Data_file_path
	newPrefix := "user_data/"+spcUsername+"/mendel_go/"+randomSlug	// e.g. user_data/brucemp/mendel_go/z59e4c
	config.Verbose(2, "Creating zip file: pathToZipUp=%s, zipFilePath=%s, prefixToReplace=%s, newPrefix=%s\n", pathToZipUp, zipFilePath, prefixToReplace, newPrefix)
	if err := utils.CreatePrefixedZip(pathToZipUp, zipFilePath, prefixToReplace, newPrefix); err != nil {
		log.Fatalf("Error creating zip of output data files: %v", err)
	}
	fmt.Printf("Zipped the output data files into %s for SPC username %s and job id %s\n", filepath.Clean(zipFilePath), spcUsername, randomSlug)

	/* this was trying to play games with sym links to get the prefix correct, but the zip file creation functions don't follow sym links...
	dataPath, err := filepath.Abs(config.Cfg.Computation.Data_file_path)	// the dataPath var will be referred to in another dir, so it needs to be absoluted
	if err != nil { log.Fatalf("Error: could not make %s absolute: %v", config.Cfg.Computation.Data_file_path, err) }
	zipDir := config.FMgr.DataFilePath + "/../.spclinks/" + config.Cfg.Basic.Case_id + "/user_data"
	linkDir := zipDir + "/" + spcUsername + "/mendel_go"
	// also: if linkDir already exists, remove any previous sym links in it
	if err := os.MkdirAll(linkDir, 0755); err != nil { log.Fatalf("Error creating link directory %s for zip file creation: %v", linkDir, err) }
	link := linkDir + "/" + randomSlug
	config.Verbose(1, "linking %s to %s", dataPath, link)
	if err := os.Symlink(dataPath, link); err != nil { log.Fatalf("Error creating sym link from %s to %s: %v", dataPath, link, err) }
	zipFile := config.FMgr.DataFilePath + "/../" + config.Cfg.Basic.Case_id + ".zip"	// this makes it a peer to the case dir
	config.Verbose(1, "creating zip file %s of directory %s", zipFile, zipDir)
	if err := archiver.Zip.Make(zipFile, []string{zipDir}); err != nil {
		log.Printf("Error: failed to create output zip file %s of directory %s: %v", zipFile, zipDir, err)
	} else {
		log.Printf("Created zip file %s of directory %s", zipFile, zipDir)
	}
	*/
}

// CreateMendelUiZip zips up the output in a form suitable for importing into the mendel web ui for data visualization
func CreateMendelUiZip(randomSlug string) {
	// Write input params to the output dir
	// Mendel web ui will never use the -z flag so dont have to worry about overwriting the toml file it has already written
	origCase_id := config.Cfg.Basic.Case_id
	if tomlWriter := config.FMgr.GetFile(config.TOML_FILENAME, 0); tomlWriter != nil {
		// insert job id into case_id param
		config.Cfg.Basic.Case_id = randomSlug
		if err := config.Cfg.WriteToFile(tomlWriter); err != nil { log.Fatalln(err) }
	}


	//todo: write real run output to OUTPUT_FILENAME, instead of just this msg
	if outputWriter := config.FMgr.GetFile(config.OUTPUT_FILENAME, 0); outputWriter != nil {
		outputStr := "The run log entries are not available in this job.\nThe plot files and inputs ARE available (click PLOTS or CONFIG below).\n"
		if _, err := io.WriteString(outputWriter, outputStr); err != nil { log.Fatalf("Error writing %s: %v", config.OUTPUT_FILENAME, err) }
	}
	config.FMgr.CloseAllFiles()	// explicitly close all files so the zip file contains all data

	pathToZipUp := config.Cfg.Computation.Data_file_path	// e.g. test/output/short
	zipFilePath := config.Cfg.Computation.Data_file_path+"/../"+origCase_id+"-"+randomSlug+".zip"	// e.g. test/output/short.zip
	prefixToReplace := config.Cfg.Computation.Data_file_path
	newPrefix := ""
	config.Verbose(2, "Creating zip file: pathToZipUp=%s, zipFilePath=%s, prefixToReplace=%s\n", pathToZipUp, zipFilePath, prefixToReplace)
	if err := utils.CreatePrefixedZip(pathToZipUp, zipFilePath, prefixToReplace, newPrefix); err != nil {
		log.Fatalf("Error creating zip of output data files: %v", err)
	}
	fmt.Printf("Zipped the output data files into %s for job id %s\n", filepath.Clean(zipFilePath), randomSlug)
}

// Shutdown does all the stuff necessary at the end of the run.
func shutdown() {
	if config.CmdArgs.SPCusername != "" {
		CreateSpcZip(config.CmdArgs.SPCusername, utils.RandomSlug(3))
	}
	if config.CmdArgs.CreateZip {
		CreateMendelUiZip(utils.RandomSlug(4))
	}
	utils.Measure.Stop("Total")
	utils.Measure.LogSummary() 		// it checks the verbosity level itself
	config.Verbose(5, "Shutting down...\n")
}

// Main handles cmd line args, reads input files, and contains the main generation loop.
func main() {
	log.SetOutput(os.Stdout) 	// needs to be done very early

	config.ReadCmdArgs()    // Get/check cmd line options and load specified input file - flags are accessible in config.CmdArgs, config values in config.Cfg

	// Handle the different input file choices
	if config.CmdArgs.Version {
		fmt.Println(MENDEL_GO_VERSION)
		os.Exit(0)

	} else if config.CmdArgs.InputFileToCreate != "" {
		if err := utils.CopyFile(config.FindDefaultFile(), config.CmdArgs.InputFileToCreate); err != nil { log.Fatalln(err) }
		os.Exit(0)

	} else if config.CmdArgs.InputFile != "" {
		if err := config.ReadFromFile(config.CmdArgs.InputFile); err != nil { log.Fatalln(err) }
		config.Verbose(3, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else { config.Usage(0) }		// this will exit

	// ReadFromFile() opened the output files, so arrange for them to be closed at the end
	defer config.FMgr.CloseAllFiles()
	if config.CmdArgs.SPCusername != "" && config.Cfg.Computation.Files_to_output != "*" { log.Fatalf("Error: if you specify the -u flag, the files_to_output value in the input file must be set to '*', so the produced zip file will have the proper content.") }
	if config.CmdArgs.CreateZip && config.Cfg.Computation.Files_to_output != "*" { log.Fatalf("Error: if you specify the -z flag, the files_to_output value in the input file must be set to '*', so the produced zip file will have the proper content.") }

	// Initialize profiling, if requested
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

	maxGenNum := config.Cfg.Basic.Num_generations
	//parentPop := pop.PopulationFactory(nil, 0) 		// genesis population
	//pop.Mdl.GenerateInitialAlleles(parentPop, uniformRandom)
	//parentPop.ReportInitial(maxGenNum)
	parentSpecies := pop.SpeciesFactory().Initialize(maxGenNum, uniformRandom)

	popMaxIsSet := pop.PopulationGrowthModelType(strings.ToLower(config.Cfg.Population.Pop_growth_model))==pop.EXPONENTIAL_POPULATON_GROWTH && config.Cfg.Population.Max_pop_size>0
	popMax := config.Cfg.Population.Max_pop_size

	// If num gens is 0 and not exponential growth, only report on genesis pop and then exit
	zeroGens := maxGenNum == 0 && !popMaxIsSet
	if config.Cfg.Population.Num_contrasting_alleles > 0 && (zeroGens || config.Cfg.Computation.Plot_allele_gens == 1) {
		totalInterimTime := utils.Measure.GetInterimTime("Total")
		//parentPop.ReportEachGen(0, zeroGens)
		parentSpecies.ReportEachGen(0, zeroGens, totalInterimTime, 0.0)
		if zeroGens {
			shutdown() // Finish up
			os.Exit(0)
		}
	}

	// Main generation loop.
	for gen := uint32(1); ; gen++ {
		utils.Measure.Start("Generations")		// this is stopped in ReportEachGen() so it can report each delta
		//childrenPop := pop.PopulationFactory(parentPop, gen)	// this creates the PopulationParts too
		//random.NextSeed = config.Cfg.Computation.Random_number_seed+1		// reset the seed to 1 above our initial seed, so when we call RandFactory() in Mate() for additional threads it will work like it did before
		childrenSpecies := parentSpecies.GetNextGeneration(gen)	// this creates the PopulationParts too
		//parentPop.Mate(childrenPop, uniformRandom)		// this fills in the next gen population object with the offspring
		parentSpecies.Mate(childrenSpecies, uniformRandom)		// this fills in the next gen populations object with the offspring
		utils.Measure.CheckAmountMemoryUsed()
		parentSpecies = nil 	// give GC a chance to reclaim the previous generation
		if config.Cfg.Computation.Force_gc { utils.CollectGarbage() }
		//childrenPop.Select(uniformRandom)
		childrenSpecies.Select(uniformRandom)

		// Check if we should stop the run
		lastGen := false
		if maxGenNum != 0 && gen >= maxGenNum {
			lastGen = true
		//} else if popMaxIsSet && childrenPop.GetCurrentSize() >= popMax {
		} else if popMaxIsSet && childrenSpecies.GetCurrentSize() >= popMax {
			log.Printf("Species has reached the max specified value of %d. Stopping simulation.", popMax)
			lastGen = true
		//} else if (pop.RecombinationType(config.Cfg.Population.Recombination_model) == pop.FULL_SEXUAL && childrenPop.GetCurrentSize() < 2) || childrenPop.GetCurrentSize() == 0 {
		} else if (pop.RecombinationType(config.Cfg.Population.Recombination_model) == pop.FULL_SEXUAL && childrenSpecies.GetCurrentSize() < 2) || childrenSpecies.GetCurrentSize() == 0 {
			// Above chceks if we don't have enough individuals to mate
			log.Println("Species is extinct. Stopping simulation.")
			lastGen = true
		//todo: check if all tribes are extinct
		//} else if aveFit, _, _, _, _ := childrenPop.GetFitnessStats(); aveFit < config.Cfg.Computation.Extinction_threshold {
		} else if childrenSpecies.GetAverageFitness() < config.Cfg.Computation.Extinction_threshold {
			// Above checks if the all the pops fitness is below the threshold
			log.Printf("Population fitness is below the extinction threshold of %.3f. Stopping simulation.", config.Cfg.Computation.Extinction_threshold)
			lastGen = true
		}

		totalInterimTime := utils.Measure.GetInterimTime("Total")
		genTime := utils.Measure.Stop("Generations")
		//childrenPop.ReportEachGen(gen, lastGen)
		childrenSpecies.ReportEachGen(gen, lastGen, totalInterimTime, genTime)
		if lastGen { break }
		//parentPop = childrenPop        // for the next iteration
		parentSpecies = childrenSpecies        // for the next iteration
	}

	shutdown()	// Finish up
}
