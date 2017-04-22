package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/random"
	"bitbucket.org/geneticentropy/mendel-go/config"
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
func MutationFactory() (m *Mutation) {
	m = &Mutation{}

	// Determine if this mutation is deleterious, neutral, or favorable
	rnd := random.Rnd.Float64()
	if rnd < config.Cfg.Mutations.Frac_fav_mutn {
		m.mType = FAVORABLE
	} else if rnd < config.Cfg.Mutations.Frac_fav_mutn + config.Cfg.Mutations.Fraction_neutral {
		m.mType = NEUTRAL
	} else {
		m.mType = DELETERIOUS
	}

	m.fitnessFactor = CalcFitnessFactor(m.mType)

	// Determine if this mutation is dominant or recessive
	m.dominant = (config.Cfg.Mutations.Fraction_recessive < random.Rnd.Float64())

	// Determine if this mutated allele is expressed or not (whether if affects the fitness)
	rnd = random.Rnd.Float64()
	if (m.dominant) {
		m.expressed = (rnd < config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		m.expressed = (rnd < config.Cfg.Mutations.Recessive_hetero_expression)
	}
	return
}

// CalcFitnessFactor determines the fitness factor using the method specfied in the input file.
// Note: this is not a method of the Mutation object, because that is immutable.
func CalcFitnessFactor(mType MutationType) (fitnessFactor float32) {
	switch {
	// Handle fixed mutation effect
	case mType == DELETERIOUS && config.Cfg.Mutations.Uniform_fitness_effect_del != 0.0:
		fitnessFactor = config.Cfg.Mutations.Uniform_fitness_effect_del
	case mType == FAVORABLE && config.Cfg.Mutations.Uniform_fitness_effect_fav != 0.0:
		fitnessFactor = config.Cfg.Mutations.Uniform_fitness_effect_fav
	default:
		//todo: use the correct mutation effect calculations. See init.f90 lines 300-311
		rnd := random.Rnd.Float64()
		if mType == DELETERIOUS {
			fitnessFactor = float32(0.0 - rnd)
		} else if mType == FAVORABLE {
			fitnessFactor = float32(rnd)
		}
		// else neutral is 0
	}
	return
}

func (m *Mutation) GetMType() MutationType { return m.mType }
func (m *Mutation) GetDominant() bool { return m.dominant }
func (m *Mutation) GetExpressed() bool { return m.expressed }
func (m *Mutation) GetFitnessFactor() float32 { return m.fitnessFactor }


/*
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
