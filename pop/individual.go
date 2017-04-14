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
	//Chromos []*dna.Chromosome 		//todo: not sure if we need this
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
	//for i := range ind.Chromos { ind.Chromos[i] = dna.ChromosomeFactory() }
	for i := range ind.LinkagesFromDad { ind.LinkagesFromDad[i] = dna.LinkageBlockFactory() }
	for i := range ind.LinkagesFromMom { ind.LinkagesFromMom[i] = dna.LinkageBlockFactory() }

	return ind
}


// Mate combines this person with the specified person to create an offspring.
func (ind *Individual) Mate(otherInd *Individual, indivIndex int, uniformRandom *rand.Rand) []*Individual {
	if RecombinationType(config.Cfg.Population.Recombination_model) == CLONAL { utils.NotImplementedYet("Do not support CLONAL recombination yet") }
	actual_offspring := ind.GetNumOffspring(indivIndex, uniformRandom)
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
	//todo: this loop is ignoring chromosomes
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
	numMutations := random.Round(uniformRandom, config.Cfg.Mutations.Mutn_rate) 	//todo: the num mutations should be a poisson distribution with the mean being Mutn_rate
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
	//utils.Verbose(9, "my mutations including new ones: %d, %d, %d", d, n, f)

	offspr.Fitness = offspr.GetFitness() 		// store resulting fitness

	return offspr
}


// GetNumOffspring returns a random number of offspring for this person
func (ind *Individual) GetNumOffspring(indivIndex int, uniformRandom *rand.Rand) int {
	//todo: i do not understand some of this logic, it is from lines 64-73 of mating.f90
	actual_offspring := int(ind.Pop.Num_offspring)		// truncate num offspring to integer
	if ind.Pop.Num_offspring - float64(int(ind.Pop.Num_offspring)) > uniformRandom.Float64() { actual_offspring++ }	// randomly round it up or down
	if indivIndex == 1 { actual_offspring = utils.Max(1, actual_offspring) }
	actual_offspring = utils.Min(int(ind.Pop.Num_offspring) + 1, actual_offspring)
	return actual_offspring
}


func (ind *Individual) GetFitness() (fitness float32) {
	//todo: the current implementation just sums all of the fitness factors
	// Get average of all the LB fitness numbers
	fitness = 0.0
	for i := range ind.LinkagesFromDad {
		fitness += ind.LinkagesFromDad[i].GetFitness() + ind.LinkagesFromMom[i].GetFitness()
	}
	//fitness = fitness / float32(len(ind.LinkagesFromDad) * 2)
	return
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
	log.Printf("  Ind: deleterious %d, neutral: %d, favorable: %d, fitness: %v", deleterious, neutral, favorable, ind.GetFitness())
}
