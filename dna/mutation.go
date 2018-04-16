package dna

import (
	"github.com/genetic-algorithms/mendel-go/config"
	"math"
	"math/rand"
	"log"
	//"unsafe"
	"github.com/genetic-algorithms/mendel-go/utils"
)

// Note: we have a lot of mutations, so to keep the size of each to a min, store del/fav and dom/rec in same enum
type MutationType uint8
const (
	DELETERIOUS_DOMINANT MutationType = iota
	DELETERIOUS_RECESSIVE MutationType = iota
	NEUTRAL MutationType = iota
	FAVORABLE_DOMINANT MutationType = iota
	FAVORABLE_RECESSIVE MutationType = iota
	DEL_ALLELE MutationType = iota  // Note: for now we assume that all initial contrasting alleles are co-dominant, so we don't have to store dominant/recessive
	FAV_ALLELE MutationType = iota
)


// A simple struct that is embedded in the LB arrays. (Not a ptr to it.) A lot of mutations exist, so need to keep its size to a minimum.
type Mutation struct {
	Id uint64
	Type MutationType
	FitnessEffect float32	// even tho we accumulate the fitness in the LB as we go, we need to save this for allele analysis
}

/* We don't get much benefit from having this as a base class (only 1 common field), and i think it is more efficient to
	to not have it. It is really the Mutator interface that allows us to have all of the subclasses in a single array...
// Mutation is the base class for all mutation types. It represents 1 mutation in 1 individual.
// We depend on mutations being immutable, once created, because we pass them from parent to child.
// To enforce that, the members are only available thru getter functions.
type Mutation struct {
	fitnessEffect float32 // this is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 9th decimal place.
}

// Member access methods
func (m *Mutation) GetFitnessEffect() float32 { return m.fitnessEffect }
func (m *Mutation) GetPointer() uintptr { return uintptr(unsafe.Pointer(m)) }

// This interface enables us to hold subclasses of Mutation in a variable of the base class
type Mutator interface {
	GetFitnessEffect() float32
	GetPointer() uintptr
}
*/

type Allele struct {
	Count         uint32
	FitnessEffect float32
}

// The number of occurrences of each allele (both mutations and initial alleles) in 1 generation. The map key is the unique id of mutation.
// Note: this is defined here instead of population.go to avoid circular dependencies
type AlleleCount struct {
	DeleteriousDom         map[uint64]Allele
	DeleteriousRec         map[uint64]Allele
	Neutral         map[uint64]Allele
	FavorableDom         map[uint64]Allele
	FavorableRec         map[uint64]Allele
	DelInitialAlleles         map[uint64]Allele
	FavInitialAlleles         map[uint64]Allele
}

func AlleleCountFactory() *AlleleCount {
	ac := &AlleleCount{}
	ac.DeleteriousDom = make(map[uint64]Allele)
	ac.DeleteriousRec = make(map[uint64]Allele)
	ac.Neutral = make(map[uint64]Allele)
	ac.FavorableDom = make(map[uint64]Allele)
	ac.FavorableRec = make(map[uint64]Allele)
	ac.DelInitialAlleles = make(map[uint64]Allele)
	ac.FavInitialAlleles = make(map[uint64]Allele)
	return ac
}


// CalcMutationType determines if the next mutation should be deleterious/neutral/favorable based on a random number and the various relevant rates for this population.
// This is used by the LB to determine which of the Mutation subclasses to create.
func CalcMutationType(uniformRandom *rand.Rand) (mType MutationType) {

	// Determine if this mutation is deleterious, neutral, or favorable.
	// Frac_fav_mutn is the fraction of the non-neutral mutations that are favorable.
	rnd := uniformRandom.Float64()
	if rnd < config.Cfg.Mutations.Frac_fav_mutn * (1.0 - config.Cfg.Mutations.Fraction_neutral) {
		dominant := config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64()
		if dominant {
			mType = FAVORABLE_DOMINANT
		} else {
			mType = FAVORABLE_RECESSIVE
		}
	} else if rnd < 1.0 - config.Cfg.Mutations.Fraction_neutral {
		dominant := config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64()
		if dominant {
			mType = DELETERIOUS_DOMINANT
		} else {
			mType = DELETERIOUS_RECESSIVE
		}
	} else {
		mType = NEUTRAL
	}
	return
}


