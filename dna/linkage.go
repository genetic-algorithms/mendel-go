package dna

import (
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"unsafe"
)

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB
// and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	dMutn []*DeleteriousMutation
	fMutn []*FavorableMutation

	// These can be combined to save space
	//TrackedDelFitnessEffect float64 // even for the tracked mutations (that are in the arrays above) keep a running sum of the fitness so we can calc the LB fitness quickly
	//TrackedFavFitnessEffect float64
	//UntrackedDelFitnessEffect float64 // these 4 members are used instead of some or all of the individual mutations if Tracking_threshold is set
	//UntrackedFavFitnessEffect float64

	// This is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 11th decimal place. It saves approx 160MB for a 10,000 pop, plus the time for allocating and copying the extra mem
	//delFitnessEffect        float64              // keep a running sum of the fitness so we can calc the LB fitness quickly
	//favFitnessEffect        float64
	delFitnessEffect        float32              // keep a running sum of the fitness so we can calc the LB fitness quickly.
	favFitnessEffect        float32

	numUntrackedDeleterious uint16
	numUntrackedFavorable   uint16

	nMutn                   []*NeutralMutation
	numUntrackedNeutrals    uint16               // this is used instead of the array above if track_neutrals==false

	dAllele                 []*DeleteriousAllele // initial alleles
	fAllele                 []*FavorableAllele
	//alleleDelFitnessEffect  float64              // keep a running sum of the fitness so we can calc the LB fitness quickly
	//alleleFavFitnessEffect  float64
	alleleDelFitnessEffect  float32              // keep a running sum of the fitness so we can calc the LB fitness quickly
	alleleFavFitnessEffect  float32
	//NAllele []*NeutralAllele   // do not know of any reason to have these

	owner                   *Chromosome          // keep track of owner so we know whether we have to copy this LB or can just transfer ownership
}


func LinkageBlockFactory(owner *Chromosome) *LinkageBlock {
	// Initially there are no mutations.
	// Note: there is no need to allocate the mutation slices with backing arrays. That will happen automatically the 1st time they are
	//		appended to with append(). See https://blog.golang.org/go-slices-usage-and-internals
	return &LinkageBlock{owner: owner}
}


// Transfer will give the "to" chromosome (of the child) the equivalent LB, by just transferring ownership of this LB instance (if it has not already given it
// to another child), or by copying the LB if it must. The reason transfer of ownership to a child is ok is once this function is called, the "from" chromosome (the parent) will
// never do anything with this LB again, except maybe copy the contents to another child. From this perspective, it is also important that
// the children of these parents not do anything with their LBs until the parents are done creating all of their children.
func (lb *LinkageBlock) Transfer(from, to *Chromosome, lbIndex int) {
	if lb.owner == from && config.Cfg.Computation.Transfer_linkage_blocks {
		// "From" still owns this LB, so it is his to give away
		//config.Verbose(9, " Transferring ownership of LB %p from %p to %p", lb, from, to)
		to.LinkageBlocks[lbIndex] = lb
		lb.owner = to
	} else {
		// This LB has already been given away to another offspring, so need to make a copy
		//config.Verbose(2, "copying LB")
		to.LinkageBlocks[lbIndex] = lb.Copy(to)
		// maybe try moving Copy() contents here to avoid another function call
	}
}


