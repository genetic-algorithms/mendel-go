package dna

import (
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/config"
	"github.com/genetic-algorithms/mendel-go/utils"
)


// Chromosome represents 1 chromosome in an individual's genome.
type Chromosome struct {
	LinkageBlocks []LinkageBlock
	FitnessEffect float32	// keep a running total of the fitness contribution of this LB to the chromosome
}


// Since the Individual's slice of chromosomes isn't ptrs, but the actual objects, we have
// this factory work on it directly (instead of creating an object and returning a ptr to it).
func (c *Chromosome) ChromosomeFactory(lBsPerChromosome uint32) {
	c.LinkageBlocks = make([]LinkageBlock, lBsPerChromosome)
}


// Not currently used, but kept here in case we want to reuse populations - Reinitialize gets an existing/old chromosome ready for reuse. In addition to their member vars, Chromosome objects have an array of LBs. We will end up
// overwriting the contents of those LB objects (including their Mutation array) in TransferLB(), but we want to reuse the memory allocation of those arrays.
// In the other Chromosome methods we can tell if the recycled chromosome exists because the ptr to it will be non-nil.
func (c *Chromosome) Reinitialize() {
	c.FitnessEffect = 0.0
}


// Copy makes a deep copy of this chromosome to offspr
func (c *Chromosome) Copy(offspr *Chromosome) (deleterious, neutral, favorable, delAllele, favAllele uint32) {
	//newChr.NumMutations = c.NumMutations   // <- TransferLB() takes care of this
	for lbIndex := range c.LinkageBlocks {
		delet, neut, fav, delAll, favAll := c.TransferLB(offspr, lbIndex)
		deleterious += delet
		neutral += neut
		favorable += fav
		delAllele += delAll
		favAllele += favAll
	}
	return
}


// TransferLB copies a LB from this chromosome to newChr. The LB in newChr may be recycled from a previous gen, we will completely overwrite it.
// As a side effect, we also update the newChr's fitness stats. Returns the numbers of each kind of mutation.
func (c *Chromosome) TransferLB(newChr *Chromosome, lbIndex int) (uint32, uint32, uint32, uint32, uint32) {
	newChr.LinkageBlocks[lbIndex] = c.LinkageBlocks[lbIndex]    // this copies all of the LB struct fields, including the slice reference (but not the mutn array that backs the slice)
	newChr.LinkageBlocks[lbIndex].IsPtrToParent = true            // indicate we are still using the parents mutn array, so we will copy it later if we have to add a mutation

	// Housekeeping for the new chromo
	newChr.FitnessEffect += newChr.LinkageBlocks[lbIndex].SumFitness()
	return newChr.LinkageBlocks[lbIndex].GetMutationStats()
}


// GetNumLinkages returns the number of linkage blocks from each parent (we assume they always have the same number of LBs from each parent)
func (c *Chromosome) GetNumLinkages() uint32 { return uint32(len(c.LinkageBlocks)) }


/* Not used right now because it simply calls the crossover model function, but may bring it back if there is more to do...
// Meiosis fills in a child chromosome as part of reproduction by implementing the crossover model specified in the config file.
// This is 1 form of Copy for the Chromosome class.
func (dad *Chromosome) Meiosis(mom *Chromosome, offspr *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (uint32, uint32, uint32, uint32, uint32) {
	//offspr.Reinitialize() 	// In case it is a recycled chromosome
	return Mdl.Crossover(dad, mom, offspr, lBsPerChromosome, uniformRandom)
}
*/


