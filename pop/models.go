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
	//FORTRAN_NUM_OFFSPRING NumOffSpringModelType = "fortran" // this ended up giving the same results as FIXED_NUM_OFFSPRING
	FITNESS_NUM_OFFSPRING NumOffSpringModelType = "fitness"
)

type MutationRateModelType string
const (
	FIXED_MUTN_RATE MutationRateModelType = "fixed"
	POISSON_MUTN_RATE MutationRateModelType = "poisson"
)

type SelectionNoiseModelType string
const (
	FULL_TRUNC_SELECTION SelectionNoiseModelType = "fulltrunc"
	UNRESTRICT_PROB_SELECTION SelectionNoiseModelType = "ups"
	PROPORT_PROB_SELECTION SelectionNoiseModelType = "spps"
	PARTIAL_TRUNC_SELECTION SelectionNoiseModelType = "partialtrunc"
)


// Models holds pointers to functions that implement the various algorithms chosen by the input file.
type Models struct {
	CalcNumOffspring CalcNumOffspringType
	CalcIndivFitness CalcIndivFitnessType
	CalcNumMutations CalcNumMutationsType
	ApplySelectionNoise ApplySelectionNoiseType
}

// Mdl is the singleton instance of Models that can be accessed throughout the dna package. It gets set in SetModels().
var Mdl *Models


// SetModels is called by main.initialize() to set the function ptrs for the various algorithms chosen by the input file.
func SetModels(c *config.Config) {
	Mdl = &Models{} 		// create and set the singleton object
	var mdlNames []string 		// gather the models we use so we can print it out

	// uniform (even distribution), fixed (rounded to nearest int), fitness (weighted according to fitness)
	switch NumOffSpringModelType(strings.ToLower(c.Population.Num_offspring_model)) {
	case UNIFORM_NUM_OFFSPRING:
		Mdl.CalcNumOffspring = CalcUniformNumOffspring
		mdlNames = append(mdlNames, "CalcUniformNumOffspring")
	case FIXED_NUM_OFFSPRING:
		Mdl.CalcNumOffspring = CalcSemiFixedNumOffspring
		mdlNames = append(mdlNames, "CalcFixedNumOffspring")
	//case FORTRAN_NUM_OFFSPRING:
	//	Mdl.CalcNumOffspring = CalcFortranNumOffspring
	//	mdlNames = append(mdlNames, "CalcFortranNumOffspring")
	case FITNESS_NUM_OFFSPRING:
		Mdl.CalcNumOffspring = CalcFitnessNumOffspring
		mdlNames = append(mdlNames, "CalcFitnessNumOffspring")
	default:
		log.Fatalf("Error: unrecognized value for mum_offspring_model: %v", c.Population.Num_offspring_model)
	}

	if c.Mutations.Multiplicative_weighting > 0.0 {
		Mdl.CalcIndivFitness = MultIndivFitness
		mdlNames = append(mdlNames, "MultIndivFitness")
	} else {
		Mdl.CalcIndivFitness = SumIndivFitness
		mdlNames = append(mdlNames, "SumIndivFitness")
	}

	switch MutationRateModelType(strings.ToLower(c.Mutations.Mutn_rate_model)) {
	case FIXED_MUTN_RATE:
		Mdl.CalcNumMutations = CalcSemiFixedNumMutations
		mdlNames = append(mdlNames, "CalcSemiFixedNumMutations")
	case POISSON_MUTN_RATE:
		Mdl.CalcNumMutations = CalcPoissonNumMutations
		mdlNames = append(mdlNames, "CalcPoissonNumMutations")
	default:
		log.Fatalf("Error: unrecognized value for mutn_rate_model: %v", c.Mutations.Mutn_rate_model)
	}

	switch SelectionNoiseModelType(strings.ToLower(c.Selection.Selection_model)) {
	case FULL_TRUNC_SELECTION:
		Mdl.ApplySelectionNoise = ApplyFullTruncationNoise
		mdlNames = append(mdlNames, "ApplyFullTruncationNoise")
	case UNRESTRICT_PROB_SELECTION:
		Mdl.ApplySelectionNoise = ApplyUnrestrictProbNoise
		mdlNames = append(mdlNames, "ApplyUnrestrictProbNoise")
	case PROPORT_PROB_SELECTION:
		Mdl.ApplySelectionNoise = ApplyProportProbNoise
		mdlNames = append(mdlNames, "ApplyProportProbNoise")
	case PARTIAL_TRUNC_SELECTION:
		if c.Selection.Partial_truncation_value <= 0.0 { log.Fatalln("partial_truncation_value must be > 0") }	// we end up dividing by it
		Mdl.ApplySelectionNoise = ApplyPartialTruncationNoise
		mdlNames = append(mdlNames, "ApplyPartialTruncationNoise")
	default:
		log.Fatalf("Error: unrecognized value for selection_model: %v", c.Selection.Selection_model)
	}

	config.Verbose(1, "Running with these pop models: %v", strings.Join(mdlNames, ", "))
}
