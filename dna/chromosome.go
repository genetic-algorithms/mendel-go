package dna

import "math/rand"


// Chromosome represents 1 chromosome in an individual's genome.
type Chromosome struct {
	LinkageBlocks []*LinkageBlock
}

// This only creates an empty chromosome for gen 0 as part of Meiosis(). Meiosis() creates a populated chromosome.
func ChromosomeFactory(lBsPerChromosome uint32, initialize bool) *Chromosome {
	c := &Chromosome{
		LinkageBlocks: make([]*LinkageBlock, lBsPerChromosome),
	}

	if initialize {
		for i := range c.LinkageBlocks { c.LinkageBlocks[i] = LinkageBlockFactory()	}
	}

	return c
}


// Makes a semi-deep copy (everything but the mutations) of a chromosome
func (c *Chromosome) Copy(lBsPerChromosome uint32) (newChr *Chromosome) {
	newChr = ChromosomeFactory(lBsPerChromosome, false)
	for lb := range c.LinkageBlocks {
		newChr.LinkageBlocks[lb] = c.LinkageBlocks[lb].Copy()
	}
	return
}


// GetNumLinkages returns the number of linkage blocks from each parent (we assume they always have the same number of LBs from each parent)
func (c *Chromosome) GetNumLinkages() uint32 { return uint32(len(c.LinkageBlocks)) }


// Meiosis creates and returns a chromosome as part of reproduction by implementing the crossover model specified in the config file.
// This is the Chromosome class version of Copy().
func (dad *Chromosome) Meiosis(mom *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (gamete *Chromosome) {
	gamete = Mdl.Crossover(dad, mom, lBsPerChromosome, uniformRandom)

	/* This is what we used to do from individual.OneOffspring(). Keeping it for reference...
	for lb:=uint32(0); lb< dad.GetNumLinkages(); lb++ {
		// randomly choose which grandparents to get the LBs from
		if uniformRandom.Intn(2) == 0 {
			offspr.LinkagesFromDad[lb] = dad.LinkagesFromDad[lb].Copy()
		} else {
			offspr.LinkagesFromDad[lb] = dad.LinkagesFromMom[lb].Copy()
		}

		if uniformRandom.Intn(2) == 0 {
			offspr.LinkagesFromMom[lb] = mom.LinkagesFromDad[lb].Copy()
		} else {
			offspr.LinkagesFromMom[lb] = mom.LinkagesFromMom[lb].Copy()
		}
	}
	*/

	return
}


// The different implementations of LB crossover to another chromosome during meiosis
type CrossoverType func(dad *Chromosome, mom *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) *Chromosome

// Create the gamete from all of dad's chromosomes or all of mom's chromosomes.
func NoCrossover(dad *Chromosome, mom *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (gamete *Chromosome) {
	//gamete = ChromosomeFactory(lBsPerChromosome, false)
	// Copy all of the LBs from the one or the other
	if uniformRandom.Intn(2) == 0 {
		gamete = dad.Copy(lBsPerChromosome)
		//for lb := uint32(0); lb < dad.GetNumLinkages(); lb++ {
		//	gamete.LinkageBlocks[lb] = dad.LinkageBlocks[lb].Copy()
		//}
	} else {
		gamete = mom.Copy(lBsPerChromosome)
		//for lb := uint32(0); lb < mom.GetNumLinkages(); lb++ {
		//	gamete.LinkageBlocks[lb] = mom.LinkageBlocks[lb].Copy()
		//}
	}
	return
}


// Create the gamete from dad and mom's chromosomes by randomly choosing each LB from either.
func FullCrossover(dad *Chromosome, mom *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (gamete *Chromosome) {
	gamete = ChromosomeFactory(lBsPerChromosome, false)
	// Each LB can come from either dad or mom
	for lb:=uint32(0); lb< dad.GetNumLinkages(); lb++ {
		if uniformRandom.Intn(2) == 0 {
			gamete.LinkageBlocks[lb] = dad.LinkageBlocks[lb].Copy()
		} else {
			gamete.LinkageBlocks[lb] = mom.LinkageBlocks[lb].Copy()
		}
	}
	return
}


// AppendMutation creates and adds a mutations to the LB specified
func (c *Chromosome) AppendMutation(lbInChr int, uniformRandom *rand.Rand) {
	c.LinkageBlocks[lbInChr].AppendMutation(uniformRandom)
}

// SumFitness combines the fitness effect of all of its LBs in the additive method
func (c *Chromosome) SumFitness() (fitness float64) {
	for _, lb := range c.LinkageBlocks {
		fitness += lb.SumFitness()
	}
	return
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (c *Chromosome) GetMutationStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Calc the average of each type of mutation: multiply the average from each LB and num mutns from each LB, then at the end divide by total num mutns
	for _,lb := range c.LinkageBlocks {
		delet, neut, fav, avD, avF := lb.GetMutationStats()
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
