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
	Pop             *Population
	GenoFitness     float64		// fitness due to genomic mutations
	PhenoFitness     float64		// fitness due to GenoFitness plus environmental noise and selection noise
	Dead            bool 		// if true, selection has identified it for elimination
	// we are not currently modeling chromosomes, only a big array of LBs
	ChromosomesFromDad []*dna.Chromosome
	ChromosomesFromMom []*dna.Chromosome
	//LinkagesFromDad []*dna.LinkageBlock
	//LinkagesFromMom []*dna.LinkageBlock
}


func IndividualFactory(pop *Population, initialize bool) *Individual{
	ind := &Individual{
		Pop: pop,
		ChromosomesFromDad: make([]*dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		ChromosomesFromMom: make([]*dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		//LinkagesFromDad: make([]*dna.LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
		//LinkagesFromMom: make([]*dna.LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
	}

	// If this is gen 0, initialize with empty chromosomes and linkage blocks (with no mutations).
	// Otherwise this is an offspring that will get its chromosomes and LBs later from meiosis.
	if initialize {
		for c := range ind.ChromosomesFromDad { ind.ChromosomesFromDad[c] = dna.ChromosomeFactory(pop.LBsPerChromosome, true) }
		for c := range ind.ChromosomesFromMom { ind.ChromosomesFromMom[c] = dna.ChromosomeFactory(pop.LBsPerChromosome, true) }
		//for i := range ind.LinkagesFromDad { ind.LinkagesFromDad[i] = dna.LinkageBlockFactory() }
		//for i := range ind.LinkagesFromMom { ind.LinkagesFromMom[i] = dna.LinkageBlockFactory() }
	}

	return ind
}


// GetNumChromosomes returns the number of chromosomes from each parent (we assume they always have the same number from each parent)
func (ind *Individual) GetNumChromosomes() uint32 { return uint32(len(ind.ChromosomesFromDad)) }


// Mate combines this person with the specified person to create a list of offspring.
func (ind *Individual) Mate(otherInd *Individual, uniformRandom *rand.Rand) []*Individual {
	if RecombinationType(config.Cfg.Population.Recombination_model) != FULL_SEXUAL { utils.NotImplementedYet("Recombination models other than FULL_SEXUAL are not yet supported") }
	actual_offspring := Mdl.CalcNumOffspring(ind, uniformRandom)
	//config.Verbose(9, " actual_offspring=%d", actual_offspring)
	offspr := make([]*Individual, actual_offspring)
	for child:=uint32(0); child<actual_offspring; child++ {
		offspr[child] = ind.OneOffspring(otherInd, uniformRandom)
	}
	return offspr
}


// Offspring returns 1 offspring of this person (dad) and the specified person (mom).
func (dad *Individual) OneOffspring(mom *Individual, uniformRandom *rand.Rand) *Individual {
	offspr := IndividualFactory(dad.Pop, false)
	//todo: make sure we are covering all of the dynamic_linkage cases. This is from mating.f90, line 335:
	// Set the number of segments.  Three linkgage blocks of the chromosome that are involved in the crossover.  Form the gametes chromosome by chromosome.
	//iseg_max := 3
	//if !config.Cfg.Population.Dynamic_linkage {
	//	iseg_max = 1  // can come from any parent
	//}

	// Inherit linkage blocks
	for c:=uint32(0); c<dad.GetNumChromosomes(); c++ {
		// Meiosis() implements the crossover model specified in the config file
		config.Verbose(9, "Copying chromosomes from dad...")
		offspr.ChromosomesFromDad[c] = dad.ChromosomesFromDad[c].Meiosis(dad.ChromosomesFromMom[c], dad.Pop.LBsPerChromosome, uniformRandom)
		config.Verbose(9, "Copying chromosomes from mom...")
		offspr.ChromosomesFromMom[c] = mom.ChromosomesFromDad[c].Meiosis(mom.ChromosomesFromMom[c], dad.Pop.LBsPerChromosome, uniformRandom)
	}

	// Apply new mutations
	numMutations := Mdl.CalcNumMutations(uniformRandom)
	for m:=uint32(1); m<=numMutations; m++ {
		// Note: we are choosing the LB this way to keep the random number generation the same as when we didn't have chromosomes.
		//		Can change this in the future if you want.
		lb := uniformRandom.Intn(int(config.Cfg.Population.Num_linkage_subunits))	// choose a random LB within the individual
		chr := lb / int(dad.Pop.LBsPerChromosome) 		// get the chromosome index
		lbInChr := lb % int(dad.Pop.LBsPerChromosome)	// get index of LB within the chromosome
		//lb := uniformRandom.Intn(int(dad.GetNumLinkages()))	// choose a random LB index

		// Randomly choose the LB from dad or mom to put the mutation in.
		// Note: AppendMutation() creates a mutation with deleterious/neutral/favorable, dominant/recessive, etc. based on the relevant input parameter rates
		if uniformRandom.Intn(2) == 0 {
			offspr.ChromosomesFromDad[chr].AppendMutation(lbInChr, uniformRandom)
		} else {
			offspr.ChromosomesFromMom[chr].AppendMutation(lbInChr, uniformRandom)
		}
	}
	//d, n, f := offspr.GetNumMutations()
	//config.Verbose(9, "my mutations including new ones: %d, %d, %d", d, n, f)

	offspr.GenoFitness = Mdl.CalcIndivFitness(offspr) 		// store resulting fitness
	if offspr.GenoFitness <= 0.0 { offspr.Dead = true }

	return offspr
}


// Various algorithms for determining the random number of offspring for an individual
type CalcNumOffspringType func(ind *Individual, uniformRandom *rand.Rand) uint32

// A uniform algorithm for calculating the number of offspring that gives an even distribution between 1 and 2*Num_offspring-1
func CalcUniformNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	// If Num_offspring is 4.5, we want a range from 1-8
	maxRange := (2 * ind.Pop.Num_offspring) - 2 		// subtract 2 to get a buffer of 1 at each end
	numOffspring := uniformRandom.Float64() * maxRange 		// some float between 0 and maxRange
	return uint32(random.Round(uniformRandom, numOffspring + 1)) 	// shift it so it is between 1 and maxRange+1, then get to an uint32
}


// Randomly rounds the desired number of offspring to the integer below or above, proportional to how close it is to each (so the resulting average should be Num_offspring)
func CalcSemiFixedNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	return uint32(random.Round(uniformRandom, ind.Pop.Num_offspring))
}