// Copy makes a semi-deep copy (makes a copy of the array of pointers to mutations, but does *not* copy the mutations themselves, because they are immutable) and returns it
func (lb *LinkageBlock) Copy(owner *Chromosome) *LinkageBlock {
	newLb := LinkageBlockFactory(owner)
	// Assigning a slice does not copy all the array elements, so we have to make that happen
	if len(lb.dMutn) > 0 {
		newLb.dMutn = make([]*DeleteriousMutation, len(lb.dMutn)) 	// allocate a new underlying array the same length as the source
		copy(newLb.dMutn, lb.dMutn)        // this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)
	}
	newLb.numUntrackedDeleterious = lb.numUntrackedDeleterious
	//newLb.UntrackedDelFitnessEffect = lb.UntrackedDelFitnessEffect
	//newLb.TrackedDelFitnessEffect = lb.TrackedDelFitnessEffect
	newLb.delFitnessEffect = lb.delFitnessEffect

	if len(lb.nMutn) > 0 {
		newLb.nMutn = make([]*NeutralMutation, len(lb.nMutn))
		copy(newLb.nMutn, lb.nMutn)
	}
	newLb.numUntrackedNeutrals = lb.numUntrackedNeutrals
	//if len(lb.NMutn) > 0 || lb.NumNeutrals > 0 { config.Verbose(3, "inheriting %d neutral mutations and %d num neutral", len(lb.NMutn), lb.NumNeutrals) }

	if len(lb.fMutn) > 0 {
		newLb.fMutn = make([]*FavorableMutation, len(lb.fMutn))
		copy(newLb.fMutn, lb.fMutn)
	}
	newLb.numUntrackedFavorable = lb.numUntrackedFavorable
	//newLb.UntrackedFavFitnessEffect = lb.UntrackedFavFitnessEffect
	//newLb.TrackedFavFitnessEffect = lb.TrackedFavFitnessEffect
	newLb.favFitnessEffect = lb.favFitnessEffect

	if len(lb.dAllele) > 0 {
		newLb.dAllele = make([]*DeleteriousAllele, len(lb.dAllele))
		copy(newLb.dAllele, lb.dAllele)
	}
	newLb.alleleDelFitnessEffect = lb.alleleDelFitnessEffect
	/*
	if len(lb.NAllele) > 0 {
		newLb.NAllele = make([]*NeutralAllele, len(lb.NAllele))
		copy(newLb.NAllele, lb.NAllele)
	}
	*/
	if len(lb.fAllele) > 0 {
		newLb.fAllele = make([]*FavorableAllele, len(lb.fAllele))
		copy(newLb.fAllele, lb.fAllele)
	}
	newLb.alleleFavFitnessEffect = lb.alleleFavFitnessEffect

	return newLb
}


/* currently not used...
// GetTotalMutnCount returns the number of mutations currently on this LB
func (lb *LinkageBlock) GetTotalMutnCount() uint32 {
	// Every mutation added to the LB is either in 1 of the arrays or in 1 of the Untracked vars (but not both), so it is ok to sum them all
	return uint32(len(lb.DMutn)) + uint32(lb.NumUntrackedDeleterious) + uint32(len(lb.NMutn)) + uint32(lb.NumUntrackedNeutrals) + uint32(len(lb.FMutn)) + uint32(lb.NumUntrackedFavorable)
}
*/


// AppendMutation creates and adds a mutation to this LB.
// Note: The implementation of golang's append() appears to be that if it has to copy the array is doubles the capacity, which is probably what we want for the Mutation arrays.
func (lb *LinkageBlock) AppendMutation(uniformRandom *rand.Rand) {
	mType := CalcMutationType(uniformRandom)
	switch mType {
	case DELETERIOUS:
		fitnessEffect := calcDelMutationAttrs(uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || fitnessEffect < -config.Cfg.Computation.Tracking_threshold {
			// We are tracking this mutation, so create it and append
			mutn := DeleteriousMutationFactory(fitnessEffect, uniformRandom)
			lb.dMutn = append(lb.dMutn, mutn)
		} else {
			lb.numUntrackedDeleterious++
		}
		// Currently the code that checks the input file only allows a Tracking_threshold if the combination model is additive, so this is appropriate
		lb.delFitnessEffect += fitnessEffect
	case NEUTRAL:
		if config.Cfg.Computation.Track_neutrals {
			//config.Verbose(3, "adding a neutral mutation")
			lb.nMutn = append(lb.nMutn, NeutralMutationFactory(uniformRandom))
		} else {
			//config.Verbose(3, "adding to the neutral mutation count")
			lb.numUntrackedNeutrals++
		}
	case FAVORABLE:
		fitnessEffect := calcFavMutationAttrs(uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || fitnessEffect > config.Cfg.Computation.Tracking_threshold {
			// We are tracking this mutation, so create it and append
			mutn := FavorableMutationFactory(fitnessEffect, uniformRandom)
			lb.fMutn = append(lb.fMutn, mutn)
		} else {
			lb.numUntrackedFavorable++
		}
		// Currently the code that checks the input file only allows a Tracking_threshold if the combination model is additive, so this is appropriate
		lb.favFitnessEffect += fitnessEffect
	}
}


// AppendInitialContrastingAlleles adds an initial contrasting allele pair to 2 LBs (favorable to 1, deleterious to the other).
// The 2 LBs passed in are typically the same LB position on the same chromosome number, 1 from each parent.
func AppendInitialContrastingAlleles(lb1, lb2 *LinkageBlock, uniformRandom *rand.Rand) {
	// Note: for now we assume that all initial contrasting alleles are co-dominant so that in the homozygous case (2 of the same favorable
	//		allele (or 2 of the deleterious allele) - 1 from each parent), the combineb fitness effect is 1.0 * the allele fitness.
	expression := 0.5
	fitnessEffect := Mdl.CalcAlleleFitness(uniformRandom) * expression

	// Add a favorable allele to the 1st LB
	// Note: we assume that if initial alleles are being created, they are being tracked
	favAllele := FavorableAlleleFactory(fitnessEffect)
	lb1.fAllele = append(lb1.fAllele, favAllele)
	lb1.alleleFavFitnessEffect += favAllele.GetFitnessEffect()

	// Add a deleterious allele to the 2nd LB
	delAllele := DeleteriousAlleleFactory(-fitnessEffect)
	lb2.dAllele = append(lb2.dAllele, delAllele)
	lb2.alleleDelFitnessEffect += delAllele.GetFitnessEffect()
}


// SumFitness combines the fitness effect of all of its mutations in the additive method
func (lb *LinkageBlock) SumFitness() (fitness float64) {
	//fitness = 0.0
	// Note: the deleterious mutation fitness factors are already negative.
	//for _, m := range lb.DMutn { fitness += m.GetFitnessEffect() }
	//for _, m := range lb.FMutn { fitness += m.GetFitnessEffect() }
	// If there are no untracked mutations, this is still ok
	//fitness = lb.UntrackedDelFitnessEffect + lb.UntrackedFavFitnessEffect + lb.TrackedDelFitnessEffect + lb.TrackedFavFitnessEffect + lb.AlleleDelFitnessEffect + lb.AlleleFavFitnessEffect
	fitness = float64(lb.delFitnessEffect + lb.favFitnessEffect + lb.alleleDelFitnessEffect + lb.alleleFavFitnessEffect)
	return
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the mean fitness factor of each.
// Note: the mean fitnesses take into account whether or not the mutation is expressed, so even for fixed mutation fitness the mean will not be that value.
func (lb *LinkageBlock) GetMutationStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(len(lb.dMutn)) + uint32(lb.numUntrackedDeleterious)
	//avDelFit = lb.UntrackedDelFitnessEffect + lb.TrackedDelFitnessEffect
	avDelFit = float64(lb.delFitnessEffect)
	if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) } 		// else avDelFit is already 0.0

	neutral = uint32(len(lb.nMutn)) + uint32(lb.numUntrackedNeutrals)

	favorable = uint32(len(lb.fMutn)) + uint32(lb.numUntrackedFavorable)
	//avFavFit = lb.UntrackedFavFitnessEffect + lb.TrackedFavFitnessEffect
	avFavFit = float64(lb.favFitnessEffect)
	if favorable > 0 { avFavFit = avFavFit / float64(favorable) } 		// else avFavFit is already 0.0
	return
}


