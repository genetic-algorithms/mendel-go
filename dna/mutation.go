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
	rnd := random.Rnd.Float64() 	// use for several of the decisions below

	// Determine if this mutation is deleterious, neutral, or favorable
	if rnd < config.Cfg.Mutations.Frac_fav_mutn {
		m.mType = FAVORABLE
	} else if rnd < config.Cfg.Mutations.Frac_fav_mutn + config.Cfg.Mutations.Fraction_neutral {
		m.mType = NEUTRAL
	} else {
		m.mType = DELETERIOUS
	}

	// Determine if this mutation is dominant or recessive
	m.dominant = (config.Cfg.Mutations.Fraction_recessive < rnd)

	// Calculate fitness factor. Todo: this should be the correct mutation effect distribution
	if m.mType == DELETERIOUS {
		m.fitnessFactor = float32(rnd * -1.0)
	} else if m.mType == FAVORABLE {
		m.fitnessFactor = float32(rnd)
	}

	// Determine if this mutated allele is expressed or not (whether if affects the fitness)
	if (m.dominant) {
		m.expressed = (rnd < config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		m.expressed = (rnd < config.Cfg.Mutations.Recessive_hetero_expression)
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
