package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"math"
	"math/rand"
	"log"
	//"unsafe"
)

type MutationType uint8
const (
	DELETERIOUS MutationType = iota
	NEUTRAL MutationType = iota
	FAVORABLE MutationType = iota
	DEL_ALLELE MutationType = iota
	FAV_ALLELE MutationType = iota
)


// A simple struct that is embedded in the LB arrays. (Not a ptr to it.)
type Mutation struct {
	Id uint64
	Type MutationType
	// When created, we add the fitness effect to the LB, so we don't have to store it here.
}

/* We don't get much benefit from having this as a base class (only 1 common field), and i think it is more efficient to
	to not have it. It is really the Mutator interface that allows us to have all of the subclasses in a single array...
// Mutation is the base class for all mutation types. It represents 1 mutation in 1 individual.
// We depend on mutations being immutable, once created, because we pass them from parent to child.
// To enforce that, the members are only available thru getter functions.
type Mutation struct {
    //fitnessEffect float64
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
	//mType         MutationType	// do not need to store this because it is represented by the Mutation subclasses
	//dominant      bool    // do not need to store this because we apply this influence during mutation factory
	//expressed     bool    // do not need to store this because we apply this influence during mutation factory
	//generation uint16	// the generation number in which this mutation was created. Used for statistics of how long the mutations survive in the pop.
}

// Member access methods
//func (m *Mutation) GetFitnessEffect() float64 { return float64(m.fitnessEffect) }
func (m *Mutation) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *Mutation) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }

// This interface enables us to hold subclasses of Mutation in a variable of the base class
type Mutator interface {
	//GetFitnessEffect() float64
	GetFitnessEffect() float32
	GetPointer() uintptr
}
*/


// The number of occurrences of each allele (both mutations and initial alleles) in 1 generation. Note: this is defined here instead of population.go to avoid circular dependencies
type AlleleCount struct {
	Deleterious         map[uint64]uint32    //map[uintptr]uint32
	Neutral         map[uint64]uint32    //map[uintptr]uint32
	Favorable         map[uint64]uint32    //map[uintptr]uint32
	DelInitialAlleles         map[uint64]uint32    //map[uintptr]uint32
	FavInitialAlleles         map[uint64]uint32    //map[uintptr]uint32
}

//func AlleleCountFactory(genNum, popSize uint32) *AlleleCount {
func AlleleCountFactory() *AlleleCount {
	ac := &AlleleCount{}
	ac.Deleterious = make(map[uint64]uint32)    //make(map[uintptr]uint32)
	ac.Neutral = make(map[uint64]uint32)    //make(map[uintptr]uint32)
	ac.Favorable = make(map[uint64]uint32)    //make(map[uintptr]uint32)
	ac.DelInitialAlleles = make(map[uint64]uint32)    //make(map[uintptr]uint32)
	ac.FavInitialAlleles = make(map[uint64]uint32)    //make(map[uintptr]uint32)
	return ac
}


// CalcMutationType determines if the next mutation should be deleterious/neutral/favorable based on a random number and the various relevant rates for this population.
// This is used by the LB to determine which of the Mutation subclasses to create.
func CalcMutationType(uniformRandom *rand.Rand) (mType MutationType) {

	// Determine if this mutation is deleterious, neutral, or favorable.
	// Frac_fav_mutn is the fraction of the non-neutral mutations that are favorable.
	rnd := uniformRandom.Float64()
	if rnd < config.Cfg.Mutations.Frac_fav_mutn * (1.0 - config.Cfg.Mutations.Fraction_neutral) {
		mType = FAVORABLE
	} else if rnd < 1.0 - config.Cfg.Mutations.Fraction_neutral {
		mType = DELETERIOUS
	} else {
		mType = NEUTRAL
	}

	/* This is the old/wrong way i was doing it...
	rnd := uniformRandom.Float64()
	if rnd < config.Cfg.Mutations.Frac_fav_mutn {
		mType = FAVORABLE
	} else if rnd < config.Cfg.Mutations.Frac_fav_mutn + config.Cfg.Mutations.Fraction_neutral {
		mType = NEUTRAL
	} else {
		mType = DELETERIOUS
	}
	*/
	return
}


