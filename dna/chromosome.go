package dna

import (
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)


// Chromosome represents 1 chromosome in an individual's genome.
type Chromosome struct {
	//LinkageBlocks []*LinkageBlock
	LinkageBlocks []LinkageBlock
	NumMutations uint32		// keep a running total of the mutations. This is both mutations and initial alleles.
	FitnessEffect float32	// keep a running total of the fitness contribution of this chromosome
}

// This only creates an empty chromosome for gen 0 as part of Meiosis(). Meiosis() creates a populated chromosome.
func ChromosomeFactory(lBsPerChromosome uint32, _ bool) *Chromosome {
	c := &Chromosome{
		//LinkageBlocks: make([]*LinkageBlock, lBsPerChromosome),
		LinkageBlocks: make([]LinkageBlock, lBsPerChromosome),
	}

	//if genesis {			// first generation
		//for i := range c.LinkageBlocks { c.LinkageBlocks[i] = LinkageBlockFactory(c)	}
		//for i := range c.LinkageBlocks { c.LinkageBlocks[i] = LinkageBlock{} } 	// <- do not need to do this because the make() above creates zero-elements
	//}

	return c
}


// Reinitialize gets an existing/old chromosome ready for reuse. In addition to their member vars, Chromosome objects have an array of LBs. We will end up
// overwriting the contents of those LB objects (including their Mutation array) in TransferLB(), but we want to reuse the memory allocation of those arrays.
// In the other Chromosome methods we can tell if the recycled chromosome exists because the ptr to it will be non-nil.
func (c *Chromosome) Reinitialize() *Chromosome {
	c.NumMutations = 0
	c.FitnessEffect = 0.0
	return c
}


/*
// Makes a semi-deep copy (everything but the mutations) of a chromosome. "Copy" means actually copy, or create a new chromosome with new LBs that point back to the LB history chain.
func (c *Chromosome) CopyOld(lBsPerChromosome uint32) (newChr *Chromosome) {
	newChr = ChromosomeFactory(lBsPerChromosome, false)
	//newChr.NumMutations = c.NumMutations   // <- Transfer() does this
	for lbIndex := range c.LinkageBlocks {
		//c.LinkageBlocks[lbIndex].Transfer(c, newChr, lbIndex)
		c.TransferLB(newChr, lbIndex)
	}
	return
}
*/


// Copy makes a deep copy of this chromosome to offspr
func (c *Chromosome) Copy(/*lBsPerChromosome uint32,*/ offspr *Chromosome) /*(newChr *Chromosome)*/ {
	//newChr.NumMutations = c.NumMutations   // <- TransferLB() takes care of this
	for lbIndex := range c.LinkageBlocks {
		c.TransferLB(offspr, lbIndex)
	}
}


// TransferLB copies a LB from this chromosome to newChr. The LB in newChr may be recycled from a previous gen, we will completely overwrite it.
// As a side effect, we also update the newChr's mutation and fitness stats
func (c *Chromosome) TransferLB(newChr *Chromosome, lbIndex int) {
	// Assign the parent-LB to this child-LB to copy all of the member vars, but we don't want to copy the mutn slice because that will point to the same backing array,
	// and the child-LB needs an independent backing array (because this parent-LB will likely be copies to other child-LBs)
	tmpMutn := newChr.LinkageBlocks[lbIndex].mutn 		// save the slice header
	newChr.LinkageBlocks[lbIndex] = c.LinkageBlocks[lbIndex] 	// this copies all of the LB struct fields (but not the mutn array that backs the slice)
	newChr.LinkageBlocks[lbIndex].mutn = tmpMutn[:0]		// put back the slice header (and its backing array) but set it to zero length

	if cap(newChr.LinkageBlocks[lbIndex].mutn) < len(c.LinkageBlocks[lbIndex].mutn) {
		// Increase the child-LB capacity so it can accept the mutations from the parent-LB
		newChr.LinkageBlocks[lbIndex].mutn = make([]Mutation, len(c.LinkageBlocks[lbIndex].mutn)) 	//todo: consider making the new LB with a capacity a little bigger (mutation_rate / num_LBs)
	} else {
		// The capacity was big enough, get the child-LB len the same as the parent-LB so the copy copies what we want
		newChr.LinkageBlocks[lbIndex].mutn = newChr.LinkageBlocks[lbIndex].mutn[:len(c.LinkageBlocks[lbIndex].mutn)]
	}
	copy(newChr.LinkageBlocks[lbIndex].mutn, c.LinkageBlocks[lbIndex].mutn) 	// now copy the mutations

	// Housekeeping for the new chromo
	newChr.NumMutations += newChr.LinkageBlocks[lbIndex].GetNumMutations()
	newChr.FitnessEffect += newChr.LinkageBlocks[lbIndex].SumFitness()
}


