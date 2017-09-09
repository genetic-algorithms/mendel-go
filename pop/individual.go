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
	popPart *PopulationPart
	GenoFitness     float64		// fitness due to genomic mutations
	PhenoFitness     float64		// fitness due to GenoFitness plus environmental noise and selection noise
	Dead            bool 		// if true, selection has identified it for elimination
	NumMutations uint32		// keep a running total of the mutations. This is both mutations and initial alleles.

	// Note: we currently don't really need to cache these, because p.GetMutationStats caches its values, and the only other function that currently uses these is ind.Report() which only gets called for small populations.
	//		But it would only save 0.56 MB for 10,000 population, so let's wait and see if we need them cached for more stats in the future.
	NumDeleterious, NumNeutral, NumFavorable uint32		// cache some of the stats we usually gather
	//MeanDelFit, MeanFavFit float64
	NumDelAllele, NumNeutAllele, NumFavAllele uint32		// cache some of the stats we usually gather about initial alleles
	//MeanDelAlleleFit, MeanFavAlleleFit float64

	//ChromosomesFromDad []*dna.Chromosome
	//ChromosomesFromMom []*dna.Chromosome
	ChromosomesFromDad []dna.Chromosome
	ChromosomesFromMom []dna.Chromosome
}


func IndividualFactory(popPart *PopulationPart, _ bool) *Individual {
	ind := &Individual{
		popPart: popPart,
		//ChromosomesFromDad: make([]*dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		//ChromosomesFromMom: make([]*dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		ChromosomesFromDad: make([]dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		ChromosomesFromMom: make([]dna.Chromosome, config.Cfg.Population.Haploid_chromosome_number),
	}

	//if genesis {   // <- always initialize with empty chromosomes and linkage blocks (with no mutations)
	//for c := range ind.ChromosomesFromDad { ind.ChromosomesFromDad[c] = dna.ChromosomeFactory(popPart.Pop.LBsPerChromosome, genesis) }
	//for c := range ind.ChromosomesFromMom { ind.ChromosomesFromMom[c] = dna.ChromosomeFactory(popPart.Pop.LBsPerChromosome, genesis) }
	for c := range ind.ChromosomesFromDad { ind.ChromosomesFromDad[c].ChromosomeFactory(popPart.Pop.LBsPerChromosome) }
	for c := range ind.ChromosomesFromMom { ind.ChromosomesFromMom[c].ChromosomeFactory(popPart.Pop.LBsPerChromosome) }
	//}

	return ind
}


// Reinitialize gets an existing/old individual ready for reuse. In addition to their member vars, Individual objects have an array of Chromosomes ptrs. We will end up
// overwriting the contents of those Chromosome objects (including their LB array), but we want to reuse the memory allocation of those arrays.
func (ind *Individual) Reinitialize() *Individual {
	// Reset the stats
	ind.GenoFitness = 0.0
	ind.PhenoFitness = 0.0
	ind.Dead = false
	ind.NumMutations = 0
	ind.NumDeleterious = 0
	ind.NumNeutral = 0
	ind.NumFavorable = 0
	ind.NumDelAllele = 0
	ind.NumNeutAllele = 0
	ind.NumFavAllele = 0

	/* Chromosome.Meiosis() reinitializes the chromosomes...
	// ChromosomesFromDad and ChromosomesFromMom have an array of ptrs to existing Chromosomes and we want reuse that memory.
	for _, c := range ind.ChromosomesFromDad { c.Reinitialize() }
	for _, c := range ind.ChromosomesFromMom { c.Reinitialize() }
	// Note that the rest of the contents of the chromosomes and LBs will get reinitialized or overwritten during meiosis
	*/

	return ind
}