// An algorithm taken from the fortran mendel for calculating the number of offspring
func CalcFortranNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	// I am still not sure about the rationale of some of this logic, it is from lines 64-73 of mating.f90
	actual_offspring := uint32(ind.Pop.Num_offspring)		// truncate num offspring to integer
	if ind.Pop.Num_offspring - float64(uint32(ind.Pop.Num_offspring)) > uniformRandom.Float64() { actual_offspring++ }	// randomly round it up sometimes
	//if indivIndex == 1 { actual_offspring = utils.Max(1, actual_offspring) } 	// assuming this was some special case specific to the fortran implementation
	actual_offspring = utils.MinUint32(uint32(ind.Pop.Num_offspring+1), actual_offspring) 	// does not seem like this line does anything, because actual_offspring will always be uint32(ind.Pop.Num_offspring)+1 or uint32(ind.Pop.Num_offspring)
	return actual_offspring
}


// Randomly choose a number of offspring that is, on average, proportional to the individual's fitness
func CalcFitnessNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	// in the fortran version this is controlled by fitness_dependent_fertility
	utils.NotImplementedYet("CalcFitnessNumOffspring not implemented yet")
	return uint32(random.Round(uniformRandom, ind.Pop.Num_offspring))
}


// Algorithms for determining the number of additional mutations a specific offspring should be given
type CalcNumMutationsType func(uniformRandom *rand.Rand) uint32

// Randomly round Mutn_rate to the uint32 below or above, proportional to how close it is to each (so the resulting average should be Mutn_rate)
func CalcSemiFixedNumMutations (uniformRandom *rand.Rand) uint32 {
	numMutations := uint32(random.Round(uniformRandom, config.Cfg.Mutations.Mutn_rate))
	return numMutations
}

// Use a poisson distribution to choose a number of mutations, with the mean of number of mutations for all individuals being Mutn_rate
func CalcPoissonNumMutations (uniformRandom *rand.Rand) uint32 {
	return uint32(random.Poisson(uniformRandom, config.Cfg.Mutations.Mutn_rate))
}


// Algorithms for aggregating all of the individual's mutation fitness factors into a single geno fitness value
type CalcIndivFitnessType func(ind *Individual) float64

// SumIndivFitness adds together the fitness factors of all of the mutations. An individual's fitness starts at 1 and then deleterious
// mutations subtract from that and favorable mutations add to it. A total fitness of 0 means the individual is dead.
func SumIndivFitness(ind *Individual) (fitness float64) {
	// Sum all the chromosome fitness numbers
	fitness = 1.0
	for _, c := range ind.ChromosomesFromDad {
		// Note: the deleterious mutation fitness factors are already negative
		fitness += c.SumFitness()
		//for _, m := range lb.DMutn { if (m.GetExpressed()) { fitness += m.GetFitnessEffect() } }
		//for _, m := range lb.FMutn { if (m.GetExpressed()) { fitness += m.GetFitnessEffect() } }
	}
	for _, c := range ind.ChromosomesFromMom {
		fitness += c.SumFitness()
		//for _, m := range lb.DMutn {	if (m.GetExpressed()) { fitness += m.GetFitnessEffect() } }
		//for _, m := range lb.FMutn {	if (m.GetExpressed()) { fitness += m.GetFitnessEffect() } }
	}
	return
}

// MultIndivFitness aggregates the fitness factors of all of the mutations using a combination of additive and mutliplicative,
// based on config.Cfg.Mutations.Multiplicative_weighting
func MultIndivFitness(_ *Individual) (fitness float64) {
	fitness = 1.0
	//todo: do not know the exact formula to use for this yet
	utils.NotImplementedYet("Multiplicative_weighting not implemented yet")
	return fitness
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (ind *Individual) GetMutationStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Calc the average of each type of mutation: multiply the average from each LB and num mutns from each LB, then at the end divide by total num mutns
	for _, c := range ind.ChromosomesFromDad {
		delet, neut, fav, avD, avF := c.GetMutationStats()
		deleterious += delet
		neutral += neut
		favorable += fav
		avDelFit += (float64(delet) * avD)
		avFavFit += (float64(fav) * avF)
	}
	for _, c := range ind.ChromosomesFromMom {
		delet, neut, fav, avD, avF := c.GetMutationStats()
		deleterious += delet
		neutral += neut
		favorable += fav
		avDelFit += (float64(delet) * avD)
		avFavFit += (float64(fav) * avF)
	}
	if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) }
	if favorable > 0 { avFavFit = avFavFit / float64(favorable) }
	return
}

// Report prints out statistics of this individual. If final==true is prints more details.
func (ind *Individual) Report(_ bool) {
	deleterious, neutral, favorable, avDelFit, avFavFit := ind.GetMutationStats()
	log.Printf("  Ind: fitness: %v, mutations: %d, deleterious: %d, neutral: %d, favorable: %d, avg del: %v, avg fav: %v", ind.GenoFitness, deleterious+neutral+favorable, deleterious, neutral, favorable, avDelFit, avFavFit)
}
