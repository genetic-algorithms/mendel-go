package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"math"
	"math/rand"
)

//todo: use subclasses for Mutation and stop using this type
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
	mType         MutationType
	dominant      bool    // dominant or recessive mutation
	expressed     bool    // whether or not this allele is expressed, based on Dominant_hetero_expression and Recessive_hetero_expression
	fitnessEffect float64 // consider making this float32 to save space
}

// This interface enables us to hold subclasses of Mutation in a variable of the base class
type Mutator interface {
	GetMType() MutationType
	GetDominant() bool
	GetExpressed() bool
	GetFitnessEffect() float64
}


// Member access methods
func (m *Mutation) GetMType() MutationType { return m.mType }
func (m *Mutation) GetDominant() bool { return m.dominant }
func (m *Mutation) GetExpressed() bool { return m.expressed }
func (m *Mutation) GetFitnessEffect() float64 { return m.fitnessEffect
}


// CalcMutationType determines if the next mutation should be deleterious/neutral/favorable based on a random number and the various relevant rates for this population.
func CalcMutationType(uniformRandom *rand.Rand) (mType MutationType) {
	//m = &Mutation{}

	// Determine if this mutation is deleterious, neutral, or favorable
	rnd := uniformRandom.Float64()
	if rnd < config.Cfg.Mutations.Frac_fav_mutn {
		mType = FAVORABLE
		//m = FavorableMutationFactory(uniformRandom)
	} else if rnd < config.Cfg.Mutations.Frac_fav_mutn + config.Cfg.Mutations.Fraction_neutral {
		mType = NEUTRAL
		//m = NeutralMutationFactory(uniformRandom)
	} else {
		mType = DELETERIOUS
		//m = DeleteriousMutationFactory(uniformRandom)
	}
	return
}


// calcMutationAttrs determines the dominant and expressed attributes of a new mutation, based on a random number and the config params
func calcMutationAttrs(uniformRandom *rand.Rand) (dominant bool, expressed bool) {
	//m.fitnessEffect = CalcFitnessEffect(*m, uniformRandom)

	// Determine if this mutation is dominant or recessive
	dominant = (config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64())

	// Determine if this mutated allele is expressed or not (whether if affects the fitness)
	rnd := uniformRandom.Float64()
	if (dominant) {
		expressed = (rnd < config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		expressed = (rnd < config.Cfg.Mutations.Recessive_hetero_expression)
	}
	return
}


// CalcFitnessFactor determines the fitness factor this mutation contributes to the overall individual fitness, using the method specified in the input file.
// Note: this is not a method of the Mutation object, because that is immutable.
/*
func CalcFitnessEffect(m Mutation, uniformRandom *rand.Rand) (fitnessEffect float64) {
	switch {
	case m.mType == DELETERIOUS:
		fitnessFactor = Mdl.CalcDelMutationFitness(m, uniformRandom)
	case m.mType == FAVORABLE:
		fitnessFactor = Mdl.CalcFavMutationFitness(m, uniformRandom)
	default:
	// else neutral is 0
	}
	switch _ := m.(type) {
	case DeleteriousMutation:
		fitnessEffect = Mdl.CalcDelMutationFitness(m, uniformRandom)
	case FavorableMutation:
		fitnessEffect = Mdl.CalcFavMutationFitness(m, uniformRandom)
	default:
		// else neutral is 0
	}
	return
}
*/


// These are the different algorithms for assigning a fitness factor to a mutation. Pointers to 2 of them are chosen at initialization time.
type CalcMutationFitnessType func(m Mutator, uniformRandom *rand.Rand) float64
func CalcFixedDelMutationFitness(_ Mutator, _ *rand.Rand) float64 { return 0.0 - config.Cfg.Mutations.Uniform_fitness_effect_del }
func CalcFixedFavMutationFitness(_ Mutator, _ *rand.Rand) float64 { return config.Cfg.Mutations.Uniform_fitness_effect_fav }

// Calculate a random fitness between -0.1 and 0 (deleterious) or 0 and 0.1 (favorable)
func CalcUniformDelMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 {return 0.0 - (uniformRandom.Float64() / 10) }
func CalcUniformFavMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 { return uniformRandom.Float64() / 10 }

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311
func CalcWeibullDelMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 {
	alphaDel := math.Log(config.Cfg.Mutations.Genome_size)
	gammaDel := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / alphaDel) /
	             math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)

	return -math.Exp(-alphaDel * math.Pow(uniformRandom.Float64(), gammaDel))
}

// Algorithm according to Wes and the Fortran version
func CalcWeibullFavMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 {
	alphaFav := math.Log(config.Cfg.Mutations.Genome_size * config.Cfg.Mutations.Max_fav_fitness_gain)
	gammaFav := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / alphaFav) /
	            math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)

	return math.Exp(-alphaFav * math.Pow(uniformRandom.Float64(), gammaFav))
}


// DeleteriousMutation represents 1 deleterious mutation in 1 individual.
type DeleteriousMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func DeleteriousMutationFactory(uniformRandom *rand.Rand) *DeleteriousMutation {
	m := &DeleteriousMutation{Mutation{mType: DELETERIOUS}}
	m.fitnessEffect = Mdl.CalcDelMutationFitness(m, uniformRandom)
	m.dominant, m.expressed = calcMutationAttrs(uniformRandom)
	return m
}

// NeutralMutation represents 1 neutral mutation in 1 individual.
type NeutralMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func NeutralMutationFactory(uniformRandom *rand.Rand) *NeutralMutation {
	m := &NeutralMutation{Mutation{mType: NEUTRAL}}
	m.dominant, m.expressed = calcMutationAttrs(uniformRandom)
	return m
}

// FavorableMutation represents 1 favorable mutation in 1 individual.
type FavorableMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func FavorableMutationFactory(uniformRandom *rand.Rand) *FavorableMutation {
	m := &FavorableMutation{Mutation{mType: FAVORABLE}}
	m.fitnessEffect = Mdl.CalcFavMutationFitness(m, uniformRandom)
	m.dominant, m.expressed = calcMutationAttrs(uniformRandom)
	return m
}