/* Not currently used - FreeChromosomes nils out ptrs to memory that this individual owns, so that GC can reclaim it.
func (ind *Individual) FreeChromosomes() {
	// The only ptrs we have are to our chromosomes, which should also free the LBs they own
	ind.ChromosomesFromDad = []*dna.Chromosome{} 	// replacing the slice should free the previous backing array which will free all the chromosome ptrs in 1 operation
	ind.ChromosomesFromMom = []*dna.Chromosome{}

	//for i := range ind.ChromosomesFromDad { ind.ChromosomesFromDad[i] = nil }
	//for i := range ind.ChromosomesFromMom { ind.ChromosomesFromMom[i] = nil }
}
*/


// GetNumChromosomes returns the number of chromosomes from each parent (we assume they always have the same number from each parent)
func (ind *Individual) GetNumChromosomes() uint32 { return uint32(len(ind.ChromosomesFromDad)) }


// Mate combines this person with the specified person to create a list of offspring.
// The offspring are added to newPopPart
func (ind *Individual) Mate(otherInd *Individual, newPopPart *PopulationPart, uniformRandom *rand.Rand) /*[]*Individual*/ {
	if RecombinationType(config.Cfg.Population.Recombination_model) != FULL_SEXUAL { utils.NotImplementedYet("Recombination models other than FULL_SEXUAL are not yet supported") }

	// Mate ind and otherInd to create offspring
	actual_offspring := Mdl.CalcNumOffspring(ind, uniformRandom)
	offspr := make([]*Individual, actual_offspring) 	// temporary slice of the children created
	for child:=uint32(0); child<actual_offspring; child++ {
		offspr[child] = ind.OneOffspring(otherInd, newPopPart, uniformRandom)
	}

	// Add mutations to each offspring. Note: this is done after mating is completed for these parents, because as an optimization some of the LBs are
	// just transferred to the child (not copied). If we added mutations during mating, some children could get the same added mutations.
	for _, child := range offspr {
		child.AddMutations(ind.popPart.Pop.LBsPerChromosome, uniformRandom)
	}

	return /*offspr*/
}


// Offspring returns 1 offspring of this person (dad) and the specified person (mom).
func (dad *Individual) OneOffspring(mom *Individual, newPopPart *PopulationPart, uniformRandom *rand.Rand) *Individual {
	//offspr := IndividualFactory(newPopPart, false)
	offspr := PopPool.GetIndividual(newPopPart)	// this gives us an indiv ready to use, with chromosomes and LBs, and ensures it is on the pop part list
	lBsPerChromosome := dad.popPart.Pop.LBsPerChromosome 		// doesn't matter which parent we get this from

	// Loop thru each chromosome and inherit linkage blocks
	for c:=uint32(0); c<dad.GetNumChromosomes(); c++ {
		// Meiosis() implements the crossover model specified in the config file
		// For your chromosome coming from your dad, combine LBs from his dad and mom
		offsprChr := &offspr.ChromosomesFromDad[c]
		/*chr :=*/ dad.ChromosomesFromDad[c].Meiosis(&dad.ChromosomesFromMom[c], offsprChr, lBsPerChromosome, uniformRandom)
		//offspr.ChromosomesFromDad[c] = chr
		offspr.NumMutations += offsprChr.NumMutations

		// For your chromosome coming from your mom, combine LBs from her dad and mom
		offsprChr = &offspr.ChromosomesFromMom[c]
		/*chr =*/ mom.ChromosomesFromDad[c].Meiosis(&mom.ChromosomesFromMom[c], offsprChr, lBsPerChromosome, uniformRandom)
		//offspr.ChromosomesFromMom[c] = chr
		offspr.NumMutations += offsprChr.NumMutations
	}

	return offspr
}


