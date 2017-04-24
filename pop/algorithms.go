package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"strings"
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

	Alg.CalcNumOffspring = CalcUniformNumOffspring 		// do not have a config param to choose between these, because i don't know if this algorithm is valid
	algNames = append(algNames, "CalcUniformNumOffspring")
	//Alg.CalcNumOffspring = CalcFortranNumOffspring
	//algNames = append(algNames, "CalcFortranNumOffspring")

	if c.Mutations.Multiplicative_weighting > 0.0 {
		Alg.CalcIndivFitness = MultIndivFitness
		algNames = append(algNames, "MultIndivFitness")
	} else {
		Alg.CalcIndivFitness = SumIndivFitness
		algNames = append(algNames, "SumIndivFitness")
	}

	Alg.CalcNumMutations = CalcSemiFixedNumMutations 		// do not have a config param to choose between these, because i don't know if this algorithm is valid
	algNames = append(algNames, "CalcSemiFixedNumMutations")
	//Alg.CalcNumMutations = CalcPoissonNumMutations
	//algNames = append(algNames, "CalcPoissonNumMutations")

	config.Verbose(3, "Running with these pop algorithms: %v", strings.Join(algNames, ", "))
}
