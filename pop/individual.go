package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/dna"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"bitbucket.org/geneticentropy/mendel-go/random"
	"log"
	"math/rand"
)


// Individual represents 1 organism in the population, tracking its mutations and alleles.
type Individual struct {
	Pop *Population
	Fitness float32
	Dead bool 		// if true, selection has identified it for elimination
	// we are not currently modeling chromosomes, only a big array of LBs
	LinkagesFromDad []*dna.LinkageBlock
	LinkagesFromMom []*dna.LinkageBlock
}


func IndividualFactory(pop *Population) *Individual{
	ind := &Individual{
		Pop: pop,
		//Chromos: make([]*dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		LinkagesFromDad: make([]*dna.LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
		LinkagesFromMom: make([]*dna.LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
	}
	// Note: there is no need to allocate the mutation slices with backing arrays. That will happen automatically the 1st time they are
	//		appended to with append(). Altho we will prob eventually want to implement our own append function to do it in bigger chunks.
	//		See https://blog.golang.org/go-slices-usage-and-internals

	//todo: there is probably a faster way to initialize these arrays
	for i := range ind.LinkagesFromDad { ind.LinkagesFromDad[i] = dna.LinkageBlockFactory() }
	for i := range ind.LinkagesFromMom { ind.LinkagesFromMom[i] = dna.LinkageBlockFactory() }

	return ind
}


// Mate combines this person with the specified person to create a list of offspring.
func (ind *Individual) Mate(otherInd *Individual, uniformRandom *rand.Rand) []*Individual {
	if RecombinationType(config.Cfg.Population.Recombination_model) == CLONAL { utils.NotImplementedYet("Do not support CLONAL recombination yet") }
	actual_offspring := Alg.CalcNumOffspring(ind, uniformRandom)
	config.Verbose(8, " actual_offspring=%d", actual_offspring)
	offspr := make([]*Individual, actual_offspring)
	for child:=0; child<actual_offspring; child++ {
		offspr[child] = ind.OneOffspring(otherInd, uniformRandom)
	}
	return offspr
}


// Offspring returns 1 offspring of this person and the specified person.
// We assume ind is the dad and otherInd is the mom.
func (ind *Individual) OneOffspring(otherInd *Individual, uniformRandom *rand.Rand) *Individual {
	offspr := IndividualFactory(ind.Pop)
	//todo: support...
	// Set the number of segments.  Three linkgage blocks of the chromosome that are involved in the crossover.  Form the gametes chromosome by chromosome.
	//iseg_max := 3
	//if !config.Cfg.Population.Dynamic_linkage {
	//	iseg_max = 1  // can come from any parent
	//}

	// Inherit linkage blocks
	//chr_length := config.Cfg.Population.Num_linkage_subunits / config.Cfg.Population.Haploid_chromosome_number 		// num LBs in each chromosome
	//for chr:=1; chr<=config.Cfg.Population.Haploid_chromosome_number; chr++ {
	for lb:=0; lb<config.Cfg.Population.Num_linkage_subunits; lb++ {
		// randomly choose which parent to get the LB from
		if uniformRandom.Intn(2) == 0 {
			offspr.LinkagesFromDad[lb] = ind.LinkagesFromDad[lb].Copy()
		} else {
			offspr.LinkagesFromDad[lb] = ind.LinkagesFromMom[lb].Copy()
		}
		if uniformRandom.Intn(2) == 0 {
			offspr.LinkagesFromMom[lb] = otherInd.LinkagesFromDad[lb].Copy()
		} else {
			offspr.LinkagesFromMom[lb] = otherInd.LinkagesFromMom[lb].Copy()
		}
	}

	// Apply new mutations
	numMutations := Alg.CalcNumMutations(uniformRandom)
	for m:=1; m<=numMutations; m++ {
		lb := uniformRandom.Intn(config.Cfg.Population.Num_linkage_subunits)	// choose a random LB index

		// Randomly choose the LB from dad or mom to put the mutation in.
		// Note: MutationFactory() chooses deleterious/neutral/favorable, dominant/recessive, etc. based on the relevant rates
		if uniformRandom.Intn(2) == 0 {
			offspr.LinkagesFromDad[lb].Append(dna.MutationFactory(uniformRandom))
		} else {
			offspr.LinkagesFromMom[lb].Append(dna.MutationFactory(uniformRandom))
		}
	}
	//d, n, f := offspr.GetNumMutations()
	//config.Verbose(9, "my mutations including new ones: %d, %d, %d", d, n, f)

	offspr.Fitness = Alg.CalcIndivFitness(offspr) 		// store resulting fitness
	//if offspr.Fitness <= 0.0 { offspr.Dead = true } 	//todo: we want to do this, but have to adjust Population.Select() process so we don't kill off too many

	return offspr
}


// Various algorithms for determining the random number of offspring for an individual
type CalcNumOffspringType func(ind *Individual, uniformRandom *rand.Rand) int

// A uniform algorithm for calculating the number of offspring that gives an even distribution between 1 and 2*Num_offspring-1
func CalcUniformNumOffspring(ind *Individual, uniformRandom *rand.Rand) int {
	// If Num_offspring is 4.5, we want a range from 1-8
	maxRange := (2 * ind.Pop.Num_offspring) - 2 		// subtract 2 to get a buffer of 1 at each end
	numOffspring := uniformRandom.Float64() * maxRange 		// some float between 0 and maxRange
	return random.Round(uniformRandom, numOffspring + 1) 	// shift it so it is between 1 and maxRange+1, then get to an int
}


// An algorithm taken from the fortran mendel for calculating the number of offspring
func CalcFortranNumOffspring(ind *Individual, uniformRandom *rand.Rand) int {
	//todo: i do not understand some of this logic, it is from lines 64-73 of mating.f90
	actual_offspring := int(ind.Pop.Num_offspring)		// truncate num offspring to integer
	if ind.Pop.Num_offspring - float64(int(ind.Pop.Num_offspring)) > uniformRandom.Float64() { actual_offspring++ }	// randomly round it up or down
	//if indivIndex == 1 { actual_offspring = utils.Max(1, actual_offspring) } 	// assuming this was some special case specific to the fortran implementation
	actual_offspring = utils.Min(int(ind.Pop.Num_offspring) + 1, actual_offspring)
	return actual_offspring
}


// Algorithms for determining the number of additional mutations a specific offspring should be given
type CalcNumMutationsType func(uniformRandom *rand.Rand) int

// Randomly round Mutn_rate to the int below or above, proportional to how close it is to each (so the resulting average should be Mutn_rate
func CalcSemiFixedNumMutations (uniformRandom *rand.Rand) int {
	numMutations := random.Round(uniformRandom, config.Cfg.Mutations.Mutn_rate)
	return numMutations
}

// Use a poisson distribution to choose a number of mutations, with the mean of number of mutations for all individuals being Mutn_rate
func CalcPoissonNumMutations (uniformRandom *rand.Rand) int {
	//todo: jon, the num mutations should be a poisson distribution with the mean being Mutn_rate
	numMutations := 10
	utils.NotImplementedYet("CalcPoissonNumMutations not implemented yet")
	return numMutations
}


// Algorithms for aggregating all of the individual's mutation fitness factors into a single pheno fitness value
type CalcIndivFitnessType func(ind *Individual) float32

// SumIndivFitness adds together the fitness factors of all of the mutations. An individual's fitness starts at 1 and then deleterious
// mutations subtract from that and favorable mutations add to it. A total fitness of 0 means the individual is dead.
func SumIndivFitness(ind *Individual) (fitness float32) {
	// Sum all the LB fitness numbers
	fitness = 1.0
	for _, lb := range ind.LinkagesFromDad {
		for _, m := range lb.Mutn {
			// Note: the deleterious mutation fitness factors are already negative
			if (m.GetExpressed()) { fitness += m.GetFitnessFactor() }
		}
	}
	for _, lb := range ind.LinkagesFromMom {
		for _, m := range lb.Mutn {
			// Note: the deleterious mutation fitness factors are already negative
			if (m.GetExpressed()) { fitness += m.GetFitnessFactor() }
		}
	}
	return
}

// MultIndivFitness aggregates the fitness factors of all of the mutations using a combination of additive and mutliplicative,
// based on config.Cfg.Mutations.Multiplicative_weighting
func MultIndivFitness(_ *Individual) (fitness float32) {
	fitness = 1.0
	//todo: do not know the exact forumla to use for this yet
	utils.NotImplementedYet("Multiplicative_weighting not implemented yet")
	return fitness
}


// GetNumMutations returns the number of deleterious, neutral, favorable mutations, respectively
func (ind *Individual) GetNumMutations() (deleterious, neutral, favorable int) {
	for _,lb := range ind.LinkagesFromDad {
		delet, neut, fav := lb.GetNumMutations()
		deleterious += delet
		neutral += neut
		favorable += fav
	}
	for _,lb := range ind.LinkagesFromMom {
		delet, neut, fav := lb.GetNumMutations()
		deleterious += delet
		neutral += neut
		favorable += fav
	}
	return
}

// Report prints out statistics of this individual. If final==true is prints more details.
func (ind *Individual) Report(final bool) {
	deleterious, neutral, favorable := ind.GetNumMutations()
	log.Printf("  Ind: deleterious %d, neutral: %d, favorable: %d, fitness: %v", deleterious, neutral, favorable, ind.Fitness)
}
