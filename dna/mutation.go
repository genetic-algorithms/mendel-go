package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)

type MutationType uint8
const (
	DELETERIOUS MutationType = iota
	NEUTRAL MutationType = iota
	FAVORABLE MutationType = iota
)

// Mutation is the base class for all mutation types. It represents 1 mutation in 1 individual.
// We are depending on mutations being immutable, once created, because we pass them from parent to child.
// To enforce that, the fields are only available thru getter functions.
type Mutation struct {
	mType MutationType
	dominant bool 		// dominant or recessive mutation
	expressed bool 		// whether or not this allele is expressed, based on Dominant_hetero_expression and Recessive_hetero_expression
	fitnessFactor float32 	// this is intentionally not float64 to save space
}


// MutationFactory creates a random mutation factoring the various relevant rates for this population.
func MutationFactory(uniformRandom *rand.Rand) (m *Mutation) {
	m = &Mutation{}

	// Determine if this mutation is deleterious, neutral, or favorable
	rnd := uniformRandom.Float64()
	if rnd < config.Cfg.Mutations.Frac_fav_mutn {
		m.mType = FAVORABLE
	} else if rnd < config.Cfg.Mutations.Frac_fav_mutn + config.Cfg.Mutations.Fraction_neutral {
		m.mType = NEUTRAL
	} else {
		m.mType = DELETERIOUS
	}

	m.fitnessFactor = CalcFitnessFactor(*m, uniformRandom)

	// Determine if this mutation is dominant or recessive
	m.dominant = (config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64())

	// Determine if this mutated allele is expressed or not (whether if affects the fitness)
	rnd = uniformRandom.Float64()
	if (m.dominant) {
		m.expressed = (rnd < config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		m.expressed = (rnd < config.Cfg.Mutations.Recessive_hetero_expression)
	}
	return
}


// Member access methods
func (m *Mutation) GetMType() MutationType { return m.mType }
func (m *Mutation) GetDominant() bool { return m.dominant }
func (m *Mutation) GetExpressed() bool { return m.expressed }
func (m *Mutation) GetFitnessFactor() float32 { return m.fitnessFactor }


// CalcFitnessFactor determines the fitness factor this mutation contributes to the overall individual fitness, using the method specified in the input file.
// Note: this is not a method of the Mutation object, because that is immutable.
func CalcFitnessFactor(m Mutation, uniformRandom *rand.Rand) (fitnessFactor float32) {
	switch {
	case m.mType == DELETERIOUS:
		fitnessFactor = Alg.CalcDelMutationFitness(m, uniformRandom)
	case m.mType == FAVORABLE:
		fitnessFactor = Alg.CalcFavMutationFitness(m, uniformRandom)
	default:
		// else neutral is 0
	}
	return
}


// These are the different algorithms for assigning a fitness factor to a mutation. Pointers to 2 of them are chosen at initialization time.
type CalcMutationFitnessType func(m Mutation, uniformRandom *rand.Rand) float32
func CalcFixedDelMutationFitness(_ Mutation, _ *rand.Rand) float32 { return float32(0.0 - config.Cfg.Mutations.Uniform_fitness_effect_del) }
func CalcFixedFavMutationFitness(_ Mutation, _ *rand.Rand) float32 { return float32(config.Cfg.Mutations.Uniform_fitness_effect_fav) }

// Calculate a random fitness between -0.1 and 0 (deleterious) or 0 and 0.1 (favorable)
func CalcUniformDelMutationFitness(_ Mutation, uniformRandom *rand.Rand) float32 {return float32(0.0 - (uniformRandom.Float64() / 10) ) }
func CalcUniformFavMutationFitness(_ Mutation, uniformRandom *rand.Rand) float32 { return float32(uniformRandom.Float64() / 10) }

//todo: jon use the Weibull distribution in these. See init.f90 lines 300-311
func CalcWeibullDelMutationFitness(_ Mutation, uniformRandom *rand.Rand) float32 {
	utils.NotImplementedYet("CalcWeibullDelMutationFitness not implemented yet")
	return 0.0
}

func CalcWeibullFavMutationFitness(_ Mutation, uniformRandom *rand.Rand) float32 {
	utils.NotImplementedYet("CalcWeibullFavMutationFitness not implemented yet")
	return 0.0
}


/* waiting to see if we need distinct subclasses for the different types of mutation...
// DeleteriousMutation represents 1 deleterious mutation in 1 individual.
type DeleteriousMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func DeleteriousMutationFactory(dominant bool) *DeleteriousMutation {
	return &DeleteriousMutation{Mutation{Mtype: DELETERIOUS, Dominant: dominant}}
}

// NeutralMutation represents 1 neutral mutation in 1 individual.
type NeutralMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func NeutralMutationFactory(dominant bool) *NeutralMutation {
	return &NeutralMutation{Mutation{Mtype: NEUTRAL, Dominant: dominant}}
}

// FavorableMutation represents 1 favorable mutation in 1 individual.
type FavorableMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func FavorableMutationFactory(dominant bool) *FavorableMutation {
	return &FavorableMutation{Mutation{Mtype: FAVORABLE, Dominant: dominant}}
}
*/
