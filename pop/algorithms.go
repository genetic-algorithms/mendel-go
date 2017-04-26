package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"strings"
	"log"
)

type NumOffSpringModelType string
const (
	UNIFORM_NUM_OFFSPRING NumOffSpringModelType = "uniform"
	FIXED_NUM_OFFSPRING NumOffSpringModelType = "fixed"
	FORTRAN_NUM_OFFSPRING NumOffSpringModelType = "fortran"
	FITNESS_NUM_OFFSPRING NumOffSpringModelType = "fitness"
)

type MutationRateModelType string
const (
	FIXED_MUTN_RATE MutationRateModelType = "fixed"
	POISSON_MUTN_RATE MutationRateModelType = "poisson"
)


// Algorithms holds pointers to functions that implement the various algorithms chosen by the input file.
type Algorithms struct {
	CalcNumOffspring CalcNumOffspringType
	CalcIndivFitness CalcIndivFitnessType
	CalcNumMutations CalcNumMutationsType
}

// Alg is the singleton instance of Algorithms that can be accessed throughout the dna package. It gets set in SetAlgorithms().
var Alg *Algorithms


// SetAlgorithms is called by main.initialize() to set the function ptrs for the various algorithms chosen by the input file.
func SetAlgorithms(c *config.Config) {
	Alg = &Algorithms{} 		// create and set the singleton object
	var algNames []string 		// gather the algorithms we use so we can print it out

	// uniform (even distribution), fixed (rounded to nearest int), fortran (what mendel-f90 used), fitness (weighted according to fitness)
	switch {
	case strings.ToLower(c.Population.Num_offspring_model) == string(UNIFORM_NUM_OFFSPRING):
		Alg.CalcNumOffspring = CalcUniformNumOffspring
		algNames = append(algNames, "CalcUniformNumOffspring")
	case strings.ToLower(c.Population.Num_offspring_model) == string(FIXED_NUM_OFFSPRING):
		Alg.CalcNumOffspring = CalcSemiFixedNumOffspring
		algNames = append(algNames, "CalcFixedNumOffspring")
	case strings.ToLower(c.Population.Num_offspring_model) == string(FORTRAN_NUM_OFFSPRING):
		Alg.CalcNumOffspring = CalcFortranNumOffspring
		algNames = append(algNames, "CalcFortranNumOffspring")
	case strings.ToLower(c.Population.Num_offspring_model) == string(FITNESS_NUM_OFFSPRING):
		Alg.CalcNumOffspring = CalcFitnessNumOffspring
		algNames = append(algNames, "CalcFitnessNumOffspring")
	default:
		log.Fatalf("Error: unrecognized value for mum_offspring_model: %v", c.Population.Num_offspring_model)
	}

	if c.Mutations.Multiplicative_weighting > 0.0 {
		Alg.CalcIndivFitness = MultIndivFitness
		algNames = append(algNames, "MultIndivFitness")
	} else {
		Alg.CalcIndivFitness = SumIndivFitness
		algNames = append(algNames, "SumIndivFitness")
	}

	switch {
	case strings.ToLower(c.Mutations.Mutn_rate_model) == string(FIXED_MUTN_RATE):
		Alg.CalcNumMutations = CalcSemiFixedNumMutations
		algNames = append(algNames, "CalcSemiFixedNumMutations")
	case strings.ToLower(c.Mutations.Mutn_rate_model) == string(POISSON_MUTN_RATE):
		Alg.CalcNumMutations = CalcPoissonNumMutations
		algNames = append(algNames, "CalcPoissonNumMutations")
	default:
		log.Fatalf("Error: unrecognized value for mutn_rate_model: %v", c.Mutations.Mutn_rate_model)
	}

	config.Verbose(3, "Running with these pop algorithms: %v", strings.Join(algNames, ", "))
}
