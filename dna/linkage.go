package dna

import (
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"unsafe"
)

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB
// and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	//TotalFitness float64   // keep a running total of the LB's fitness if the combination model is only additive?
	DMutn                  []*DeleteriousMutation
	FMutn                  []*FavorableMutation
	//todo: should/can we combine the Tracked/Untracked FitnessEffect vars to save space?
	TrackedDelFitnessEffect float64 // even for the tracked mutations (that are in the arrays above) keep a running sum of the fitness so we can calc the LB fitness quickly
	TrackedFavFitnessEffect float64

	UntrackedDelFitnessEffect float64 // these 4 members are used instead of some or all of the individual mutations if Tracking_threshold is set
	UntrackedFavFitnessEffect float64
	NumUntrackedDeleterious         uint16
	NumUntrackedFavorable           uint16

	NMutn                  []*NeutralMutation
	NumUntrackedNeutrals   uint16  // this is used instead of the array above if track_neutrals==false

	DAllele []*DeleteriousAllele	// initial alleles
	FAllele []*FavorableAllele
	AlleleDelFitnessEffect float64 // keep a running sum of the fitness so we can calc the LB fitness quickly
	AlleleFavFitnessEffect float64
	//NAllele []*NeutralAllele   // do not know of any reason to have these

	Owner *Chromosome		// keep track of owner so we know whether we have to copy this LB or can just transfer ownership
}


func LinkageBlockFactory(owner *Chromosome) *LinkageBlock {
	// Initially there are no mutations.
	// Note: there is no need to allocate the mutation slices with backing arrays. That will happen automatically the 1st time they are
	//		appended to with append(). See https://blog.golang.org/go-slices-usage-and-internals
	return &LinkageBlock{Owner: owner}
}


