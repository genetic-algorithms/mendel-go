package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"strings"
	"log"
)

type MutationFitnessModelType string
const (
	UNIFORM_FITNESS_EFFECT MutationFitnessModelType = "uniform"
	WEIBULL_FITNESS_EFFECT MutationFitnessModelType = "weibull"
)


// Algorithms holds pointers to functions that implement the various algorithms chosen by the input file.
type Algorithms struct {
	CalcDelMutationFitness CalcMutationFitnessType
	CalcFavMutationFitness CalcMutationFitnessType
}

// Alg is the singleton instance of Algorithms that can be accessed throughout the dna package. It gets set in SetAlgorithms().
var Alg *Algorithms


// SetAlgorithms is called by main.initialize() to set the function ptrs for the various algorithms chosen by the input file.
func SetAlgorithms(c *config.Config) {
	Alg = &Algorithms{} 		// create and set the singleton object
	var algNames []string 		// gather the algorithms we use so we can print it out

	if c.Mutations.Uniform_fitness_effect_del != 0.0 {
		Alg.CalcDelMutationFitness = CalcFixedDelMutationFitness
		algNames = append(algNames, "CalcFixedDelMutationFitness")
	} else {
		switch {
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(UNIFORM_FITNESS_EFFECT):
			Alg.CalcDelMutationFitness = CalcUniformDelMutationFitness
			algNames = append(algNames, "CalcUniformDelMutationFitness")
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(WEIBULL_FITNESS_EFFECT):
			Alg.CalcDelMutationFitness = CalcWeibullDelMutationFitness
			algNames = append(algNames, "CalcWeibullDelMutationFitness")
		default:
			log.Fatalf("Error: unrecognized value for fitness_effect_model: %v", c.Mutations.Fitness_effect_model)
		}
	}

	if c.Mutations.Uniform_fitness_effect_fav != 0.0 {
		Alg.CalcFavMutationFitness = CalcFixedFavMutationFitness
		algNames = append(algNames, "CalcFixedFavMutationFitness")
	} else {
		switch {
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(UNIFORM_FITNESS_EFFECT):
			Alg.CalcFavMutationFitness = CalcUniformFavMutationFitness
			algNames = append(algNames, "CalcUniformFavMutationFitness")
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(WEIBULL_FITNESS_EFFECT):
			Alg.CalcFavMutationFitness = CalcWeibullFavMutationFitness
			algNames = append(algNames, "CalcWeibullFavMutationFitness")
		default:
			log.Fatalf("Error: unrecognized value for fitness_effect_model: %v", c.Mutations.Fitness_effect_model)
		}
	}

	config.Verbose(3, "Running with these dna algorithms: %v", strings.Join(algNames, ", "))
}