// AddMutations adds new mutations to this child right after mating.
func (child *Individual) AddMutations(lBsPerChromosome uint32, uniformRandom *rand.Rand) {
	// Apply new mutations
	numMutations := Mdl.CalcNumMutations(uniformRandom)
	popPart := child.popPart
	for m:=uint32(1); m<=numMutations; m++ {
		// Note: we are choosing the LB this way to keep the random number generation the same as when we didn't have chromosomes.
		//		Can change this in the future if you want.
		lb := uniformRandom.Intn(int(config.Cfg.Population.Num_linkage_subunits))	// choose a random LB within the individual
		chr := lb / int(lBsPerChromosome) 		// get the chromosome index
		lbInChr := lb % int(lBsPerChromosome)	// get index of LB within the chromosome
		//lb := uniformRandom.Intn(int(dad.GetNumLinkages()))	// choose a random LB index

		// Randomly choose the LB from dad or mom to put the mutation in.
		// Note: AppendMutation() creates a mutation with deleterious/neutral/favorable, dominant/recessive, etc. based on the relevant input parameter rates
		//if popPart.NextMutId > popPart.LastMutId { log.Printf("Warning: population part exceeded LastMutId %d with NextMutId %d", popPart.LastMutId, popPart.NextMutId)}
		if uniformRandom.Intn(2) == 0 {
			child.ChromosomesFromDad[chr].AppendMutation(lbInChr, popPart.MyUniqueInt.NextInt(), uniformRandom)
		} else {
			child.ChromosomesFromMom[chr].AppendMutation(lbInChr, popPart.MyUniqueInt.NextInt(), uniformRandom)
		}
		//popPart.NextMutId++
	}
	child.NumMutations += numMutations
	//d, n, f := offspr.GetNumMutations()

	child.GenoFitness = Mdl.CalcIndivFitness(child) 		// store resulting fitness
	if child.GenoFitness <= 0.0 { child.Dead = true }

	return
}


// AddInitialContrastingAlleles adds numAlleles pairs of contrasting alleles to this individual
func (ind *Individual) AddInitialContrastingAlleles(numAlleles uint32, uniformRandom *rand.Rand) (uint32, uint32) {
	// Spread the allele pairs throughout the LBs as evenly as possible: if numAlleles < num_linkage_subunits then skip some LBs to
	// space the allele pairs evenly. If numAlleles == num_linkage_subunits then 1 allele pair per LB. If numAlleles > num_linkage_subunits then
	// every LB gets some allele pairs and space the rest out evenly.
	allelesPerLB := numAlleles / config.Cfg.Population.Num_linkage_subunits		// every LB gets this many allele pairs
	allelesRemainder := numAlleles % config.Cfg.Population.Num_linkage_subunits		// spread this many allele pairs evenly over the LBs
	config.Verbose(9, " Intending to give %v allele pairs to each LB, and spread %v allele pairs among all LBs", allelesPerLB, allelesRemainder)

	// Use the same approach as pop.GenerateInitialAlleles() for spreading allelesRemainder: keep a running ratio of LBs with alleles / LBs processed
	desiredRemainderRatio := float64(allelesRemainder) / float64(config.Cfg.Population.Num_linkage_subunits)
	var numWithAllelesRemainder uint32 = 0		// used to calc the running ratio of the number of remainers we've passed out
	var numWithAllelesEvenly uint32 = 0		// keep track of the number of allele pairs we evenly give out to every LB
	var numProcessedLBs uint32 = 0		// start at 0 because it is the number from the previous iteration of the loop

	for c := range ind.ChromosomesFromDad {
		for lb := range ind.ChromosomesFromDad[c].LinkageBlocks {
			// If there are some allele pairs on every LB
			for i:=1; i<=int(allelesPerLB); i++ {
				config.Verbose(9, " Appending initial alleles to chromosome[%v].LB[%v]", c, lb)
				// Note: we can use the global UniqueInt object because this method is called before we create go routines.
				//dna.AppendInitialContrastingAlleles(ind.ChromosomesFromDad[c].LinkageBlocks[lb], ind.ChromosomesFromMom[c].LinkageBlocks[lb], utils.GlobalUniqueInt, uniformRandom)
				dna.ChrAppendInitialContrastingAlleles(&ind.ChromosomesFromDad[c], &ind.ChromosomesFromMom[c], lb, utils.GlobalUniqueInt, uniformRandom)
				numWithAllelesEvenly++
				//ind.popPart.Pop.NextMutId += 2
			}

			// Decide if this LB should get 1 of the remaining alleles
			var ratioSoFar float64
			if numProcessedLBs > 0 { ratioSoFar = float64(numWithAllelesRemainder) / float64(numProcessedLBs) }
			// else ratioSoFar = 0
			if ratioSoFar <= desiredRemainderRatio && numWithAllelesRemainder < allelesRemainder {
				config.Verbose(9, " Appending initial alleles to chromosome[%v].LB[%v]", c, lb)
				//dna.AppendInitialContrastingAlleles(ind.ChromosomesFromDad[c].LinkageBlocks[lb], ind.ChromosomesFromMom[c].LinkageBlocks[lb], utils.GlobalUniqueInt, uniformRandom)
				dna.ChrAppendInitialContrastingAlleles(&ind.ChromosomesFromDad[c], &ind.ChromosomesFromMom[c], lb, utils.GlobalUniqueInt, uniformRandom)
				numWithAllelesRemainder++
				//ind.popPart.Pop.NextMutId += 2
			}

			numProcessedLBs++
		}
	}
	ind.NumMutations += numAlleles * 2

	//config.Verbose(5, " Initial alleles given to %v faction of LBs in the individual ((%v+%v)/%v)", float64(numWithAllelesRemainder + numWithAllelesEvenly) / float64(numProcessedLBs), numWithAllelesEvenly, numWithAllelesRemainder, numProcessedLBs)
	return numWithAllelesRemainder + numWithAllelesEvenly, numProcessedLBs
}


