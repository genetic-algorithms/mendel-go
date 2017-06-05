package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"strings"
	"log"
)

type MutationFitnessModelType string
const (
	FIXED_FITNESS_EFFECT MutationFitnessModelType = "fixed"
	UNIFORM_FITNESS_EFFECT MutationFitnessModelType = "uniform"
	WEIBULL_FITNESS_EFFECT MutationFitnessModelType = "weibull"
)

type CrossoverModelType string
const (
	NO_CROSSOVER CrossoverModelType = "none"
	FULL_CROSSOVER CrossoverModelType = "full"
)


// Models holds pointers to functions that implement the various algorithms chosen by the input file.
type Models struct {
	CalcDelMutationFitness CalcMutationFitnessType
	CalcFavMutationFitness CalcMutationFitnessType
	Crossover CrossoverType
}

// Mdl is the singleton instance of Models that can be accessed throughout the dna package. It gets set in SetModels().
var Mdl *Models


// SetModels is called by main.initialize() to set the function ptrs for the various algorithms chosen by the input file.
func SetModels(c *config.Config) {
	Mdl = &Models{} 		// create and set the singleton object
	var mdlNames []string 		// gather the models we use so we can print it out

	switch MutationFitnessModelType(strings.ToLower(c.Mutations.Fitness_effect_model)) {
	case  FIXED_FITNESS_EFFECT:
		if c.Mutations.Uniform_fitness_effect_del == 0.0 { log.Fatal("Error: if fitness_effect_model==fixed, you must set uniform_fitness_effect_del to a non-zero value.") }
		Mdl.CalcDelMutationFitness = CalcFixedDelMutationFitness
		mdlNames = append(mdlNames, "CalcFixedDelMutationFitness")
	case UNIFORM_FITNESS_EFFECT:
		Mdl.CalcDelMutationFitness = CalcUniformDelMutationFitness
		mdlNames = append(mdlNames, "CalcUniformDelMutationFitness")
	case WEIBULL_FITNESS_EFFECT:
		Mdl.CalcDelMutationFitness = CalcWeibullDelMutationFitness
		mdlNames = append(mdlNames, "CalcWeibullDelMutationFitness")
	default:
		log.Fatalf("Error: unrecognized value for fitness_effect_model: %v", c.Mutations.Fitness_effect_model)
	}

	switch MutationFitnessModelType(strings.ToLower(c.Mutations.Fitness_effect_model)) {
	case FIXED_FITNESS_EFFECT:
		if c.Mutations.Uniform_fitness_effect_fav == 0.0 { log.Fatal("Error: if fitness_effect_model==fixed, you must set uniform_fitness_effect_fav to a non-zero value.") }
		Mdl.CalcFavMutationFitness = CalcFixedFavMutationFitness
		mdlNames = append(mdlNames, "CalcFixedFavMutationFitness")
	case UNIFORM_FITNESS_EFFECT:
		Mdl.CalcFavMutationFitness = CalcUniformFavMutationFitness
		mdlNames = append(mdlNames, "CalcUniformFavMutationFitness")
	case WEIBULL_FITNESS_EFFECT:
		Mdl.CalcFavMutationFitness = CalcWeibullFavMutationFitness
		mdlNames = append(mdlNames, "CalcWeibullFavMutationFitness")
	default:
		log.Fatalf("Error: unrecognized value for fitness_effect_model: %v", c.Mutations.Fitness_effect_model)
	}

	switch CrossoverModelType(strings.ToLower(c.Population.Crossover_model)) {
	case NO_CROSSOVER:
		Mdl.Crossover = NoCrossover
		mdlNames = append(mdlNames, "NoCrossover")
	case FULL_CROSSOVER:
		Mdl.Crossover = FullCrossover
		mdlNames = append(mdlNames, "FullCrossover")
	default:
		log.Fatalf("Error: unrecognized value for crossover_model: %v", c.Population.Crossover_model)
	}

	config.Verbose(2, "Running with these dna models: %v", strings.Join(mdlNames, ", "))
}
