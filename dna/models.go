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


// Models holds pointers to functions that implement the various algorithms chosen by the input file.
type Models struct {
	CalcDelMutationFitness CalcMutationFitnessType
	CalcFavMutationFitness CalcMutationFitnessType
}

// Mdl is the singleton instance of Models that can be accessed throughout the dna package. It gets set in SetModels().
var Mdl *Models


// SetModels is called by main.initialize() to set the function ptrs for the various algorithms chosen by the input file.
func SetModels(c *config.Config) {
	Mdl = &Models{} 		// create and set the singleton object
	var mdlNames []string 		// gather the models we use so we can print it out

	if c.Mutations.Uniform_fitness_effect_del != 0.0 {
		Mdl.CalcDelMutationFitness = CalcFixedDelMutationFitness
		mdlNames = append(mdlNames, "CalcFixedDelMutationFitness")
	} else {
		switch {
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(UNIFORM_FITNESS_EFFECT):
			Mdl.CalcDelMutationFitness = CalcUniformDelMutationFitness
			mdlNames = append(mdlNames, "CalcUniformDelMutationFitness")
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(WEIBULL_FITNESS_EFFECT):
			Mdl.CalcDelMutationFitness = CalcWeibullDelMutationFitness
			mdlNames = append(mdlNames, "CalcWeibullDelMutationFitness")
		default:
			log.Fatalf("Error: unrecognized value for fitness_effect_model: %v", c.Mutations.Fitness_effect_model)
		}
	}

	if c.Mutations.Uniform_fitness_effect_fav != 0.0 {
		Mdl.CalcFavMutationFitness = CalcFixedFavMutationFitness
		mdlNames = append(mdlNames, "CalcFixedFavMutationFitness")
	} else {
		switch {
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(UNIFORM_FITNESS_EFFECT):
			Mdl.CalcFavMutationFitness = CalcUniformFavMutationFitness
			mdlNames = append(mdlNames, "CalcUniformFavMutationFitness")
		case strings.ToLower(c.Mutations.Fitness_effect_model) == string(WEIBULL_FITNESS_EFFECT):
			Mdl.CalcFavMutationFitness = CalcWeibullFavMutationFitness
			mdlNames = append(mdlNames, "CalcWeibullFavMutationFitness")
		default:
			log.Fatalf("Error: unrecognized value for fitness_effect_model: %v", c.Mutations.Fitness_effect_model)
		}
	}

	config.Verbose(3, "Running with these dna models: %v", strings.Join(mdlNames, ", "))
}
