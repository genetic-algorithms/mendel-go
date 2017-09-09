// Package config provides the object for reading a mendel input file and accessing the variables in it.
package config

import (
	//"github.com/naoina/toml" 		// implementation of TOML we are using to read input files
	"github.com/BurntSushi/toml" 		// implementation of TOML we are using to read input files
	//"io/ioutil"
	"log"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"runtime"
)

// Config is the struct that gets filled in by TOML automatically from the input file.
type Config struct {
	Basic struct {
		Case_id string
		Pop_size uint32
		Num_generations uint32
	}
	Mutations struct {
		Mutn_rate float64
		Mutn_rate_model string 		// toml does not know how to handle user-defined types like MutationRateModelType
		Frac_fav_mutn float64
		Fraction_neutral float64
		Genome_size float64
		High_impact_mutn_fraction float64
		High_impact_mutn_threshold float64
		Num_initial_fav_mutn uint32
		Max_fav_fitness_gain float64
		Uniform_fitness_effect_del float64
		Uniform_fitness_effect_fav float64
		Fitness_effect_model string
		Fraction_recessive float64
		Recessive_hetero_expression float64
		Dominant_hetero_expression float64
		Multiplicative_weighting float64
		Synergistic_epistasis bool
		Se_nonlinked_scaling float64
		Se_linked_scaling float64
		Upload_mutations bool
		Allow_back_mutn bool
		Polygenic_beneficials bool
		Polygenic_init string
		Polygenic_target string
		Polygenic_effect float64
	}
	Selection struct {
		Fraction_random_death float64
		Heritability float64
		Non_scaling_noise float64
		Fitness_dependent_fertility bool
		Selection_model string
		Partial_truncation_value float64
	}
	Population struct {
		Reproductive_rate float64
		Num_offspring_model string
		Recombination_model uint32
		Fraction_self_fertilization float64
		Num_contrasting_alleles uint32
		Initial_alleles_pop_frac float64
		Initial_allele_fitness_model string
		Max_total_fitness_increase float64
		Crossover_model string
		Mean_num_crossovers uint32
		Haploid_chromosome_number uint32
		Num_linkage_subunits uint32
		Pop_growth_model string
		Pop_growth_rate float64
		Pop_growth_rate2 float64
		Max_pop_size uint32
		Carrying_capacity uint32
		//Bottleneck_yes bool
		Bottleneck_generation uint32
		Bottleneck_pop_size uint32
		Num_bottleneck_generations uint32
	}
	Substructure struct {
		Is_parallel bool
		Homogenous_tribes bool
		Num_indiv_exchanged uint32
		Migration_generations uint32
		Migration_model int
		Tribal_competition bool
		Tribal_fission bool
		Tc_scaling_factor float64
		Group_heritability float64
		Social_bonus_factor float64
	}
	Computation struct {
		Tracking_threshold float32
		Extinction_threshold float64
		Track_neutrals bool
		Num_threads uint32
		Random_number_seed int64
		Restart_case bool
		Restart_dump_number uint32
		Plot_allele_gens uint32
		Verbosity uint32
		Data_file_path string
		Files_to_output string
		Performance_profile string
		//Transfer_linkage_blocks bool
		Force_gc bool
		Allele_count_gc_interval uint32
		Reuse_populations bool
	}
}

// Cfg is the singleton instance of Config that can be accessed throughout the mendel code.
// It gets set in ReadFromFile().
var Cfg *Config

// ReadFromFile reads the specified input file and parses all of the values into the Config struct.
// This is also the factory method for the Config class and will store the created instance in this packages Cfg var.
func ReadFromFile(filename string) error {
	Cfg = &Config{} 		// create and set the singleton config

	// 1st read defaults and then apply the specified config file values on top of that
	/* this is the old code when i was using github.com/naoina/toml ...
	buf, err := ioutil.ReadFile(DEFAULT_INPUT_FILE)
	if err != nil { return err }
	if err := toml.Unmarshal(buf, Cfg); err != nil { return err }
	buf, err = ioutil.ReadFile(filename)
	if err != nil { return err }
	if err := toml.Unmarshal(buf, Cfg); err != nil { return err }
	*/
	defaultFile := FindDefaultFile()
	if defaultFile == "" { return errors.New("Error: can not find "+DEFAULT_INPUT_FILE) }
	log.Printf("Using defaults file %v\n", defaultFile) 	// can not use verbosity here because we have not read the config file yet
	if _, err := toml.DecodeFile(defaultFile, Cfg); err != nil { return err }
	if filename != defaultFile {
		log.Printf("Using config file %v\n", filename) 	// can not use verbosity here because we have not read the config file yet
		if _, err := toml.DecodeFile(filename, Cfg); err != nil { return err }
	}

	// Do this before validate, because we need to know what output files have been requested for some of the validation testing
	if Cfg.Computation.Data_file_path == "" { Cfg.Computation.Data_file_path = "./test/output/" + Cfg.Basic.Case_id }
	FileMgrFactory(Cfg.Computation.Data_file_path, Cfg.Computation.Files_to_output)

	if err := Cfg.validateAndAdjust(); err != nil { log.Fatalln(err) }
	return nil
}