// The different implementations of LB crossover to another chromosome during meiosis
type CrossoverType func(dad *Chromosome, mom *Chromosome, offspr *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (uint32, uint32, uint32, uint32, uint32)

// Create the gamete from all of dad's chromosomes or all of mom's chromosomes. Returns the number of each kind of mutation in the new chromosome.
func NoCrossover(dad *Chromosome, mom *Chromosome, offspr *Chromosome, _ uint32, uniformRandom *rand.Rand) (uint32, uint32, uint32, uint32, uint32) {
	// Create the chromosome (if necessary) and copy all of the LBs from the one or the other
	if uniformRandom.Intn(2) == 0 {
		return dad.Copy(offspr)
	} else {
		return mom.Copy(offspr)
	}
}


// Create the gamete from dad and mom's chromosomes by randomly choosing each LB from either. Returns the number of each kind of mutation in the new chromosome.
func FullCrossover(dad *Chromosome, mom *Chromosome, offspr *Chromosome, _ uint32, uniformRandom *rand.Rand) (deleterious, neutral, favorable, delAllele, favAllele uint32) {
	// Each LB can come from either dad or mom
	for lbIndex :=0; lbIndex <int(dad.GetNumLinkages()); lbIndex++ {
		var delet, neut, fav, delAll, favAll uint32
		if uniformRandom.Intn(2) == 0 {
			delet, neut, fav, delAll, favAll = dad.TransferLB(offspr, lbIndex)
		} else {
			delet, neut, fav, delAll, favAll = mom.TransferLB(offspr, lbIndex)
		}
		deleterious += delet
		neutral += neut
		favorable += fav
		delAllele += delAll
		favAllele += favAll
	}
	return
}


// Create the gamete from dad and mom's chromosomes by randomly choosing sections of LBs from either. Returns the number of each kind of mutation in the new chromosome.
func PartialCrossover(dad *Chromosome, mom *Chromosome, offspr *Chromosome, lBsPerChromosome uint32, uniformRandom *rand.Rand) (deleterious, neutral, favorable, delAllele, favAllele uint32) {
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
		deleterious, neutral, favorable, delAllele, favAllele = primary.Copy(offspr)
		return
	default:
		numLbSections = 2 * numCrossovers
	}
	meanSectionSize := utils.RoundIntDiv(float64(lBsPerChromosome), float64(numLbSections))

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
		for lbIndex :=begIndex; lbIndex <=endIndex; lbIndex++ {
			delet, neut, fav, delAll, favAll := parent.TransferLB(offspr, lbIndex)
			deleterious += delet
			neutral += neut
			favorable += fav
			delAllele += delAll
			favAllele += favAll
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


// AppendMutation creates and adds a mutations to the LB specified. Returns the type of mutation added.
func (c *Chromosome) AppendMutation(lbInChr int, mutId uint64, uniformRandom *rand.Rand) MutationType {
	// Note: to try to save time, we could accumulate the chromosome fitness as we go, but doing so would bypass the LB method
	//		of calculating its own fitness, so we won't do that.
	mType, fitnessEffect := c.LinkageBlocks[lbInChr].AppendMutation(mutId, uniformRandom)
	c.FitnessEffect += fitnessEffect
	return mType
}


// ChrAppendInitialContrastingAlleles adds an initial contrasting allele pair to 2 LBs on 2 chromosomes (favorable to 1, deleterious to the other).
func ChrAppendInitialContrastingAlleles(chr1, chr2 *Chromosome, lbIndex int, uniqueInt *utils.UniqueInt, uniformRandom *rand.Rand) {
	fitnessEffect1, fitnessEffect2 := AppendInitialContrastingAlleles(&chr1.LinkageBlocks[lbIndex], &chr2.LinkageBlocks[lbIndex], uniqueInt, uniformRandom)
	chr1.FitnessEffect += fitnessEffect1
	chr2.FitnessEffect += fitnessEffect2
}

// ChrAppendInitialAllelePair adds an initial contrasting allele pair to 2 LBs on 2 chromosomes (favorable to 1, deleterious to the other).
func ChrAppendInitialAllelePair(chr1, chr2 *Chromosome, lbIndex int, favMutn, delMutn Mutation) {
	AppendInitialAllelePair(&chr1.LinkageBlocks[lbIndex], &chr2.LinkageBlocks[lbIndex], favMutn, delMutn)
	chr1.FitnessEffect += favMutn.FitnessEffect
	chr2.FitnessEffect += delMutn.FitnessEffect
}

// SumFitness combines the fitness effect of all of its LBs in the additive method
func (c *Chromosome) SumFitness() float64 {
	// Now we keep a running total instead
	return float64(c.FitnessEffect)
}


// CountAlleles adds all of this chromosome's alleles (both mutations and initial alleles) to the given struct
func (c *Chromosome) CountAlleles(allelesForThisIndiv *AlleleCount) {
	for _, lb := range c.LinkageBlocks { lb.CountAlleles(allelesForThisIndiv) }
}
