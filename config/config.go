// Package config provides the object for reading a mendel input file and accessing the variables in it.
package config

import (
	//"github.com/naoina/toml" 		// implementation of TOML we are using to read input files
	"github.com/BurntSushi/toml" 		// implementation of TOML we are using to read input files
	//"io/ioutil"
	"log"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"runtime"
	"github.com/genetic-algorithms/mendel-go/utils"
	"fmt"
	"bytes"
)

// Config is the struct that gets filled in by TOML automatically from the input file.
type Config struct {
	Basic struct {
		Case_id string  `toml:"case_id"`
		Description string  `toml:"description"`
		Pop_size uint32  `toml:"pop_size"`
		Num_generations uint32  `toml:"num_generations"`
	}  `toml:"basic"`
	Mutations struct {
		Mutn_rate float64  `toml:"mutn_rate"`
		Mutn_rate_model string  `toml:"mutn_rate_model"`	// toml does not know how to handle user-defined types like MutationRateModelType
		Frac_fav_mutn float64  `toml:"frac_fav_mutn"`
		Fraction_neutral float64  `toml:"fraction_neutral"`
		Genome_size float64  `toml:"genome_size"`
		Fitness_effect_model string  `toml:"fitness_effect_model"`
		Uniform_fitness_effect_del float64  `toml:"uniform_fitness_effect_del"`
		Uniform_fitness_effect_fav float64  `toml:"uniform_fitness_effect_fav"`
		High_impact_mutn_fraction float64  `toml:"high_impact_mutn_fraction"`
		High_impact_mutn_threshold float64  `toml:"high_impact_mutn_threshold"`
		Max_fav_fitness_gain float64  `toml:"max_fav_fitness_gain"`
		Fraction_recessive float64  `toml:"fraction_recessive"`
		Recessive_hetero_expression float64  `toml:"recessive_hetero_expression"`
		Dominant_hetero_expression float64  `toml:"dominant_hetero_expression"`
		Multiplicative_weighting float64  `toml:"multiplicative_weighting"`
		Synergistic_epistasis bool  `toml:"synergistic_epistasis"`
		Se_nonlinked_scaling float64  `toml:"se_nonlinked_scaling"`
		Se_linked_scaling float64  `toml:"se_linked_scaling"`
		Upload_mutations bool  `toml:"upload_mutations"`
		Allow_back_mutn bool  `toml:"allow_back_mutn"`
		Polygenic_beneficials bool  `polygenic_beneficials`
		Polygenic_init string  `toml:"polygenic_init"`
		Polygenic_target string  `toml:"polygenic_target"`
		Polygenic_effect float64  `toml:"polygenic_effect"`
	}  `toml:"mutations"`
	Selection struct {
		Fraction_random_death float64  `toml:"fraction_random_death"`
		Fitness_dependent_fertility bool  `toml:"fitness_dependent_fertility"`
		Selection_model string  `toml:"selection_model"`
		Heritability float64  `toml:"heritability"`
		Non_scaling_noise float64  `toml:"non_scaling_noise"`
		Partial_truncation_value float64  `toml:"partial_truncation_value"`
	}  `toml:"selection"`
	Population struct {
		Reproductive_rate float64  `toml:"reproductive_rate"`
		Num_offspring_model string  `toml:"num_offspring_model"`
		Recombination_model uint32  `toml:"recombination_model"`
		Fraction_self_fertilization float64  `toml:"fraction_self_fertilization"`
		Crossover_model string  `toml:"crossover_model"`
		Mean_num_crossovers uint32  `toml:"mean_num_crossovers"`
		Haploid_chromosome_number uint32  `toml:"haploid_chromosome_number"`
		Num_linkage_subunits uint32  `toml:"num_linkage_subunits"`
		Num_contrasting_alleles uint32  `toml:"num_contrasting_alleles"`
		Initial_allele_fitness_model string  `toml:"initial_allele_fitness_model"`
		Initial_alleles_pop_frac float64  `toml:"initial_alleles_pop_frac"`
		Initial_alleles_frequencies string  `toml:"initial_alleles_frequencies"`
		Max_total_fitness_increase float64  `toml:"max_total_fitness_increase"`
		Pop_growth_model string  `toml:"pop_growth_model"`
		Pop_growth_rate float64  `toml:"pop_growth_rate"`
		Pop_growth_rate2 float64  `toml:"pop_growth_rate2"`
		Max_pop_size uint32  `toml:"max_pop_size"`
		Carrying_capacity uint32  `toml:"carrying_capacity"`
		Multiple_Bottlenecks string  `toml:"multiple_bottlenecks"`
		Bottleneck_generation uint32  `toml:"bottleneck_generation"`
		Bottleneck_pop_size uint32  `toml:"bottleneck_pop_size"`
		Num_bottleneck_generations uint32  `toml:"num_bottleneck_generations"`
	}  `toml:"population"`
	Tribes struct {
		Num_tribes uint32  `toml:"num_tribes"`
		Homogenous_tribes bool  `toml:"homogenous_tribes"`
		Num_indiv_exchanged uint32  `toml:"num_indiv_exchanged"`
		Migration_generations uint32  `toml:"migration_generations"`
		Migration_model int  `toml:"migration_model"`
		Tribal_competition bool  `toml:"tribal_competition"`
		Tribal_fission bool  `toml:"tribal_fission"`
		Tc_scaling_factor float64  `toml:"tc_scaling_factor"`
		Group_heritability float64  `toml:"group_heritability"`
		Social_bonus_factor float64  `toml:"social_bonus_factor"`
	}  `toml:"tribes"`
	Computation struct {
		Tracking_threshold float32  `toml:"tracking_threshold"`
		Track_neutrals bool  `toml:"track_neutrals"`
		Extinction_threshold float64  `toml:"extinction_threshold"`
		Verbosity uint32  `toml:"verbosity"`
		Data_file_path string  `toml:"data_file_path"`
		Files_to_output string  `toml:"files_to_output"`
		Plot_allele_gens uint32  `toml:"plot_allele_gens"`
		Omit_first_allele_bin bool  `toml:"omit_first_allele_bin"`
		//Restart_case bool  `toml:"restart_case"`
		//Restart_dump_number uint32  `toml:"restart_dump_number"`
		// Considered advanced options:
		Num_threads uint32  `toml:"num_threads"`
		Random_number_seed int64  `toml:"random_number_seed"`
		Count_duplicate_alleles bool  `toml:"count_duplicate_alleles"`
		Performance_profile string  `toml:"performance_profile"`
		Force_gc bool  `toml:"force_gc"`
		Allele_count_gc_interval uint32  `toml:"allele_count_gc_interval"`
		Perf_option int  `toml:"perf_option"`
		//Transfer_linkage_blocks bool  `toml:"transfer_linkage_blocks"`
		//Reuse_populations bool  `toml:"reuse_populations"`
	}  `toml:"computation"`
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
	if defaultFile == "" { return errors.New("can not find "+ DEFAULTS_INPUT_FILE) }
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
	if _, err := os.Stat(DEFAULTS_INPUT_FILE); err == nil { return DEFAULTS_INPUT_FILE
	}
	lookedIn := []string{"."}

	// Check in the directory where the executable is
	//fmt.Printf("arg 0: %s\n", os.Args[0])
	if strings.Contains(os.Args[0], "/") {
		canonicalPath, err := filepath.EvalSymlinks(os.Args[0])
		if err != nil { log.Fatal(err) }
		//fmt.Println("here1")
		executableDir, err := filepath.Abs(filepath.Dir(canonicalPath)) // the result of EvalSymlinks can be relative in some situations
		if err != nil { log.Fatal(err) }
		defaultsFile := executableDir + "/" + DEFAULTS_INPUT_FILE
		if _, err := os.Stat(defaultsFile); err == nil { return defaultsFile }
		lookedIn = append(lookedIn, executableDir)
	}
	// else the executable came from the path, so we don't know where from, so can't look for the defaults file there

	// Now check in the dirs listed in DEFAULTS_INPUT_DIRS (these are where the built pkgs install them on various linuxes)
	for _, dir := range DEFAULTS_INPUT_DIRS {
		// we assume these dirs are already canonical and absolute
		defaultsFile := dir + "/" + DEFAULTS_INPUT_FILE
		if _, err := os.Stat(defaultsFile); err == nil { return defaultsFile }
		lookedIn = append(lookedIn, dir)
	}

	log.Fatalf("Error: can not find %v. Looked in: %v", DEFAULTS_INPUT_FILE, strings.Join(lookedIn, ", "))
	return ""		// could not find it
}

// WriteToFile writes the current config to a file descriptor. The caller is responsible to open the file,
// log that it is being written, and close the file (so it can be used with files managed by FileMgr).
func (c *Config) WriteToFile(file *os.File) error {
	//buf, err := toml.Marshal(*c)
	buf := new(bytes.Buffer)
	// Note: for this to work properly, you must have struct tags on all of the fields specifying the name starting w/lowercase
	if err := toml.NewEncoder(buf).Encode(c); err != nil { return err }
	if _, err := file.Write(buf.Bytes()); err != nil { return err }
	return nil
}

// These are here, instead of of in pkg utils, to avoid circular imports
func Verbose(level uint32, msg string, args ...interface{}) {
	if Cfg.Computation.Verbosity >= level { log.Printf("V"+fmt.Sprint(level)+" "+msg, args...) }
}

// IsVerbose tests whether the level given is within the verbose level being output
func IsVerbose(level uint32) bool {
	return Cfg.Computation.Verbosity >= level
}