// GetNumLinkages returns the number of linkage blocks from each parent (we assume they always have the same number of LBs from each parent)
func (c *Chromosome) GetNumLinkages() uint32 { return uint32(len(c.LinkageBlocks)) }


// Meiosis fills in a child chromosome as part of reproduction by implementing the crossover model specified in the config file.
// This is 1 form of Copy for the Chromosome class.
func (dad *Chromosome) Meiosis(mom *Chromosome, offspr *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) /*(gamete *Chromosome)*/ {
	offspr.Reinitialize() 	// In case it is a recycled chromosome

	/*gamete =*/ Mdl.Crossover(dad, mom, offspr, lBsPerChromosome, uniformRandom)

	return
}


// The different implementations of LB crossover to another chromosome during meiosis
type CrossoverType func(dad *Chromosome, mom *Chromosome, offspr *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) //*Chromosome

// Create the gamete from all of dad's chromosomes or all of mom's chromosomes.
func NoCrossover(dad *Chromosome, mom *Chromosome, offspr *Chromosome, _ uint32, uniformRandom *rand.Rand) /*(gamete *Chromosome)*/ {
	//gamete = ChromosomeFactory(lBsPerChromosome, false)
	// Create the chromosome (if necessary) and copy all of the LBs from the one or the other
	if uniformRandom.Intn(2) == 0 {
		/*gamete =*/ dad.Copy(/*lBsPerChromosome,*/ offspr)
	} else {
		/*gamete =*/ mom.Copy(/*lBsPerChromosome,*/ offspr)
	}
	return
}


// Create the gamete from dad and mom's chromosomes by randomly choosing each LB from either.
func FullCrossover(dad *Chromosome, mom *Chromosome, offspr *Chromosome, _ uint32, uniformRandom *rand.Rand) /*(gamete *Chromosome)*/ {
	//gamete = ChromosomeFactory(lBsPerChromosome, false)
	// Each LB can come from either dad or mom
	for lbIndex :=0; lbIndex <int(dad.GetNumLinkages()); lbIndex++ {
		if uniformRandom.Intn(2) == 0 {
			//if config.Cfg.Computation.Transfer_linkage_blocks {
			///*lb :=*/ dad.LinkageBlocks[lbIndex].Transfer(dad, gamete, lbIndex)
			//gamete.NumMutations += lb.GetNumMutations()   // <- Transfer() does this
			dad.TransferLB(offspr, lbIndex)
			//} else {
			//	lb := LinkageBlockFactory(gamete, dad.LinkageBlocks[lbIndex])
			//	gamete.LinkageBlocks[lbIndex] = lb
			//	gamete.NumMutations += lb.GetNumMutations()
			//}
		} else {
			//if config.Cfg.Computation.Transfer_linkage_blocks {
			///*lb :=*/ mom.LinkageBlocks[lbIndex].Transfer(mom, gamete, lbIndex)
			//gamete.NumMutations += lb.GetNumMutations()   // <- Transfer() does this
			mom.TransferLB(offspr, lbIndex)
			//} else {
			//	lb := LinkageBlockFactory(gamete, mom.LinkageBlocks[lbIndex])
			//	gamete.LinkageBlocks[lbIndex] = lb
			//	gamete.NumMutations += lb.GetNumMutations()
			//}
		}
	}
	return
}


