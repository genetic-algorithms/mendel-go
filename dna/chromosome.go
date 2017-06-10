package dna

import (
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)


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

	/* This is what we used to do in individual.OneOffspring(). Keeping it for reference...
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


// Create the gamete from dad and mom's chromosomes by randomly choosing sections of LBs from either.
func PartialCrossover(dad *Chromosome, mom *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (gamete *Chromosome) {
	gamete = ChromosomeFactory(lBsPerChromosome, false)

	// Algorithm: choose random sizes for <numCrossovers> LB sections for primary and <numCrossovers> LB sections for secondary

	// Choose if dad or mom is the primary chromosome
	var primary, secondary *Chromosome
	if uniformRandom.Intn(2) == 0 {
		primary = dad
		secondary = mom
	} else {
		primary = mom
		secondary = dad
	}

	// Determine number crossovers for this meiosis, between 0 and twice the mean plus 1 (so the average turns out to be Mean_num_crossovers)
	numCrossovers := uniformRandom.Intn(int(2 * config.Cfg.Population.Mean_num_crossovers) + 1)
	//todo: track mean of numCrossovers
	if numCrossovers <= 0 {
		// Handle special case of no crossover - copy all LBs from primary
		config.Verbose(9, " Copying all LBs from primary")
		gamete = primary.Copy(lBsPerChromosome)
		return
	}
	numLbSections := 2 * numCrossovers		// numCrossovers sections for primary, numCrossovers sections for secondary
	primaryMeanSectionSize := utils.RoundIntDiv(float64(lBsPerChromosome)*(1.0-config.Cfg.Population.Crossover_fraction), float64(numCrossovers))	// weight the primary section size to be bigger
	secondaryMeanSectionSize := utils.RoundIntDiv(float64(lBsPerChromosome)*config.Cfg.Population.Crossover_fraction, float64(numCrossovers))
	config.Verbose(9, " Mean_num_crossovers=%v, numCrossovers=%v, numLbSections=%v, primaryMeanSectionSize=%v, secondaryMeanSectionSize=%v\n", config.Cfg.Population.Mean_num_crossovers, numCrossovers, numLbSections, primaryMeanSectionSize, secondaryMeanSectionSize)

	// Copy each LB section
	begIndex := 0		// points to the beginning of the next LB section
	maxIndex := int(lBsPerChromosome) - 1	// 0 based
	// go 2 at a time thru the sections, 1 for primary, 1 for secondary
	for section:=1; section<numLbSections; section+=2 {
		// Copy LB section from primary
		if begIndex > maxIndex { break }
		var sectionLen int
		if primaryMeanSectionSize <= 0 {
			sectionLen = 1
		} else {
			sectionLen = uniformRandom.Intn(2 * primaryMeanSectionSize) + 1		// randomly choose a length for this section that on average will be meanSectionSize. Should never be 0
		}
		endIndex := utils.MinInt(begIndex+sectionLen-1, maxIndex)
		config.Verbose(9, " Copying LBs %v-%v from primary\n", begIndex, endIndex)
		for lb:=begIndex; lb<=endIndex; lb++ { gamete.LinkageBlocks[lb] = primary.LinkageBlocks[lb].Copy() }

		// Copy LB section from secondary
		begIndex = endIndex + 1
		if begIndex > maxIndex { break }
		if secondaryMeanSectionSize <= 0 {
			sectionLen = 0
		} else {
			sectionLen = uniformRandom.Intn(2 * secondaryMeanSectionSize) + 1		// randomly choose a length for this section that on average will be meanSectionSize. Should never be 0
		}
		endIndex = utils.MinInt(begIndex+sectionLen-1, maxIndex)
		if section+1 >=  numLbSections { endIndex = maxIndex }		// make the last section reach to the end of the chromosome
		config.Verbose(9, " Copying LBs %v-%v from secondary\n", begIndex, endIndex)
		for lb:=begIndex; lb<=endIndex; lb++ { gamete.LinkageBlocks[lb] = secondary.LinkageBlocks[lb].Copy() }

		// For next iteration
		begIndex = endIndex + 1
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
