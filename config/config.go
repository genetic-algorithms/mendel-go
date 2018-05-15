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
	"github.com/genetic-algorithms/mendel-go/utils"
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
		Initial_allele_fitness_model string
		Initial_alleles_pop_frac float64
		Initial_alleles_frequencies string
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
	Tribes struct {
		Num_tribes uint32
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
		//Restart_case bool
		//Restart_dump_number uint32
		Plot_allele_gens uint32
		Omit_first_allele_bin bool
		Count_duplicate_alleles bool
		Verbosity uint32
		Data_file_path string
		Files_to_output string
		Performance_profile string
		//Transfer_linkage_blocks bool
		Force_gc bool
		Allele_count_gc_interval uint32
		//Reuse_populations bool
		Perf_option int
	}
}

// Cfg is the singleton instance of Config that can be accessed throughout the mendel code.
// It gets set in ReadFromFile().
var Cfg *Config

// These values are computed from the Config params and used in several places in the code
type ComputedValues struct {
	Lb_modulo float64
	Alpha_del float64
	Alpha_fav float64
	Gamma_del float64
	Gamma_fav float64
	Del_scale float64		// not sure if i really need these
	Fav_scale float64
}

var Computed *ComputedValues

// ReadFromFile reads the specified input file and parses all of the values into the Config struct.
// This is also the factory method for the Config class and will store the created instance in this packages Cfg var.
func ReadFromFile(filename string) error {
	Cfg = &Config{} 		// create and set the singleton config

	// 1st read defaults and then apply the specified config file values on top of that
	defaultFile := FindDefaultFile()
	if defaultFile == "" { return errors.New("can not find "+DEFAULT_INPUT_FILE) }
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
	Computed = ComputedValuesFactory()
	return nil
}

// Validate checks the config values to make sure they are valid.
func (c *Config) validateAndAdjust() error {
	// Check and adjust certain config values
	if c.Basic.Pop_size % 2 != 0 { return errors.New("basic.pop_size must be an even number") }
	if (c.Population.Num_linkage_subunits % c.Population.Haploid_chromosome_number) != 0 { return errors.New("num_linkage_subunits must be an exact multiple of haploid_chromosome_number") }

	c.Selection.Heritability = math.Max(1.e-20, c.Selection.Heritability)   // Limit the minimum value of heritability to be 10**-20

	if c.Mutations.Max_fav_fitness_gain <= 0.0	{ return errors.New("max_fav_fitness_gain must be > 0.0") }

	if c.Mutations.Allow_back_mutn && c.Computation.Tracking_threshold != 0.0 { return errors.New("can not set both allow_back_mutn and a non-zero tracking_threshold") }
	if c.Mutations.Multiplicative_weighting != 0.0 && c.Computation.Tracking_threshold != 0.0 { return errors.New("setting tracking_threshold with multiplicative_weighting is not yet supported") }

	if c.Computation.Tracking_threshold >= 1.0 && (FMgr.IsDir(ALLELE_BINS_DIRECTORY) || FMgr.IsDir(NORMALIZED_ALLELE_BINS_DIRECTORY) || FMgr.IsDir(DISTRIBUTION_DEL_DIRECTORY) || FMgr.IsDir(DISTRIBUTION_FAV_DIRECTORY)) {
		return errors.New(ALLELE_BINS_DIRECTORY+", "+NORMALIZED_ALLELE_BINS_DIRECTORY+", "+DISTRIBUTION_DEL_DIRECTORY+", or "+DISTRIBUTION_FAV_DIRECTORY+" file output was requested, but no alleles can be plotted when tracking_threshold >= 1.0")
	}
	if !FMgr.IsDir(ALLELE_BINS_DIRECTORY) && !FMgr.IsDir(NORMALIZED_ALLELE_BINS_DIRECTORY) && !FMgr.IsDir(DISTRIBUTION_DEL_DIRECTORY) && !FMgr.IsDir(DISTRIBUTION_FAV_DIRECTORY) {
		log.Printf("Since %v, %v, %v, and %v were not requested to be written, setting tracking_threshold=9.0 to save space/time\n", ALLELE_BINS_DIRECTORY, NORMALIZED_ALLELE_BINS_DIRECTORY, DISTRIBUTION_DEL_DIRECTORY, DISTRIBUTION_FAV_DIRECTORY)
		c.Computation.Tracking_threshold = 9.0
	}
	//if c.Computation.Track_neutrals && c.Computation.Tracking_threshold != 0.0 { c.Computation.Track_neutrals = false }

	if c.Computation.Num_threads == 0 { c.Computation.Num_threads = uint32(runtime.NumCPU()) }

	if c.Tribes.Num_tribes <= 0 { return errors.New("num_tribes can not be <= 0") }

	return nil
}


func ComputedValuesFactory() (c *ComputedValues) {
	c = &ComputedValues{}
	logn := math.Log // can't use log() because there is a package named that
	exp := math.Exp
	pow := math.Pow
	max_fav_fitness_gain := Cfg.Mutations.Max_fav_fitness_gain
	tracking_threshold := utils.MaxFloat64(1.0/Cfg.Mutations.Genome_size, float64(Cfg.Computation.Tracking_threshold))
	high_impact_mutn_threshold := Cfg.Mutations.High_impact_mutn_threshold
	high_impact_mutn_fraction := Cfg.Mutations.High_impact_mutn_fraction

	// Taken from mendel-f90/init.f90
	c.Lb_modulo = (pow(2,30)-2) / float64(Cfg.Population.Num_linkage_subunits)

	c.Alpha_del = logn(Cfg.Mutations.Genome_size)		// this is the lower bound of how small (close to 0) a del mutn can be when using weibull
	if Cfg.Mutations.Max_fav_fitness_gain > 0.0 {		// Alpha_fav is also the bound of how small a fav mutn fitness can be
		c.Alpha_fav = logn(Cfg.Mutations.Genome_size * Cfg.Mutations.Max_fav_fitness_gain)
	} else {
		c.Alpha_fav = c.Alpha_del
	}

	c.Gamma_del = logn(-logn(high_impact_mutn_threshold) / c.Alpha_del) / logn(high_impact_mutn_fraction)
	c.Gamma_fav = logn(-logn(high_impact_mutn_threshold) / c.Alpha_fav) / logn(high_impact_mutn_fraction)

	if tracking_threshold <= 1.0/Cfg.Mutations.Genome_size {
		c.Del_scale = 1. / (c.Lb_modulo - 2)
		c.Fav_scale = 1. / (c.Lb_modulo - 2)
	} else if tracking_threshold >= 1.0 {
		c.Del_scale = 0. // dont track any mutations
		c.Fav_scale = 0.
	} else {
		c.Del_scale = exp(logn(-logn(tracking_threshold)/c.Alpha_del) / c.Gamma_del) / (c.Lb_modulo - 2)
		if Cfg.Mutations.Max_fav_fitness_gain > 0. {
			c.Fav_scale = exp(logn(-logn(tracking_threshold / max_fav_fitness_gain)/c.Alpha_fav)/c.Gamma_fav) / (c.Lb_modulo - 2)
		} else {
			c.Fav_scale = 0.
		}
	}
	return
}


// FindDefaultFile looks for the defaults input file and returns the 1st one it finds. It exits with error if it can't find one.
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

/* toml.Marshal writes floats in a long exponent form not convenient
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
