package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"strings"
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
		Alg.CalcDelMutationFitness = CalcUniformDelMutationFitness
		algNames = append(algNames, "CalcUniformDelMutationFitness")
		//Alg.CalcDelMutationFitness = CalcWeibullDelMutationFitness
		//algNames = append(algNames, "CalcWeibullDelMutationFitness")
	}

	if c.Mutations.Uniform_fitness_effect_fav != 0.0 {
		Alg.CalcFavMutationFitness = CalcFixedFavMutationFitness
		algNames = append(algNames, "CalcFixedFavMutationFitness")
	} else {
		Alg.CalcFavMutationFitness = CalcUniformFavMutationFitness
		algNames = append(algNames, "CalcUniformFavMutationFitness")
		//Alg.CalcFavMutationFitness = CalcWeibullFavMutationFitness
		//algNames = append(algNames, "CalcWeibullFavMutationFitness")
	}

	config.Verbose(3, "Running with these dna algorithms: %v", strings.Join(algNames, ", "))
}