// Transfer will give the "to" chromosome (of the child) the equivalent LB, by just transferring ownership of this LB instance (if it has not already given it
// to another child), or by copying the LB if it must. The reason transfer of ownership to a child is ok is once this function is called, the "from" chromosome (the parent) will
// never do anything with this LB again, except maybe copy the contents to another child. From this perspective, it is also important that
// the children of these parents not do anything with their LBs until the parents are done creating all of their children.
func (lb *LinkageBlock) Transfer(from, to *Chromosome, lbIndex int) {
	if lb.Owner == from && config.Cfg.Computation.Transfer_linkage_blocks {
		// "From" still owns this LB, so it is his to give away
		//config.Verbose(9, " Transferring ownership of LB %p from %p to %p", lb, from, to)
		to.LinkageBlocks[lbIndex] = lb
		lb.Owner = to
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
	if len(lb.DMutn) > 0 {
		newLb.DMutn = make([]*DeleteriousMutation, len(lb.DMutn)) 	// allocate a new underlying array the same length as the source
		copy(newLb.DMutn, lb.DMutn)        // this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)
	}
	newLb.NumUntrackedDeleterious = lb.NumUntrackedDeleterious
	newLb.UntrackedDelFitnessEffect = lb.UntrackedDelFitnessEffect
	newLb.TrackedDelFitnessEffect = lb.TrackedDelFitnessEffect

	if len(lb.NMutn) > 0 {
		newLb.NMutn = make([]*NeutralMutation, len(lb.NMutn))
		copy(newLb.NMutn, lb.NMutn)
	}
	newLb.NumUntrackedNeutrals = lb.NumUntrackedNeutrals
	//if len(lb.NMutn) > 0 || lb.NumNeutrals > 0 { config.Verbose(3, "inheriting %d neutral mutations and %d num neutral", len(lb.NMutn), lb.NumNeutrals) }

	if len(lb.FMutn) > 0 {
		newLb.FMutn = make([]*FavorableMutation, len(lb.FMutn))
		copy(newLb.FMutn, lb.FMutn)
	}
	newLb.NumUntrackedFavorable = lb.NumUntrackedFavorable
	newLb.UntrackedFavFitnessEffect = lb.UntrackedFavFitnessEffect
	newLb.TrackedFavFitnessEffect = lb.TrackedFavFitnessEffect

	if len(lb.DAllele) > 0 {
		newLb.DAllele = make([]*DeleteriousAllele, len(lb.DAllele))
		copy(newLb.DAllele, lb.DAllele)
	}
	newLb.AlleleDelFitnessEffect = lb.AlleleDelFitnessEffect
	/*
	if len(lb.NAllele) > 0 {
		newLb.NAllele = make([]*NeutralAllele, len(lb.NAllele))
		copy(newLb.NAllele, lb.NAllele)
	}
	*/
	if len(lb.FAllele) > 0 {
		newLb.FAllele = make([]*FavorableAllele, len(lb.FAllele))
		copy(newLb.FAllele, lb.FAllele)
	}
	newLb.AlleleFavFitnessEffect = lb.AlleleFavFitnessEffect

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
		mutn := DeleteriousMutationFactory(uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || mutn.GetFitnessEffect() < -config.Cfg.Computation.Tracking_threshold {
			lb.DMutn = append(lb.DMutn, mutn)
			lb.TrackedDelFitnessEffect += mutn.GetFitnessEffect()
		} else {
			// Currently the code that checks the input file only allows a Tracking_threshold if the combination model is additive
			lb.UntrackedDelFitnessEffect += mutn.GetFitnessEffect()
			lb.NumUntrackedDeleterious++
		}
	case NEUTRAL:
		if config.Cfg.Computation.Track_neutrals {
			//config.Verbose(3, "adding a neutral mutation")
			lb.NMutn = append(lb.NMutn, NeutralMutationFactory(uniformRandom))
		} else {
			//config.Verbose(3, "adding to the neutral mutation count")
			lb.NumUntrackedNeutrals++
		}
	case FAVORABLE:
		mutn := FavorableMutationFactory(uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || mutn.GetFitnessEffect() > config.Cfg.Computation.Tracking_threshold {
			lb.FMutn = append(lb.FMutn, mutn)
			lb.TrackedFavFitnessEffect += mutn.GetFitnessEffect()
		} else {
			// Currently the code that checks the input file only allows a Tracking_threshold if the combination model is additive
			lb.UntrackedFavFitnessEffect += mutn.GetFitnessEffect()
			lb.NumUntrackedFavorable++
		}
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
	favAllele := FavorableAlleleFactory(fitnessEffect)
	lb1.FAllele = append(lb1.FAllele, favAllele)
	lb1.AlleleFavFitnessEffect += favAllele.GetFitnessEffect()

	// Add a deleterious allele to the 1st LB
	delAllele := DeleteriousAlleleFactory(-fitnessEffect)
	lb2.DAllele = append(lb2.DAllele, delAllele)
	lb2.AlleleDelFitnessEffect += delAllele.GetFitnessEffect()
}


// SumFitness combines the fitness effect of all of its mutations in the additive method
func (lb *LinkageBlock) SumFitness() (fitness float64) {
	//fitness = 0.0
	// Note: the deleterious mutation fitness factors are already negative.
	//for _, m := range lb.DMutn { fitness += m.GetFitnessEffect() }
	//for _, m := range lb.FMutn { fitness += m.GetFitnessEffect() }
	// If there are no untracked mutations, this is still ok
	fitness = lb.UntrackedDelFitnessEffect + lb.UntrackedFavFitnessEffect + lb.TrackedDelFitnessEffect + lb.TrackedFavFitnessEffect + lb.AlleleDelFitnessEffect + lb.AlleleFavFitnessEffect
	return
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the mean fitness factor of each.
// Note: the mean fitnesses take into account whether or not the mutation is expressed, so even for fixed mutation fitness the mean will not be that value.
func (lb *LinkageBlock) GetMutationStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(len(lb.DMutn)) + uint32(lb.NumUntrackedDeleterious)
	avDelFit = lb.UntrackedDelFitnessEffect + lb.TrackedDelFitnessEffect
	if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) } 		// else avDelFit is already 0.0

	neutral = uint32(len(lb.NMutn)) + uint32(lb.NumUntrackedNeutrals)

	favorable = uint32(len(lb.FMutn)) + uint32(lb.NumUntrackedFavorable)
	avFavFit = lb.UntrackedFavFitnessEffect + lb.TrackedFavFitnessEffect
	if favorable > 0 { avFavFit = avFavFit / float64(favorable) } 		// else avFavFit is already 0.0
	return
}


// GetInitialAlleleStats returns the number of deleterious, neutral, favorable initial alleles, and the mean fitness factor of each.
func (lb *LinkageBlock) GetInitialAlleleStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(len(lb.DAllele))
	if deleterious > 0 { avDelFit = lb.AlleleDelFitnessEffect / float64(deleterious) } 		// else avDelFit is already 0.0

	//neutral = uint32(len(lb.NAllele))
	neutral = 0

	favorable = uint32(len(lb.FAllele))
	if favorable > 0 { avFavFit = lb.AlleleFavFitnessEffect / float64(favorable) } 		// else avFavFit is already 0.0
	return
}