// Validate checks the config values to make sure they are valid.
func (c *Config) validateAndAdjust() error {
	// Check and adjust certain config values
	if c.Basic.Pop_size % 2 != 0 { return errors.New("Error: basic.pop_size must be an even number") }
	if (c.Population.Num_linkage_subunits % c.Population.Haploid_chromosome_number) != 0 { return errors.New("Error: Num_linkage_subunits must be an exact multiple of haploid_chromosome_number.") }

	c.Selection.Heritability = math.Max(1.e-20, c.Selection.Heritability)   // Limit the minimum value of heritability to be 10**-20

	if c.Mutations.Allow_back_mutn && c.Computation.Tracking_threshold != 0.0 { return errors.New("Error: Can not set both allow_back_mutn and a non-zero tracking_threshold.") }
	if c.Mutations.Multiplicative_weighting != 0.0 && c.Computation.Tracking_threshold != 0.0 { return errors.New("Error: Setting tracking_threshold with multiplicative_weighting is not yet supported.") }

	if c.Computation.Tracking_threshold >= 1.0 && FMgr.IsDir(ALLELE_BINS_DIRECTORY) { return errors.New("Error: "+ALLELE_BINS_DIRECTORY+" file output was requested, but no alleles can be plotted when tracking_threshold >= 1.0") }
	if !FMgr.IsDir(ALLELE_BINS_DIRECTORY) {
		log.Printf("Since %v was not requested to be written, setting tracking_threshold=9.0 to save space/time\n", ALLELE_BINS_DIRECTORY)
		c.Computation.Tracking_threshold = 9.0
	}
	//if c.Mutations.Fraction_neutral == 0 { c.Computation.Track_neutrals = false }   // do not actually need this
	if c.Computation.Track_neutrals && c.Computation.Tracking_threshold != 0.0 {
		//return errors.New("Error: Does not make sense to set both track_neutrals and a non-zero tracking_threshold.")  // override Track_neutrals instead of returning error
		c.Computation.Track_neutrals = false
	}

	if c.Population.Num_contrasting_alleles > 0 && (c.Population.Initial_alleles_pop_frac <= 0.0 || c.Population.Initial_alleles_pop_frac > 1.0) { return errors.New("Error: If num_contrasting_alleles is > 0, then initial_alleles_pop_frac must be > 0 and <= 1.0.") }

	if c.Computation.Num_threads == 0 { c.Computation.Num_threads = uint32(runtime.NumCPU()) }

	if c.Computation.Reuse_populations && c.Population.Pop_growth_model != "none" { 	//todo: can't use enum for node, because of circular import
		log.Println("Forcing reuse_populations to false because population growth was specified")
		c.Computation.Reuse_populations = false
	}

	return nil
}


// FindDefaultFile looks for the defaults input file and returns the 1st one it finds. It exists with error if it can't find one.
func FindDefaultFile() string {
	// If they explicitly told us on the cmd line where it is use that
	if CmdArgs.DefaultFile != "" {
		if _, err := os.Stat(CmdArgs.DefaultFile); err == nil { return CmdArgs.DefaultFile }
		log.Fatalf("Error: specified defaults file %v does not exist", CmdArgs.DefaultFile)
	}

	// Check for it in the current directory
	if _, err := os.Stat(DEFAULT_INPUT_FILE); err == nil { return DEFAULT_INPUT_FILE }
	lookedIn := []string{"."}

	canonicalPath, err := filepath.EvalSymlinks(os.Args[0])
	if err != nil { log.Fatal(err) }
	executableDir, err := filepath.Abs(filepath.Dir(canonicalPath))   // the result of EvalSymlinks can be relative in some situations
	if err != nil { log.Fatal(err) }
	defaultsFile := executableDir + "/" + DEFAULT_INPUT_FILE
	if _, err := os.Stat(defaultsFile); err == nil { return defaultsFile }
	lookedIn = append(lookedIn, executableDir)

	log.Fatalf("Error: can not find %v. Looked in: %v", DEFAULT_INPUT_FILE, strings.Join(lookedIn, " and "))
	return ""		// could not find it
}

/* toml.Marshal writes floats in a long exponention form not convenient
func (c *Config) WriteToFile(filename string) error {
	log.Printf("Writing %v...\n", filename)
	buf, err := toml.Marshal(*c)
	if err != nil { return err }
	if err := ioutil.WriteFile(filename, buf, 0644); err != nil { return err }
	return nil
}
*/

// These are here, instead of of in pkg utils, to avoid circular imports
func Verbose(level uint32, msg string, args ...interface{}) {
	if Cfg.Computation.Verbosity >= level { log.Printf("V"+fmt.Sprint(level)+" "+msg, args...) }
}

// IsVerbose tests whether the level given is within the verbose level being output
func IsVerbose(level uint32) bool {
	return Cfg.Computation.Verbosity >= level
}
