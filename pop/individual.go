package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/dna"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)

// Individual represents 1 organism in the population, tracking its mutations and alleles.
type Individual struct {
	Pop *Population
	Fitness *float64
	Chromos []*dna.Chromosome 		//todo: not sure if we need this
	LinkagesFromDad []*dna.LinkageBlock
	LinkagesFromMom []*dna.LinkageBlock
}

func IndividualFactory(pop *Population) *Individual{
	ind := &Individual{
		Pop: pop,
		Chromos: make([]*dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		LinkagesFromDad: make([]*dna.LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
		LinkagesFromMom: make([]*dna.LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
	}
	// Note: there is no need to allocate the mutation slices with backing arrays. That will happen automatically the 1st time they are
	//		appended to with append(). Altho we will prob eventually want to implement our own append function to do it in bigger chunks.
	//		See https://blog.golang.org/go-slices-usage-and-internals

	//todo: there is probably a faster way to initilize these arrays
	//for i := range ind.Chromos { ind.Chromos[i] = dna.ChromosomeFactory() }
	for i := range ind.LinkagesFromDad { ind.LinkagesFromDad[i] = dna.LinkageBlockFactory() }
	for i := range ind.LinkagesFromMom { ind.LinkagesFromMom[i] = dna.LinkageBlockFactory() }

	return ind
}

// Mate mates this person with the specified person to create an offspring.
func (ind *Individual) Mate(otherInd *Individual, indivIndex int) []*Individual {
	actual_offspring := ind.GetNumOffspring(indivIndex)
	offspr := make([]*Individual, actual_offspring)
	for child:=1; child<=actual_offspring; child++ {
		if config.Cfg.Population.Recombination_model != CLONAL {
			offspr[child-1] = ind.Offspring(otherInd)
		}
	}
	return offspr
}

// Offspring returns 1 offspring of this person and the specified person.
func (ind *Individual) Offspring(otherInd *Individual) *Individual {
	offspr := IndividualFactory(ind.Pop)
	// Set the number of segments.  Three sections of the chromosome that are involved in the crossover.  Form the gametes chromosome by chromosome.
	iseg_max := 3
	if !config.Cfg.Population.Dynamic_linkage {
		iseg_max = 1  // can come from any parent
	}

	// Loop over the total number of chromosomes
	chr_length := config.Cfg.Population.Num_linkage_subunits / config.Cfg.Population.Haploid_chromosome_number
	for chr:=1; chr<=config.Cfg.Population.Haploid_chromosome_number; chr++ {

	}

	return offspr
}

// GetNumOffspring returns a random number of offspring for this person
func (ind *Individual) GetNumOffspring(indivIndex int) int {
	//todo: i do not understand this logic, it is from lines 64-73 of mating.f90
	actual_offspring := int(ind.Pop.Num_offspring)
	if ind.Pop.Num_offspring - int(ind.Pop.Num_offspring) > rand.Float64() { actual_offspring++ }
	if indivIndex == 1 { actual_offspring = utils.Max(1, actual_offspring) }
	actual_offspring = utils.Min(int(ind.Pop.Num_offspring) + 1, actual_offspring)
	return actual_offspring
}