// GetInitialAlleleStats returns the number of deleterious, neutral, favorable initial alleles, and the mean fitness factor of each.
func (lb *LinkageBlock) GetInitialAlleleStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(len(lb.dAllele))
	if deleterious > 0 { avDelFit = float64(lb.alleleDelFitnessEffect) / float64(deleterious) } 		// else avDelFit is already 0.0

	//neutral = uint32(len(lb.NAllele))
	neutral = 0

	favorable = uint32(len(lb.fAllele))
	if favorable > 0 { avFavFit = float64(lb.alleleFavFitnessEffect) / float64(favorable) } 		// else avFavFit is already 0.0
	return
}


// CountAlleles counts all of this LB's alleles (both mutations and initial alleles) and adds them to the given struct
func (lb *LinkageBlock) CountAlleles(allelesForThisIndiv *AlleleCount) {
	// We are getting the alleles for just this individual so we don't want to double count the same allele from both parents,
	// so we only ever set the value to 1 for a particular allele id.
	for _, m := range lb.dMutn {
		// Use the ptr to the mutation object as the key in the map.
		id := uintptr(unsafe.Pointer(m))
		//alleles.Deleterious[id] += 1
		allelesForThisIndiv.Deleterious[id] = 1
	}
	for _, m := range lb.nMutn {
		id := uintptr(unsafe.Pointer(m))
		//alleles.Neutral[id] += 1
		allelesForThisIndiv.Neutral[id] = 1
	}
	for _, m := range lb.fMutn {
		id := uintptr(unsafe.Pointer(m))
		//alleles.Favorable[id] += 1
		allelesForThisIndiv.Favorable[id] = 1
	}
	for _, a := range lb.dAllele {
		id := uintptr(unsafe.Pointer(a))
		//alleles.DelInitialAlleles[id] += 1
		allelesForThisIndiv.DelInitialAlleles[id] = 1
	}
	for _, a := range lb.fAllele {
		id := uintptr(unsafe.Pointer(a))
		//alleles.FavInitialAlleles[id] += 1
		allelesForThisIndiv.FavInitialAlleles[id] = 1
	}
}
