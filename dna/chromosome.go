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

	if initialize {			// first generation
		for i := range c.LinkageBlocks { c.LinkageBlocks[i] = LinkageBlockFactory(c)	}
	}

	return c
}


// Makes a semi-deep copy (everything but the mutations) of a chromosome
func (c *Chromosome) Copy(lBsPerChromosome uint32) (newChr *Chromosome) {
	newChr = ChromosomeFactory(lBsPerChromosome, false)
	for lb := range c.LinkageBlocks {
		//newChr.LinkageBlocks[lb] = c.LinkageBlocks[lb].Copy()
		c.LinkageBlocks[lb].Transfer(c, newChr, lb)
	}
	return
}


// GetNumLinkages returns the number of linkage blocks from each parent (we assume they always have the same number of LBs from each parent)
func (c *Chromosome) GetNumLinkages() uint32 { return uint32(len(c.LinkageBlocks)) }


// Meiosis creates and returns a chromosome as part of reproduction by implementing the crossover model specified in the config file.
// This is 1 form of Copy() for the Chromosome class.
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
	for lb:=0; lb<int(dad.GetNumLinkages()); lb++ {
		if uniformRandom.Intn(2) == 0 {
			//gamete.LinkageBlocks[lb] = dad.LinkageBlocks[lb].Copy()
			dad.LinkageBlocks[lb].Transfer(dad, gamete, lb)
		} else {
			//gamete.LinkageBlocks[lb] = mom.LinkageBlocks[lb].Copy()
			mom.LinkageBlocks[lb].Transfer(mom, gamete, lb)
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

	// Mean_num_crossovers is the average number of crossovers for the chromosome PAIR during Meiosis 1 Metaphase. So for each chromosome (chromotid)
	// the mean = (Mean_num_crossovers / 2). When determining the actual num crossovers for this instance, we get a random number in the
	// range: 0 - (2 * mean + 1) which is (2 * Mean_num_crossovers / 2 + 1) which is (Mean_num_crossovers + 1)
	// To clarify, numCrossovers is the num crossovers in this specific 1 chromosome
	numCrossovers := uniformRandom.Intn(int(config.Cfg.Population.Mean_num_crossovers) + 1)
	//todo: track mean of numCrossovers
	// For numCrossovers=2 the chromosome would normally look like this :  |  S  |         P         |  S  |
	// But to make the section sizes of the secondary and primary more similar we will model it like this :  |  P  |  S  |  P  |  S  |
	// numCrossovers=1 means 2 LB sections, numCrossovers=2 means 3 LB sections, numCrossovers=3 means 5 LB sections
	var numLbSections int
	switch {
	case numCrossovers <= 0:
		// Handle special case of no crossover - copy all LBs from primary
		//config.Verbose(9, " Copying all LBs from primary")
		gamete = primary.Copy(lBsPerChromosome)
		return
	default:
		numLbSections = (2 * numCrossovers)
	}
	//primaryMeanSectionSize := utils.RoundIntDiv(float64(lBsPerChromosome)*(1.0-config.Cfg.Population.Crossover_fraction), float64(numCrossovers))	// weight the primary section size to be bigger
	//secondaryMeanSectionSize := utils.RoundIntDiv(float64(lBsPerChromosome)*config.Cfg.Population.Crossover_fraction, float64(numCrossovers))
	meanSectionSize := utils.RoundIntDiv(float64(lBsPerChromosome), float64(numLbSections))
	//config.Verbose(9, " Mean_num_crossovers=%v, numCrossovers=%v, numLbSections=%v, meanSectionSize=%v\n", config.Cfg.Population.Mean_num_crossovers, numCrossovers, numLbSections, meanSectionSize)

	// Copy each LB section.
	begIndex := 0		// points to the beginning of the next LB section
	maxIndex := int(lBsPerChromosome) - 1	// 0 based
	parent := primary		// we will alternate between secondary and primary
	// go 2 at a time thru the sections, 1 for primary, 1 for secondary
	for section:=1; section<=numLbSections; section++ {
		// Copy LB section
		if begIndex > maxIndex { break }
		var sectionLen int
		if meanSectionSize <= 0 {
			sectionLen = 1		// because we can not pass 0 into Intn()
		} else {
			sectionLen = uniformRandom.Intn(2 * meanSectionSize) + 1		// randomly choose a length for this section that on average will be meanSectionSize. Should never be 0
		}
		endIndex := utils.MinInt(begIndex+sectionLen-1, maxIndex)
		if section >=  numLbSections { endIndex = maxIndex }		// make the last section reach to the end of the chromosome
		//config.Verbose(9, " Copying LBs %v-%v from %v\n", begIndex, endIndex, parent==primary)
		for lb:=begIndex; lb<=endIndex; lb++ {
			//gamete.LinkageBlocks[lb] = parent.LinkageBlocks[lb].Copy()
			parent.LinkageBlocks[lb].Transfer(parent, gamete, lb)
		}

		/*
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
		*/

		// For next iteration
		begIndex = endIndex + 1
		if parent == primary {
			parent = secondary
		} else {
			parent = primary
		}
	}
	return
}


// AppendMutation creates and adds a mutations to the LB specified
func (c *Chromosome) AppendMutation(lbInChr int, uniformRandom *rand.Rand) {
	// Note: to try to save time, we could accumulate the chromosome fitness as we go, but doing so would bypass the LB method
	//		of calculating its own fitness, so we won't do that.
	c.LinkageBlocks[lbInChr].AppendMutation(uniformRandom)
}

// SumFitness combines the fitness effect of all of its LBs in the additive method
func (c *Chromosome) SumFitness() (fitness float64) {
	for _, lb := range c.LinkageBlocks {
		fitness += lb.SumFitness()
	}
	// Note: we don't bother caching the fitness in the chromosome, because we cache the total in the individual, and we know better when to cache there.
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
	// Note: we don't bother caching the fitness stats in the chromosome, because we cache the total in the individual, and we know better when to cache there.
	return
}


// GetInitialAlleleStats returns the number of deleterious, neutral, favorable initial alleles, and the average fitness factor of each
func (c *Chromosome) GetInitialAlleleStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Calc the average of each type of allele: multiply the average from each LB and num alleles from each LB, then at the end divide by total num alleles
	for _,lb := range c.LinkageBlocks {
		delet, neut, fav, avD, avF := lb.GetInitialAlleleStats()
		deleterious += delet
		neutral += neut
		favorable += fav
		avDelFit += (float64(delet) * avD)
		avFavFit += (float64(fav) * avF)
	}
	if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) }
	if favorable > 0 { avFavFit = avFavFit / float64(favorable) }
	// Note: we don't bother caching the fitness stats in the chromosome, because we cache the total in the individual, and we know better when to cache there.
	return
}


/*
// GatherAlleles counts all of this chromosome's alleles (both mutations and initial alleles) and adds them to the given struct
func (c *Chromosome) GatherAlleles(alleles *Alleles) {
	for _, lb := range c.LinkageBlocks { lb.GatherAlleles(alleles) }
}
*/


// CountAlleles adds all of this chromosome's alleles (both mutations and initial alleles) to the given struct
func (c *Chromosome) CountAlleles(allelesForThisIndiv *AlleleCount) {
	for _, lb := range c.LinkageBlocks { lb.CountAlleles(allelesForThisIndiv) }
}