// Various algorithms for determining the random number of offspring for a mating pair of individuals
type CalcNumOffspringType func(ind *Individual, uniformRandom *rand.Rand) uint32

// A uniform algorithm for calculating the number of offspring that gives an even distribution between 1 and 2*(Num_offspring*2)-1
func CalcUniformNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	// If (Num_offspring*2) is 4.5, we want a range from 1-8
	maxRange := (2 * ind.popPart.Pop.Num_offspring * 2) - 2 		// subtract 2 to get a buffer of 1 at each end
	numOffspring := uniformRandom.Float64() * maxRange 		// some float between 0 and maxRange
	return uint32(random.Round(uniformRandom, numOffspring + 1)) 	// shift it so it is between 1 and maxRange+1, then get to an uint32
}


// Randomly rounds the desired number of offspring to the integer below or above, proportional to how close it is to each (so the resulting average should be (Num_offspring*2) )
func CalcSemiFixedNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	return uint32(random.Round(uniformRandom, ind.popPart.Pop.Num_offspring*2))
}


/* This turns out to be functionally equivalent to CalcSemiFixedNumOffspring, except in CalcSemiFixedNumOffspring if ind.popPart.Pop.Num_offspring is a whole number (common case) it
  doesn't invoke uniformRandom.Float64(). That results in different results between CalcFortranNumOffspring and CalcSemiFixedNumOffspring simply due to different
  random number sequences.
// An algorithm taken from the fortran mendel for calculating the number of offspring.
func CalcFortranNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	// This logic is from lines 64-73 of mating.f90
	offspring_per_pair := ind.popPart.Pop.Num_offspring * 2
	actual_offspring := uint32(offspring_per_pair)		// truncate num offspring to integer
	if offspring_per_pair - float64(actual_offspring) > uniformRandom.Float64() { actual_offspring++ }	// randomly round it up sometimes
	//if offspring_per_pair - float64(uint32(offspring_per_pair)) > uniformRandom.Float64() { actual_offspring++ }	// randomly round it up sometimes
	//if indivIndex == 1 { actual_offspring = utils.Max(1, actual_offspring) } 	// assuming this was some special case specific to the fortran implementation
	//actual_offspring = utils.MinUint32(uint32(offspring_per_pair+1), actual_offspring) 	// does not seem like this line does anything, because actual_offspring will always be uint32(offspring_per_pair)+1 or uint32(offspring_per_pair)
	return actual_offspring
}
*/


