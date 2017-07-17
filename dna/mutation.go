package dna

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"math"
	"math/rand"
	"log"
)

type MutationType uint8
const (
	DELETERIOUS MutationType = iota
	NEUTRAL MutationType = iota
	FAVORABLE MutationType = iota
	/* do not think we need these...
	DEL_ALLELE MutationType = iota
	FAV_ALLELE MutationType = iota
	*/
)

// Mutation is the base class for all mutation types. It represents 1 mutation in 1 individual.
// We depend on mutations being immutable, once created, because we pass them from parent to child.
// To enforce that, the members are only available thru getter functions.
type Mutation struct {
	//mType         MutationType
	//dominant      bool    // dominant or recessive mutation <- i do not think i need to save this, because its effect is embodied in expressed. Do we need to save this info??
	//expressed     bool    // whether or not this allele is expressed, based on Dominant_hetero_expression and Recessive_hetero_expression
	fitnessEffect float64 // consider making this float32 to save space
	//generation uint16	// the generation number in which this mutation was created. Used for statistics of how long the mutations survive in the pop.
}

// This interface enables us to hold subclasses of Mutation in a variable of the base class
type Mutator interface {
	//GetMType() MutationType
	//GetDominant() bool
	//GetExpressed() bool
	GetFitnessEffect() float64
}

// All the alleles (both mutations and initial alleles) for 1 generation. Note: this is defined here instead of population.go to avoid circular dependencies
type Alleles struct {
	GenerationNumber     uint32     `json:"generationNumber"`
	Deleterious         []uintptr `json:"deleterious"`
	Neutral         []uintptr `json:"neutral"`
	Favorable         []uintptr `json:"favorable"`
	DelInitialAlleles         []uintptr `json:"delInitialAlleles"`
	FavInitialAlleles         []uintptr `json:"favInitialAlleles"`
}

// The number of occurrences of each allele (both mutations and initial alleles) in 1 generation. Note: this is defined here instead of population.go to avoid circular dependencies
type AlleleCount struct {
	GenerationNumber     uint32     `json:"generationNumber"`
	Deleterious         map[uintptr]uint32 `json:"deleterious"`
	Neutral         map[uintptr]uint32 `json:"neutral"`
	Favorable         map[uintptr]uint32 `json:"favorable"`
	DelInitialAlleles         map[uintptr]uint32 `json:"delInitialAlleles"`
	FavInitialAlleles         map[uintptr]uint32 `json:"favInitialAlleles"`
}

func AlleleCountFactory(genNum uint32) *AlleleCount {
	ac := &AlleleCount{GenerationNumber: genNum}
	ac.Deleterious = make(map[uintptr]uint32)
	ac.Neutral = make(map[uintptr]uint32)
	ac.Favorable = make(map[uintptr]uint32)
	ac.DelInitialAlleles = make(map[uintptr]uint32)
	ac.FavInitialAlleles = make(map[uintptr]uint32)
	return ac
}


// Member access methods
//func (m *Mutation) GetMType() MutationType { return m.mType }
//func (m *Mutation) GetDominant() bool { return m.dominant }
//func (m *Mutation) GetExpressed() bool { return m.expressed }
func (m *Mutation) GetFitnessEffect() float64 { return m.fitnessEffect
}


// CalcMutationType determines if the next mutation should be deleterious/neutral/favorable based on a random number and the various relevant rates for this population.
// This is used by the LB to determine which of the Mutation subclasses to create.
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


// calcMutationAttrs determines the dominant and expressed attributes of a new mutation, based on a random number and the config params.
// This is used in the subclass factories to initialize the base Mutation class members.
//func calcMutationAttrs(uniformRandom *rand.Rand) (expressed bool) {
func calcMutationAttrs(uniformRandom *rand.Rand) (dominant bool) {
	//m.fitnessEffect = CalcFitnessEffect(*m, uniformRandom)

	// Determine if this mutation is dominant or recessive
	dominant = (config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64())

	// Determine if this mutated allele is expressed or not (whether if affects the fitness)
	//rnd := uniformRandom.Float64()
	//if (dominant) {
	//	expressed = (rnd < config.Cfg.Mutations.Dominant_hetero_expression)
	//} else {
	//	expressed = (rnd < config.Cfg.Mutations.Recessive_hetero_expression)
	//}
	return
}


