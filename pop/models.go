package pop

import (
	"github.com/genetic-algorithms/mendel-go/config"
	"strings"
	"log"
	"github.com/genetic-algorithms/mendel-go/dna"
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

type PopulationGrowthModelType string
const (
	NO_POPULATON_GROWTH PopulationGrowthModelType = "none"
	EXPONENTIAL_POPULATON_GROWTH PopulationGrowthModelType = "exponential"
	CAPACITY_POPULATON_GROWTH PopulationGrowthModelType = "capacity"
	FOUNDERS_POPULATON_GROWTH PopulationGrowthModelType = "founders"
)

type InitialAlleleModelType string
const (
	UNIFORM_INITIAL_ALLELES InitialAlleleModelType = "uniform"
	VARIABLE_FREQ_INITIAL_ALLELES InitialAlleleModelType = "variablefreq"
)


// Models holds pointers to functions that implement the various algorithms chosen by the input file.
type Models struct {
	CalcNumOffspring CalcNumOffspringType
	CalcIndivFitness CalcIndivFitnessType
	CalcNumMutations CalcNumMutationsType
	ApplySelectionNoise ApplySelectionNoiseType
	PopulationGrowth PopulationGrowthType
	GenerateInitialAlleles GenerateInitialAllelesType
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

	switch PopulationGrowthModelType(strings.ToLower(c.Population.Pop_growth_model)) {
	case NO_POPULATON_GROWTH:
		Mdl.PopulationGrowth = NoPopulationGrowth
		mdlNames = append(mdlNames, "NoPopulationGrowth")
	case EXPONENTIAL_POPULATON_GROWTH:
		Mdl.PopulationGrowth = ExponentialPopulationGrowth
		mdlNames = append(mdlNames, "ExponentialPopulationGrowth")
		if c.Population.Pop_growth_rate <= 0.0 { log.Fatalln("For pop_growth_model==exponential pop_growth_rate must be > 0.0") }
		if c.Basic.Num_generations == 0 && c.Population.Max_pop_size == 0 { log.Fatalln("For pop_growth_model==exponential at least 1 of num_generations and max_pop_size must be non-zero") }
	case CAPACITY_POPULATON_GROWTH:
		Mdl.PopulationGrowth = CapacityPopulationGrowth
		mdlNames = append(mdlNames, "CapacityPopulationGrowth")
		if c.Population.Pop_growth_rate <= 0.0 { log.Fatalln("For pop_growth_model==capacity pop_growth_rate must be > 0.0") }
	case FOUNDERS_POPULATON_GROWTH:
		Mdl.PopulationGrowth = FoundersPopulationGrowth
		mdlNames = append(mdlNames, "FoundersPopulationGrowth")
		if c.Population.Pop_growth_rate <= 0.0 || c.Population.Pop_growth_rate2 <= 0.0 { log.Fatalln("For pop_growth_model==founders pop_growth_rate and pop_growth_rate2 must be > 0.0") }
		if c.Population.Bottleneck_generation > 0 && (c.Population.Bottleneck_pop_size == 0 | c.Population.Num_bottleneck_generations) { log.Fatalln("For pop_growth_model==founders and bottleneck_generation > then bottleneck_pop_size and num_bottleneck_generations must be > 0.0") }
	default:
		log.Fatalf("Error: unrecognized value for pop_growth_model: %v", c.Population.Pop_growth_model)
	}

	switch InitialAlleleModelType(strings.ToLower(c.Population.Initial_allele_fitness_model)) {
	case UNIFORM_INITIAL_ALLELES:
		if c.Population.Num_contrasting_alleles > 0 && (c.Population.Initial_alleles_pop_frac <= 0.0 || c.Population.Initial_alleles_pop_frac > 1.0) { log.Fatalf("if num_contrasting_alleles is > 0 and initial_allele_fitness_model==%s, then initial_alleles_pop_frac must be > 0 and <= 1.0", string(UNIFORM_INITIAL_ALLELES)) }
		if c.Population.Num_contrasting_alleles > 0 && c.Population.Max_total_fitness_increase <= 0.0 { log.Fatalf("Error: if initial_allele_fitness_model==%s, then max_total_fitness_increase must be > 0.", string(UNIFORM_INITIAL_ALLELES)) }
		Mdl.GenerateInitialAlleles = GenerateUniformInitialAlleles
		mdlNames = append(mdlNames, "GenerateUniformInitialAlleles")
		dna.Mdl.CalcAlleleFitness = dna.CalcUniformAlleleFitness
		mdlNames = append(mdlNames, "CalcUniformAlleleFitness")
	case VARIABLE_FREQ_INITIAL_ALLELES:
		if c.Population.Num_contrasting_alleles > 0 && c.Population.Initial_alleles_frequencies == "" { log.Fatalf("if num_contrasting_alleles is > 0 and initial_allele_fitness_model==%s, then initial_alleles_frequencies must be like: alfrac1:freq1, alfrac2:freq2, ...", string(VARIABLE_FREQ_INITIAL_ALLELES)) }
		if c.Population.Num_contrasting_alleles > 0 && c.Population.Max_total_fitness_increase <= 0.0 { log.Fatalf("Error: if initial_allele_fitness_model==%s, then max_total_fitness_increase must be > 0.", string(VARIABLE_FREQ_INITIAL_ALLELES)) }
		Mdl.GenerateInitialAlleles = GenerateVariableFreqInitialAlleles
		mdlNames = append(mdlNames, "GenerateVariableFreqInitialAlleles")
		dna.Mdl.CalcAlleleFitness = dna.CalcUniformAlleleFitness
		mdlNames = append(mdlNames, "CalcUniformAlleleFitness")
	default:
		log.Fatalf("Error: unrecognized value for initial_allele_fitness_model: %v", c.Population.Initial_allele_fitness_model)
	}

	config.Verbose(1, "Running with these pop models: %v", strings.Join(mdlNames, ", "))
}