// Randomly choose a number of offspring that is, on average, proportional to the individual's fitness
func CalcFitnessNumOffspring(ind *Individual, uniformRandom *rand.Rand) uint32 {
	// in the fortran version this is controlled by fitness_dependent_fertility
	utils.NotImplementedYet("CalcFitnessNumOffspring not implemented yet")
	return uint32(random.Round(uniformRandom, ind.popPart.Pop.Num_offspring*2))
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
	// Note: AddMutations() will cache the fitness
	return
}

// Note implemented yet. MultIndivFitness aggregates the fitness factors of all of the mutations using a combination of additive and mutliplicative,
// based on config.Cfg.Mutations.Multiplicative_weighting
func MultIndivFitness(_ *Individual) (fitness float64) {
	fitness = 1.0
	// do not know the exact formula to use for this yet
	utils.NotImplementedYet("Multiplicative_weighting not implemented yet")
	return fitness
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the average fitness factor of deleterious and favorable
func (ind *Individual) GetMutationStats() (uint32, uint32, uint32 /*, float64, float64*/) {
	// See if we already calculated and cached the values. Note: we only check deleterious, because fav and neutral could be 0
	if ind.NumDeleterious > 0 /*&& ind.MeanDelFit < 0.0*/ { return ind.NumDeleterious, ind.NumNeutral, ind.NumFavorable /*, ind.MeanDelFit, ind.MeanFavFit*/ }
	ind.NumDeleterious=0;  ind.NumNeutral=0;  ind.NumFavorable=0  //;  ind.MeanDelFit=0.0;  ind.MeanFavFit=0.0

	// Calc the average of each type of mutation: multiply the average from each chromosome and num mutns from each chromosome, then at the end divide by total num mutns
	for _, c := range ind.ChromosomesFromDad {
		delet, neut, fav /*, avD, avF*/ := c.GetMutationStats()
		ind.NumDeleterious += delet
		ind.NumNeutral += neut
		ind.NumFavorable += fav
		//ind.MeanDelFit += (float64(delet) * avD)
		//ind.MeanFavFit += (float64(fav) * avF)
	}
	for _, c := range ind.ChromosomesFromMom {
		delet, neut, fav /*, avD, avF*/ := c.GetMutationStats()
		ind.NumDeleterious += delet
		ind.NumNeutral += neut
		ind.NumFavorable += fav
		//ind.MeanDelFit += (float64(delet) * avD)
		//ind.MeanFavFit += (float64(fav) * avF)
	}
	//if ind.NumDeleterious > 0 { ind.MeanDelFit = ind.MeanDelFit / float64(ind.NumDeleterious) }
	//if ind.NumFavorable > 0 { ind.MeanFavFit = ind.MeanFavFit / float64(ind.NumFavorable) }
	return ind.NumDeleterious, ind.NumNeutral, ind.NumFavorable  //, ind.MeanDelFit, ind.MeanFavFit
}


// GetInitialAlleleStats returns the number of deleterious, neutral, favorable initial alleles, and the average fitness factor of deleterious and favorable
func (ind *Individual) GetInitialAlleleStats() (uint32, uint32, uint32 /*, float64, float64*/ ) {
	// See if we already calculated and cached the values. Note: we only check deleterious, because fav and neutral could be 0
	if ind.NumDelAllele > 0 /*&& ind.MeanDelAlleleFit < 0.0*/ { return ind.NumDelAllele, ind.NumNeutAllele, ind.NumFavAllele /*, ind.MeanDelAlleleFit, ind.MeanFavAlleleFit*/ }
	ind.NumDelAllele=0;  ind.NumNeutAllele=0;  ind.NumFavAllele=0  //;  ind.MeanDelAlleleFit=0.0;  ind.MeanFavAlleleFit=0.0

	// Calc the average of each type of allele: multiply the average from each chromosome and num alleles from each chromosome, then at the end divide by total num alleles
	for _, c := range ind.ChromosomesFromDad {
		delet, neut, fav /*, avD, avF*/ := c.GetInitialAlleleStats()
		ind.NumDelAllele += delet
		ind.NumNeutAllele += neut
		ind.NumFavAllele += fav
		//ind.MeanDelAlleleFit += (float64(delet) * avD)
		//ind.MeanFavAlleleFit += (float64(fav) * avF)
	}
	for _, c := range ind.ChromosomesFromMom {
		delet, neut, fav /*, avD, avF*/ := c.GetInitialAlleleStats()
		ind.NumDelAllele += delet
		ind.NumNeutAllele += neut
		ind.NumFavAllele += fav
		//ind.MeanDelAlleleFit += (float64(delet) * avD)
		//ind.MeanFavAlleleFit += (float64(fav) * avF)
	}
	//if ind.NumDelAllele > 0 { ind.MeanDelAlleleFit = ind.MeanDelAlleleFit / float64(ind.NumDelAllele) }
	//if ind.NumFavAllele > 0 { ind.MeanFavAlleleFit = ind.MeanFavAlleleFit / float64(ind.NumFavAllele) }
	return ind.NumDelAllele, ind.NumNeutAllele, ind.NumFavAllele  //, ind.MeanDelAlleleFit, ind.MeanFavAlleleFit
}


/*
// GatherAlleles adds all of this individual's alleles (both mutations and initial alleles) to the given struct
func (ind *Individual) GatherAlleles(alleles *dna.Alleles) {
	for _, c := range ind.ChromosomesFromDad { c.GatherAlleles(alleles) }
	for _, c := range ind.ChromosomesFromMom { c.GatherAlleles(alleles) }
}
*/


// CountAlleles counts all of this individual's alleles (both mutations and initial alleles) and adds them to the given struct
func (ind *Individual) CountAlleles(alleles *dna.AlleleCount) {
	// Get the alleles for this individual
	allelesForThisIndiv := dna.AlleleCountFactory()		// so we don't double count the same allele from both parents, the count in this map for each allele found is always 1
	for _, c := range ind.ChromosomesFromDad { c.CountAlleles(allelesForThisIndiv) }
	for _, c := range ind.ChromosomesFromMom { c.CountAlleles(allelesForThisIndiv) }

	// Add the alleles found for this individual to the alleles map for the whole population
	// Note: map returns the zero value of the value type for keys which are not yet in the map (zero value for int is 0), so we do not need to check if it is there with: if count, ok := alleles.Deleterious[id]; ok {
	for id := range allelesForThisIndiv.Deleterious { alleles.Deleterious[id] += 1 }
	for id := range allelesForThisIndiv.Neutral { alleles.Neutral[id] += 1 }
	for id := range allelesForThisIndiv.Favorable { alleles.Favorable[id] += 1 }
	for id := range allelesForThisIndiv.DelInitialAlleles { alleles.DelInitialAlleles[id] += 1 }
	for id := range allelesForThisIndiv.FavInitialAlleles { alleles.FavInitialAlleles[id] += 1 }
}


// Report prints out statistics of this individual. If final==true it could print more details.
func (ind *Individual) Report(_ bool) {
	deleterious, neutral, favorable /*, avDelFit, avFavFit*/ := ind.GetMutationStats()
	//log.Printf("  Ind: fitness: %v, mutations: %d, deleterious: %d, neutral: %d, favorable: %d, avg del: %v, avg fav: %v", ind.GenoFitness, deleterious+neutral+favorable, deleterious, neutral, favorable, avDelFit, avFavFit)
	log.Printf("  Ind: fitness: %v, mutations: %d, deleterious: %d, neutral: %d, favorable: %d", ind.GenoFitness, deleterious+neutral+favorable, deleterious, neutral, favorable)
}