// These are the different algorithms for assigning a fitness factor to a mutation. Pointers to 2 of them are chosen at initialization time.
type CalcMutationFitnessType func(m Mutator, uniformRandom *rand.Rand) float64
func CalcFixedDelMutationFitness(_ Mutator, _ *rand.Rand) float64 { return -config.Cfg.Mutations.Uniform_fitness_effect_del }
func CalcFixedFavMutationFitness(_ Mutator, _ *rand.Rand) float64 { return config.Cfg.Mutations.Uniform_fitness_effect_fav }

// Calculate a random fitness between -Uniform_fitness_effect_del and 0 (deleterious) or 0 and Uniform_fitness_effect_fav (favorable)
func CalcUniformDelMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 {return -(uniformRandom.Float64() * config.Cfg.Mutations.Uniform_fitness_effect_del) }
func CalcUniformFavMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 { return uniformRandom.Float64() * config.Cfg.Mutations.Uniform_fitness_effect_fav }

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311
func CalcWeibullDelMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 {
	alphaDel := math.Log(config.Cfg.Mutations.Genome_size)
	gammaDel := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / alphaDel) /
	             math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)

	return -math.Exp(-alphaDel * math.Pow(uniformRandom.Float64(), gammaDel))
}

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311 and mutation.f90 line 104
func CalcWeibullFavMutationFitness(_ Mutator, uniformRandom *rand.Rand) float64 {
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



/* To switch on Mutation subclass, do something like this:
	switch _ := m.(type) {
	case DeleteriousMutation:
		fitnessEffect = Mdl.CalcDelMutationFitness(m, uniformRandom)
	case FavorableMutation:
		fitnessEffect = Mdl.CalcFavMutationFitness(m, uniformRandom)
	default:
		// else neutral is 0
	}
*/


// DeleteriousMutation represents 1 deleterious mutation in 1 individual.
type DeleteriousMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func DeleteriousMutationFactory(uniformRandom *rand.Rand) *DeleteriousMutation {
	m := &DeleteriousMutation{Mutation{}}
	dominant := calcMutationAttrs(uniformRandom)
	if (dominant) {
		m.fitnessEffect = Mdl.CalcDelMutationFitness(m, uniformRandom) * config.Cfg.Mutations.Dominant_hetero_expression
	} else {
		m.fitnessEffect = Mdl.CalcDelMutationFitness(m, uniformRandom) * config.Cfg.Mutations.Recessive_hetero_expression
	}
	return m
}

// NeutralMutation represents 1 neutral mutation in 1 individual.
type NeutralMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func NeutralMutationFactory(_ *rand.Rand) *NeutralMutation {
	return &NeutralMutation{Mutation{}}
}

// FavorableMutation represents 1 favorable mutation in 1 individual.
type FavorableMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func FavorableMutationFactory(uniformRandom *rand.Rand) *FavorableMutation {
	m := &FavorableMutation{Mutation{}}
	dominant := calcMutationAttrs(uniformRandom)
	if (dominant) {
		m.fitnessEffect = Mdl.CalcFavMutationFitness(m, uniformRandom) * config.Cfg.Mutations.Dominant_hetero_expression
	} else {
		m.fitnessEffect = Mdl.CalcFavMutationFitness(m, uniformRandom) * config.Cfg.Mutations.Recessive_hetero_expression
	}
	return m
}


// DeleteriousAllele represents 1 deleterious allele in 1 individual.
type DeleteriousAllele struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func DeleteriousAlleleFactory(fitnessEffect float64) *DeleteriousAllele {
	return &DeleteriousAllele{Mutation{fitnessEffect: fitnessEffect}}
}


// NeutralAllele represents 1 neutral allele in 1 individual.
type NeutralAllele struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func NeutralAlleleFactory() *NeutralAllele {
	return &NeutralAllele{Mutation{}}
}


// DeleteriousAllele represents 1 deleterious allele in 1 individual.
type FavorableAllele struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func FavorableAlleleFactory(fitnessEffect float64) *FavorableAllele {
	return &FavorableAllele{Mutation{fitnessEffect: fitnessEffect}}
}