/*
// GatherAlleles adds all of this LB's alleles (both mutations and initial alleles) to the given struct
func (lb *LinkageBlock) GatherAlleles(alleles *Alleles) {
	for _, m := range lb.DMutn {
		// Use the ptr to the mutation object as its unique identifier for the allele bin data file.
		// Note: Go calls this pkg unsafe because is can be used to avoid type safety and access arbitrary memory. But we aren't doing
		// any of those bad things. We are just converting the mutation object ptr to an int so that printing it to the alleles bin
		// data file is in a format easier to consume.
		id := uintptr(unsafe.Pointer(m))
		alleles.Deleterious = append(alleles.Deleterious, id)
	}
	for _, m := range lb.NMutn {
		id := uintptr(unsafe.Pointer(m))
		alleles.Neutral = append(alleles.Neutral, id)
	}
	for _, m := range lb.FMutn {
		id := uintptr(unsafe.Pointer(m))
		alleles.Favorable = append(alleles.Favorable, id)
	}
	for _, a := range lb.DAllele {
		id := uintptr(unsafe.Pointer(a))
		alleles.DelInitialAlleles = append(alleles.DelInitialAlleles, id)
	}
	for _, a := range lb.FAllele {
		id := uintptr(unsafe.Pointer(a))
		alleles.FavInitialAlleles = append(alleles.FavInitialAlleles, id)
	}
}
*/


// CountAlleles counts all of this LB's alleles (both mutations and initial alleles) and adds them to the given struct
func (lb *LinkageBlock) CountAlleles(allelesForThisIndiv *AlleleCount) {
	// We are getting the alleles for just this individual so we don't want to double count the same allele from both parents,
	// so we only ever set the value to 1 for a particular allele id.
	for _, m := range lb.DMutn {
		// Use the ptr to the mutation object as the key in the map.
		id := uintptr(unsafe.Pointer(m))
		//alleles.Deleterious[id] += 1
		allelesForThisIndiv.Deleterious[id] = 1
	}
	for _, m := range lb.NMutn {
		id := uintptr(unsafe.Pointer(m))
		//alleles.Neutral[id] += 1
		allelesForThisIndiv.Neutral[id] = 1
	}
	for _, m := range lb.FMutn {
		id := uintptr(unsafe.Pointer(m))
		//alleles.Favorable[id] += 1
		allelesForThisIndiv.Favorable[id] = 1
	}
	for _, a := range lb.DAllele {
		id := uintptr(unsafe.Pointer(a))
		//alleles.DelInitialAlleles[id] += 1
		allelesForThisIndiv.DelInitialAlleles[id] = 1
	}
	for _, a := range lb.FAllele {
		id := uintptr(unsafe.Pointer(a))
		//alleles.FavInitialAlleles[id] += 1
		allelesForThisIndiv.FavInitialAlleles[id] = 1
	}
}