// calcDelMutationAttrs determines the attributes of a new mutation, based on a random number and the config params.
// This is used in the subclass factory to initialize the base Mutation class members, and in LB AppendMutation() if it is untracked.
//func calcDelMutationAttrs(uniformRandom *rand.Rand) (fitnessEffect float32) {
func calcDelMutationAttrs(mType MutationType, uniformRandom *rand.Rand) (fitnessEffect float32) {
	// Determine if this mutation is dominant or recessive and use that to calc the fitness
	//dominant := config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64()
	if mType == DELETERIOUS_DOMINANT {
		fitnessEffect = float32(Mdl.CalcDelMutationFitness(uniformRandom) * config.Cfg.Mutations.Dominant_hetero_expression)
	} else {
		fitnessEffect = float32(Mdl.CalcDelMutationFitness(uniformRandom) * config.Cfg.Mutations.Recessive_hetero_expression)
	}

	return
}


// calcFavMutationAttrs determines the attributes of a new mutation, based on a random number and the config params.
// This is used in the subclass factory to initialize the base Mutation class members, and in LB AppendMutation() if it is untracked.
//func calcFavMutationAttrs(uniformRandom *rand.Rand) (fitnessEffect float32) {
func calcFavMutationAttrs(mType MutationType, uniformRandom *rand.Rand) (fitnessEffect float32) {
	// Determine if this mutation is dominant or recessive and use that to calc the fitness
	//dominant := config.Cfg.Mutations.Fraction_recessive < uniformRandom.Float64()
	if mType == FAVORABLE_DOMINANT {
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

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311 and mutation.f90 lines 102-109
func CalcWeibullDelMutationFitness(uniformRandom *rand.Rand) float64 {
	//alphaDel := math.Log(config.Cfg.Mutations.Genome_size)
	//gammaDel := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / config.Computed.alpha_del) /
	//             math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)

	return -math.Exp( -config.Computed.Alpha_del * math.Pow(uniformRandom.Float64(),config.Computed.Gamma_del) )
}

// Algorithm according to Wes and the Fortran version. See init.f90 lines 300-311 and mutation.f90 line 104
func CalcWeibullFavMutationFitness(uniformRandom *rand.Rand) float64 {
	/* these are now computed in config.ComputedValuesFactory()
	var alphaFav float64
	if config.Cfg.Mutations.Max_fav_fitness_gain > 0.0 {
		alphaFav = math.Log(config.Cfg.Mutations.Genome_size * config.Cfg.Mutations.Max_fav_fitness_gain)
	} else {
		alphaFav = math.Log(config.Cfg.Mutations.Genome_size)
	}
	gammaFav := math.Log(-math.Log(config.Cfg.Mutations.High_impact_mutn_threshold) / alphaFav) /
	            math.Log(config.Cfg.Mutations.High_impact_mutn_fraction)
	*/

	return config.Cfg.Mutations.Max_fav_fitness_gain * math.Exp(-config.Computed.Alpha_fav * math.Pow(uniformRandom.Float64(), config.Computed.Gamma_fav))
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

func CreateInitialAllelePair(uniqueInt *utils.UniqueInt, uniformRandom *rand.Rand) (favMutn, delMutn Mutation) {
	// Note: for now we assume that all initial contrasting alleles are co-dominant so that in the homozygous case (2 of the same favorable
	//		allele (or 2 of the deleterious allele) - 1 from each parent), the combined fitness effect is 1.0 * the allele fitness.
	expression := 0.5
	fitnessEffect := Mdl.CalcAlleleFitness(uniformRandom) * expression

	favMutn = Mutation{Id: uniqueInt.NextInt(), Type: FAV_ALLELE, FitnessEffect: float32(fitnessEffect)}
	delMutn = Mutation{Id: uniqueInt.NextInt(), Type: DEL_ALLELE, FitnessEffect: float32(-fitnessEffect)}
	return
}
