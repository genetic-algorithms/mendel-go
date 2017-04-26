// Package config provides the object for reading a mendel input file and accessing the variables in it.
package config

import (
	"github.com/naoina/toml" 		// implementation of TOML we are using to read input files
	"io/ioutil"
	"log"
	"errors"
	"fmt"
)

// Config is the struct that gets filled in by TOML automatically from the input file.
type Config struct {
	Basic struct {
		Case_id string
		Pop_size int
		Num_generations int
	}
	Mutations struct {
		Mutn_rate float64
		Mutn_rate_model string 		// toml does not know how to handle user-defined types like MutationRateModelType
		Frac_fav_mutn float64
		Fraction_neutral float64
		Fitness_distrib_type int
		Genome_size float64
		High_impact_mutn_fraction float64
		High_impact_mutn_threshold float64
		Num_initial_fav_mutn int
		Max_fav_fitness_gain float64
		Uniform_fitness_effect_del float64
		Uniform_fitness_effect_fav float64
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
		Selection_scheme int
		Partial_truncation_value float64
	}
	Population struct {
		Reproductive_rate float64
		Num_offspring_model string
		Recombination_model int
		Fraction_self_fertilization float64
		Num_contrasting_alleles int
		Max_total_fitness_increase int
		Initial_alleles_pop_frac float64
		Initial_alleles_amp_factor int
		Dynamic_linkage bool
		Haploid_chromosome_number int
		Num_linkage_subunits int
		Pop_growth_model int
		Pop_growth_rate float64
		Carrying_capacity int
		Bottleneck_yes bool
		Bottleneck_generation int
		Bottleneck_pop_size int
		Num_bottleneck_generations int
	}
	Substructure struct {
		Is_parallel bool
		Homogenous_tribes bool
		Num_indiv_exchanged int
		Migration_generations int
		Migration_model int
		Tribal_competition bool
		Tribal_fission bool
		Tc_scaling_factor float64
		Group_heritability float64
		Social_bonus_factor float64
	}
	Computation struct {
		Tracking_threshold float64
		Extinction_threshold float64
		Max_del_mutn_per_indiv int
		Max_neu_mutn_per_indiv int
		Max_fav_mutn_per_indiv int
		Random_number_seed int64
		Reseed_rng bool			// not used. If Random_number_seed==0 we use a truly random seed
		Track_neutrals bool
		Write_dump bool
		Write_vcf bool
		Restart_case bool
		Restart_dump_number int
		Plot_allele_gens int
		Verbosity int
		Data_file_path string
		Files_to_output string
	}
}

// Cfg is the singleton instance of Config that can be accessed throughout the mendel code.
// It gets set in ReadFromFile().
var Cfg *Config

// ReadFromFile reads the specified input file and parses all of the values into the Config struct.
// This is also the factory method for the Config class and will store the created instance in this packages Cfg var.
func ReadFromFile(filename string) error {
	log.Printf("Reading %v...\n", filename) 	// can not use verbosity here because we have not read the config file yet
	buf, err := ioutil.ReadFile(filename)
	if err != nil { return err }
	Cfg = &Config{} 		// create and set the singleton config
	if err := toml.Unmarshal(buf, Cfg); err != nil { return err }

	FileMgrFactory(Cfg.Computation.Data_file_path, Cfg.Computation.Files_to_output)
	return Cfg.validate()
}

// Validate checks the config values to make sure they are valid.
func (c *Config) validate() error {
	if c.Basic.Pop_size % 2 != 0 { return errors.New("Error: basic.pop_size must be an even number") }
	return nil
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
func Verbose(level int, msg string, args ...interface{}) {
	if Cfg.Computation.Verbosity >= level { log.Printf("V"+fmt.Sprint(level)+" "+msg, args...) }
}

// IsVerbose tests whether the level given is within the verbose level being output
func IsVerbose(level int) bool {
	return Cfg.Computation.Verbosity >= level
}