// Create the gamete from dad and mom's chromosomes by randomly choosing sections of LBs from either.
func PartialCrossover(dad *Chromosome, mom *Chromosome, offspr *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) /*(gamete *Chromosome)*/ {
	//gamete = ChromosomeFactory(lBsPerChromosome, false)

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
		/*gamete =*/ primary.Copy(/*lBsPerChromosome,*/ offspr)
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
		for lbIndex :=begIndex; lbIndex <=endIndex; lbIndex++ {
			//if config.Cfg.Computation.Transfer_linkage_blocks {
			///*lb :=*/ parent.LinkageBlocks[lbIndex].Transfer(parent, gamete, lbIndex)
			//gamete.NumMutations += lb.GetNumMutations()   // <- Transfer() does this
			parent.TransferLB(offspr, lbIndex)
			//} else {
			//	lb := LinkageBlockFactory(gamete, parent.LinkageBlocks[lbIndex])
			//	gamete.LinkageBlocks[lbIndex] = lb
			//	gamete.NumMutations += lb.GetNumMutations()
			//}
		}

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
func (c *Chromosome) AppendMutation(lbInChr int, mutId uint64, uniformRandom *rand.Rand) {
	// Note: to try to save time, we could accumulate the chromosome fitness as we go, but doing so would bypass the LB method
	//		of calculating its own fitness, so we won't do that.
	fitnessEffect := c.LinkageBlocks[lbInChr].AppendMutation(mutId, uniformRandom)
	c.NumMutations++
	c.FitnessEffect += fitnessEffect
}


// ChrAppendInitialContrastingAlleles adds an initial contrasting allele pair to 2 LBs on 2 chromosomes (favorable to 1, deleterious to the other).
func ChrAppendInitialContrastingAlleles(chr1, chr2 *Chromosome, lbIndex int, uniqueInt *utils.UniqueInt, uniformRandom *rand.Rand) {
	fitnessEffect1, fitnessEffect2 := AppendInitialContrastingAlleles(&chr1.LinkageBlocks[lbIndex], &chr2.LinkageBlocks[lbIndex], uniqueInt, uniformRandom)
	chr1.NumMutations++
	chr1.FitnessEffect += fitnessEffect1
	chr2.NumMutations++
	chr2.FitnessEffect += fitnessEffect2
}

// SumFitness combines the fitness effect of all of its LBs in the additive method
func (c *Chromosome) SumFitness() float64 {
	return float64(c.FitnessEffect)
	//var fitness float32
	//for _, lb := range c.LinkageBlocks {
	//	fitness += lb.SumFitness()
	//}
	//return float64(fitness)
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (c *Chromosome) GetMutationStats() (deleterious, neutral, favorable uint32 /*, avDelFit, avFavFit float64*/) {
	// Calc the average of each type of mutation: multiply the average from each LB and num mutns from each LB, then at the end divide by total num mutns
	for _,lb := range c.LinkageBlocks {
		delet, neut, fav /*, avD, avF*/ := lb.GetMutationStats()
		deleterious += delet
		neutral += neut
		favorable += fav
		//avDelFit += (float64(delet) * avD)
		//avFavFit += (float64(fav) * avF)
	}
	//if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) }
	//if favorable > 0 { avFavFit = avFavFit / float64(favorable) }
	// Note: we don't bother caching the fitness stats in the chromosome, because we cache the total in the individual, and we know better when to cache there.
	return
}


// GetInitialAlleleStats returns the number of deleterious, neutral, favorable initial alleles, and the average fitness factor of each
func (c *Chromosome) GetInitialAlleleStats() (deleterious, neutral, favorable uint32 /*, avDelFit, avFavFit float64*/) {
	// Calc the average of each type of allele: multiply the average from each LB and num alleles from each LB, then at the end divide by total num alleles
	for _,lb := range c.LinkageBlocks {
		delet, neut, fav /*, avD, avF*/ := lb.GetInitialAlleleStats()
		deleterious += delet
		neutral += neut
		favorable += fav
		//avDelFit += (float64(delet) * avD)
		//avFavFit += (float64(fav) * avF)
	}
	//if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) }
	//if favorable > 0 { avFavFit = avFavFit / float64(favorable) }
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