// calcDelMutationAttrs determines the attributes of a new mutation, based on a random number and the config params.
// This is used in the subclass factory to initialize the base Mutation class members, and in LB AppendMutation() if it is untracked.
func calcDelMutationAttrs(uniformRandom *rand.Rand) (fitnessEffect float32) {
	// Determine if this mutation is dominant or recessive and use that to calc the fitness
	dominant := (config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64())
	if (dominant) {
		fitnessEffect = float32(Mdl.CalcDelMutationFitness(uniformRandom) * config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		fitnessEffect = float32(Mdl.CalcDelMutationFitness(uniformRandom) * config.Cfg.Mutations.Recessive_hetero_expression)
	}

	return
}


// calcFavMutationAttrs determines the attributes of a new mutation, based on a random number and the config params.
// This is used in the subclass factory to initialize the base Mutation class members, and in LB AppendMutation() if it is untracked.
func calcFavMutationAttrs(uniformRandom *rand.Rand) (fitnessEffect float32) {
	// Determine if this mutation is dominant or recessive and use that to calc the fitness
	dominant := (config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64())
	if (dominant) {
		fitnessEffect = float32(Mdl.CalcFavMutationFitness(uniformRandom) * config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		fitnessEffect = float32(Mdl.CalcFavMutationFitness(uniformRandom) * config.Cfg.Mutations.Recessive_hetero_expression)
	}

	return
}


// These are the different algorithms for assigning a fitness factor to a mutation. Pointers to 2 of them are chosen at initialization time.
type CalcMutationFitnessType func(uniformRandom *rand.Rand) float64
func CalcFixedDelMutationFitness(_ *rand.Rand) float64 { return -config.Cfg.Mutations.Uniform_fitness_effect_del }
func CalcFixedFavMutationFitness(_ *rand.Rand) float64 { return config.Cfg.Mutations.Uniform_fitness_effect_fav }

// Calculate a random fitness between -Uniform_fitness_effect_del and 0 (deleterious) or 0 and Uniform_fitness_effect_fav (favorable)
func CalcUniformDelMutationFitness(uniformRandom *rand.Rand) float64 {return -(uniformRandom.Float64() * config.Cfg.Mutations.Uniform_fitness_effect_del) }
func CalcUniformFavMutationFitness(uniformRandom *rand.Rand) float64 { return uniformRandom.Float64() * config.Cfg.Mutations.Uniform_fitness_effect_fav }

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311
func CalcWeibullDelMutationFitness(uniformRandom *rand.Rand) float64 {
	alphaDel := math.Log(config.Cfg.Mutations.Genome_size)
	gammaDel := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / alphaDel) /
	             math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)

	return -math.Exp(-alphaDel * math.Pow(uniformRandom.Float64(), gammaDel))
}

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311 and mutation.f90 line 104
func CalcWeibullFavMutationFitness(uniformRandom *rand.Rand) float64 {
	var alphaFav float64
	if config.Cfg.Mutations.Max_fav_fitness_gain > 0.0 {
		alphaFav = math.Log(config.Cfg.Mutations.Genome_size * config.Cfg.Mutations.Max_fav_fitness_gain)
	} else {
		alphaFav = math.Log(config.Cfg.Mutations.Genome_size)
	}
	gammaFav := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / alphaFav) /
	            math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)

	return config.Cfg.Mutations.Max_fav_fitness_gain * math.Exp(-alphaFav * math.Pow(uniformRandom.Float64(), gammaFav))
}


// These are the different algorithms for assigning a fitness factor to an initial allele. Pointers to 2 of them are chosen at initialization time.
type CalcAlleleFitnessType func(uniformRandom *rand.Rand) float64

func CalcUniformAlleleFitness(uniformRandom *rand.Rand) float64 {
	if config.Cfg.Population.Num_contrasting_alleles == 0 { log.Fatalln("System Error: CalcUniformAlleleFitness() called when Num_contrasting_alleles==0") }
	initial_alleles_mean_effect := config.Cfg.Population.Max_total_fitness_increase / float64(config.Cfg.Population.Num_contrasting_alleles)
	if config.Cfg.Population.Num_contrasting_alleles <= 10 {
		return initial_alleles_mean_effect		// the number of alleles is small enough that using uniformRandom probably won't give us a good average
	} else {
		return 2.0 * initial_alleles_mean_effect * uniformRandom.Float64()		// so the average works out to be initial_alleles_mean_effect
	}
}

/*
// DeleteriousMutation represents 1 deleterious mutation in 1 individual.
type DeleteriousMutation struct {
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
}

func DeleteriousMutationFactory(fitnessEffect float32, _ *rand.Rand) *DeleteriousMutation {
	return &DeleteriousMutation{fitnessEffect: fitnessEffect}
}

func (m *DeleteriousMutation) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *DeleteriousMutation) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }

// NeutralMutation represents 1 neutral mutation in 1 individual.
type NeutralMutation struct {
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
}

func NeutralMutationFactory(_ *rand.Rand) *NeutralMutation {
	return &NeutralMutation{}
}

func (m *NeutralMutation) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *NeutralMutation) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }

// FavorableMutation represents 1 favorable mutation in 1 individual.
type FavorableMutation struct {
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
}

func FavorableMutationFactory(fitnessEffect float32, _ *rand.Rand) *FavorableMutation {
	return &FavorableMutation{fitnessEffect: fitnessEffect}
}

func (m *FavorableMutation) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *FavorableMutation) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }


// DeleteriousAllele represents 1 deleterious allele in 1 individual.
type DeleteriousAllele struct {
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
}

func DeleteriousAlleleFactory(fitnessEffect float64) *DeleteriousAllele {
	return &DeleteriousAllele{fitnessEffect: float32(fitnessEffect)}
}

func (m *DeleteriousAllele) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *DeleteriousAllele) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }


// FavorableAllele represents 1 deleterious allele in 1 individual.
type FavorableAllele struct {
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
}

func FavorableAlleleFactory(fitnessEffect float64) *FavorableAllele {
	return &FavorableAllele{fitnessEffect: float32(fitnessEffect)}
}

func (m *FavorableAllele) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *FavorableAllele) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }
*/
