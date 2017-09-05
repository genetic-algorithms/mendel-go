package dna

import (
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"log"
	//"unsafe"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)

// Note: with a typical 10K population (30K during mating) and 989 LBs per individual there are a lot of LBs, so saving
//		space in them is important.

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	mutn []Mutation		// holds deleterious, neutral, favorable, initial deleterious, initial favorable
	fitnessEffect float32
	//delFitnessEffect       float32              // keep a running sum of the fitness so we can calc the LB fitness quickly.
	//favFitnessEffect       float32
	numDeleterious         uint16
	numFavorable           uint16
	numNeutrals            uint16               // this is used instead of the array above if track_neutrals==false
	numDelAllele uint16
	numFavAllele uint16
	//alleleDelFitnessEffect float32              // keep a running sum of the fitness so we can calc the LB fitness quickly
	//alleleFavFitnessEffect float32
	//TODO: remove if we no longer transfer ownership
	//owner                   *Chromosome         // keep track of owner so we know whether we have to copy this LB or can just transfer ownership
}

/*
// LBMutations holds the mutations for 1 LB in 1 individual in 1 generation. As the generations progress, an LB has
// a chain of these LBMutations back thru its ancestors. (This avoids copying ptrs to mutations during mating.)
// The stats are not stored in this struct, because those only have to be stored in the latest generation (because they are cumulative).
// The goal is that in the case of high mutation rates and many generations the time will scale linearly with num of gens
// because each gen you only have to process the new mutations, not copy the mutations from previous gens.
type LBMutations struct {
	parentMutations *LBMutations         // the previous generation's version of this LB
	mutn []Mutator		// holds deleterious, neutral, favorable, initial deleterious, initial favorable
}


// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB and the cumulative fitness affect on the individual's fitness.
type LinkageBlock2 struct {
	mutations *LBMutations		// holds the new mutations created in this generation, plus a pointer back to the parent's mutations
	//dMutn []*DeleteriousMutation
	//fMutn []*FavorableMutation

	// This is float32 to save space, and doesn't make any difference in the mean fitness in a typical run until the 11th decimal place. It saves approx 160MB for a 10,000 pop, plus the time for allocating and copying the extra mem
	delFitnessEffect       float32              // keep a running sum of the fitness so we can calc the LB fitness quickly.
	favFitnessEffect       float32

	numDeleterious         uint16
	numFavorable           uint16

	//nMutn                  []*NeutralMutation
	numNeutrals            uint16               // this is used instead of the array above if track_neutrals==false

	//dAllele                []*DeleteriousAllele // initial alleles
	numDelAllele uint16
	//fAllele                []*FavorableAllele
	numFavAllele uint16
	alleleDelFitnessEffect float32              // keep a running sum of the fitness so we can calc the LB fitness quickly
	alleleFavFitnessEffect float32

	owner                   *Chromosome         // keep track of owner so we know whether we have to copy this LB or can just transfer ownership
}
*/

/*
// This version of the LB factory is used with the config.Cfg.Computation.Transfer_linkage_blocks option
func LinkageBlockFactory(owner *Chromosome) *LinkageBlock {
	// Initially there are no mutations.
	// Note: there is no need to allocate the mutation slices with backing arrays. That will happen automatically the 1st time they are
	//		appended to with append(). See https://blog.golang.org/go-slices-usage-and-internals
	return &LinkageBlock{owner: owner}
}


// LinkageBlockFactory creates an LB for an individual with a ptr back to the LBMutations object of the parent this LB was inherited from.
// It also starts with all of the cumulative stats from its parent's LB.
func LinkageBlockFactory2(owner *Chromosome, parentLB *LinkageBlock2) (lb *LinkageBlock2) {
	// Initially there are no mutations in this generation.
	// Note: there is no need to allocate the mutation slices with backing arrays. That will happen automatically the 1st time they are
	//		appended to with append(). See https://blog.golang.org/go-slices-usage-and-internals
	mutations := &LBMutations{}
	if parentLB != nil && parentLB.GetNumMutations() > 0 {
		// The logic of the 2nd half of this if is: the immediate parent has mutations or the parent points back to his parent (which could have mutations)
		//if parentLB.mutations != nil && (len(parentLB.mutations.mutn) > 0 || parentLB.mutations.parentMutations != nil) { mutations.parentMutations = parentLB.mutations }
		// else if there are no tracked mutations in the parent, don't bother pointing back to it
		mutations.parentMutations = parentLB.mutations 		// if num mutations of parent is non-zero, point back to mutation list
		lb = &LinkageBlock{
			mutations: mutations,
			delFitnessEffect: parentLB.delFitnessEffect,
			favFitnessEffect: parentLB.favFitnessEffect,
			numDeleterious: parentLB.numDeleterious,
			numFavorable: parentLB.numFavorable,
			numNeutrals: parentLB.numNeutrals,
			numDelAllele: parentLB.numDelAllele,
			numFavAllele: parentLB.numFavAllele,
			alleleDelFitnessEffect: parentLB.alleleDelFitnessEffect,
			alleleFavFitnessEffect: parentLB.alleleFavFitnessEffect,
			owner: owner,		// this is to support the LB transfer case
		}
	} else {
		lb = &LinkageBlock{
			mutations: mutations,
			owner: owner,		// this is to support the LB transfer case
		}
	}
	return
}
*/


// GetNumMutations returns the current total number of mutations and initial alleles
func (lb *LinkageBlock) GetNumMutations() uint32 {
	return uint32(lb.numDeleterious + lb.numFavorable + lb.numNeutrals + lb.numDelAllele + lb.numFavAllele)
}


/*
// Transfer will give the "to" chromosome (of the child) the equivalent LB, by just transferring ownership of this LB instance (if it has not already given it
// to another child), or by copying the LB if it must. The reason transfer of ownership to a child is ok is once this function is called, the "from" chromosome (the parent) will
// never do anything with this LB again, except maybe copy the contents to another child. From this perspective, it is also important that
// the children of these parents not do anything with their LBs until the parents are done creating all of their children.
func (lb *LinkageBlock) Transfer(from, to *Chromosome, lbIndex int) *LinkageBlock {
	//if lb.owner == from && config.Cfg.Computation.Transfer_linkage_blocks {
	if lb.owner == from {
		// "From" still owns this LB, so it is his to give away
		//config.Verbose(9, " Transferring ownership of LB %p from %p to %p", lb, from, to)
		to.LinkageBlocks[lbIndex] = lb
		to.NumMutations += lb.GetNumMutations()
		to.FitnessEffect += lb.SumFitness()
		lb.owner = to
		return lb
	} else {
		// This LB has already been given away to another offspring, so need to make a copy
		newLB := lb.Copy(to)
		to.LinkageBlocks[lbIndex] = newLB
		to.NumMutations += newLB.GetNumMutations()
		to.FitnessEffect += newLB.SumFitness()
		return newLB
	}
}


// Copy makes a semi-deep copy (makes a copy of the array of pointers to mutations, but does *not* copy the mutations themselves, because they are immutable) and returns it
func (lb *LinkageBlock) Copy(owner *Chromosome) *LinkageBlock {
	newLb := LinkageBlockFactory(owner)
	// Assigning a slice does not copy all the array elements, so we have to make that happen
	if len(lb.mutn) > 0 {
		newLb.mutn = make([]Mutation, len(lb.mutn)) 	// allocate a new underlying array the same length as the source
		//newLb.mutn = make([]Mutation, len(lb.mutn), len(lb.mutn)+2) 	//todo: consider making the new LB with a capacity a little bigger (mutation_rate / num_LBs) so it doesn't have to be copied as soon as a mutation is added
		copy(newLb.mutn, lb.mutn)        // this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)
	}
	//if len(lb.mutations.mutn) > 0 {
	//	newLb.mutations.mutn = make([]Mutator, len(lb.mutations.mutn)) 	// allocate a new underlying array the same length as the source
	//	copy(newLb.mutations.mutn, lb.mutations.mutn)        // this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)
	//}

	newLb.fitnessEffect = lb.fitnessEffect

	newLb.numDeleterious = lb.numDeleterious
	//newLb.delFitnessEffect = lb.delFitnessEffect

	newLb.numNeutrals = lb.numNeutrals

	newLb.numFavorable = lb.numFavorable
	//newLb.favFitnessEffect = lb.favFitnessEffect

	newLb.numDelAllele = lb.numDelAllele
	//newLb.alleleDelFitnessEffect = lb.alleleDelFitnessEffect
	newLb.numFavAllele = lb.numFavAllele
	//newLb.alleleFavFitnessEffect = lb.alleleFavFitnessEffect

	return newLb
}
*/


// AppendMutation creates and adds a mutation to this LB.
// Note: The implementation of golang's append() appears to be that if it has to copy the array is doubles the capacity, which is probably what we want for the Mutation arrays.
func (lb *LinkageBlock) AppendMutation(mutId uint64, uniformRandom *rand.Rand) (fitnessEffect float32) {
	mType := CalcMutationType(uniformRandom)
	switch mType {
	case DELETERIOUS:
		fitnessEffect = calcDelMutationAttrs(uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || fitnessEffect < -config.Cfg.Computation.Tracking_threshold {
			// We are tracking this mutation, so create it and append
			//mutn := DeleteriousMutationFactory(fitnessEffect, uniformRandom)
			//lb.mutations.mutn = append(lb.mutations.mutn, mutn)
			lb.mutn = append(lb.mutn, Mutation{Id: mutId, Type: DELETERIOUS})
		}
		lb.numDeleterious++
		//lb.delFitnessEffect += fitnessEffect		// currently only the additive combination model is supported, so this is appropriate
		lb.fitnessEffect += fitnessEffect		// currently only the additive combination model is supported, so this is appropriate
	case NEUTRAL:
		if config.Cfg.Computation.Track_neutrals {
			//lb.mutations.mutn = append(lb.mutations.mutn, NeutralMutationFactory(uniformRandom))
			lb.mutn = append(lb.mutn, Mutation{Id: mutId, Type: NEUTRAL})
		}
		lb.numNeutrals++
	case FAVORABLE:
		fitnessEffect = calcFavMutationAttrs(uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || fitnessEffect > config.Cfg.Computation.Tracking_threshold {
			// We are tracking this mutation, so create it and append
			//mutn := FavorableMutationFactory(fitnessEffect, uniformRandom)
			//lb.mutations.mutn = append(lb.mutations.mutn, mutn)
			lb.mutn = append(lb.mutn, Mutation{Id: mutId, Type: FAVORABLE})
		}
		lb.numFavorable++
		//lb.favFitnessEffect += fitnessEffect	// currently only the additive combination model is supported, so this is appropriate
		lb.fitnessEffect += fitnessEffect	// currently only the additive combination model is supported, so this is appropriate
	}
	return
}


// AppendInitialContrastingAlleles adds an initial contrasting allele pair to 2 LBs (favorable to 1, deleterious to the other).
// The 2 LBs passed in are typically the same LB position on the same chromosome number, 1 from each parent.
func AppendInitialContrastingAlleles(lb1, lb2 *LinkageBlock, uniqueInt *utils.UniqueInt, uniformRandom *rand.Rand) (fitnessEffect1, fitnessEffect2 float32) {
	// Note: for now we assume that all initial contrasting alleles are co-dominant so that in the homozygous case (2 of the same favorable
	//		allele (or 2 of the deleterious allele) - 1 from each parent), the combined fitness effect is 1.0 * the allele fitness.
	expression := 0.5
	fitnessEffect := Mdl.CalcAlleleFitness(uniformRandom) * expression

	// Add a favorable allele to the 1st LB
	// Note: we assume that if initial alleles are being created, they are being tracked
	//favAllele := FavorableAlleleFactory(fitnessEffect)
	//lb1.mutations.mutn = append(lb1.mutations.mutn, favAllele)
	//lb1.alleleFavFitnessEffect += favAllele.GetFitnessEffect()
	lb1.mutn = append(lb1.mutn, Mutation{Id: uniqueInt.NextInt(), Type: FAV_ALLELE})
	lb1.numFavAllele++
	//lb1.alleleFavFitnessEffect += float32(fitnessEffect)
	fitnessEffect1 = float32(fitnessEffect)
	lb1.fitnessEffect += fitnessEffect1

	// Add a deleterious allele to the 2nd LB
	//delAllele := DeleteriousAlleleFactory(-fitnessEffect)
	//lb2.mutations.mutn = append(lb2.mutations.mutn, delAllele)
	//lb2.alleleDelFitnessEffect += delAllele.GetFitnessEffect()
	lb2.mutn = append(lb2.mutn, Mutation{Id: uniqueInt.NextInt()+1, Type: DEL_ALLELE})
	lb2.numDelAllele++
	//lb2.alleleDelFitnessEffect += float32(-fitnessEffect)
	fitnessEffect2 = float32(-fitnessEffect)
	lb2.fitnessEffect += fitnessEffect2
	return
}


// SumFitness combines the fitness effect of all of its mutations in the additive method
func (lb *LinkageBlock) SumFitness() (fitness float32) {
	//fitness = float64(lb.delFitnessEffect + lb.favFitnessEffect + lb.alleleDelFitnessEffect + lb.alleleFavFitnessEffect)
	//fitness = float64(lb.fitnessEffect)
	fitness = lb.fitnessEffect
	return
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the mean fitness factor of each.
// Note: the mean fitnesses take into account whether or not the mutation is expressed, so even for fixed mutation fitness the mean will not be that value.
func (lb *LinkageBlock) GetMutationStats() (deleterious, neutral, favorable uint32 /*, avDelFit, avFavFit float64*/ ) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(lb.numDeleterious)
	//avDelFit = float64(lb.delFitnessEffect)
	//if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) } 		// else avDelFit is already 0.0

	//neutral = uint32(len(lb.nMutn)) + uint32(lb.numNeutrals)
	neutral = uint32(lb.numNeutrals)

	favorable = uint32(lb.numFavorable)
	//avFavFit = float64(lb.favFitnessEffect)
	//if favorable > 0 { avFavFit = avFavFit / float64(favorable) } 		// else avFavFit is already 0.0
	return
}


// GetInitialAlleleStats returns the number of deleterious, neutral, favorable initial alleles, and the mean fitness factor of each.
func (lb *LinkageBlock) GetInitialAlleleStats() (deleterious, neutral, favorable uint32 /*, avDelFit, avFavFit float64*/ ) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(lb.numDelAllele)
	//if deleterious > 0 { avDelFit = float64(lb.alleleDelFitnessEffect) / float64(deleterious) } 		// else avDelFit is already 0.0

	neutral = 0

	favorable = uint32(lb.numFavAllele)
	//if favorable > 0 { avFavFit = float64(lb.alleleFavFitnessEffect) / float64(favorable) } 		// else avFavFit is already 0.0
	return
}


// CountAlleles counts all of this LB's alleles (both mutations and initial alleles) and adds them to the given struct
func (lb *LinkageBlock) CountAlleles(allelesForThisIndiv *AlleleCount) {
	// We are getting the alleles for just this individual so we don't want to double count the same allele from both parents,
	// so we only ever set the value to 1 for a particular allele id.
	//config.Verbose(1, "  LB owner=%p", lb.owner)
	for _, m := range lb.mutn {
		id := m.Id
		//config.Verbose(1, "    m.Id=%v, m.Type=%v", id, m.Type)
		switch m.Type {
		case DELETERIOUS:
			allelesForThisIndiv.Deleterious[id] = 1
		case NEUTRAL:
			allelesForThisIndiv.Neutral[id] = 1
		case FAVORABLE:
			allelesForThisIndiv.Favorable[id] = 1
		case DEL_ALLELE:
			allelesForThisIndiv.DelInitialAlleles[id] = 1
		case FAV_ALLELE:
			allelesForThisIndiv.FavInitialAlleles[id] = 1
		default:
			log.Fatalf("Error: unknown Mutation type %v found when counting alleles.", m.Type)
		}
	}

	/*
	mutns := lb.mutations
	for mutns != nil {		// we need to follow the chain of LBMutations objects back thru all of his ancestors
		for _, m := range mutns.mutn {
			// Use the ptr to the mutation object as the key in the map.
			//id := uintptr(unsafe.Pointer(m))
			id := m.GetPointer()
			switch m.(type) {
			case *DeleteriousMutation:
				allelesForThisIndiv.Deleterious[id] = 1
			case *NeutralMutation:
				allelesForThisIndiv.Neutral[id] = 1
			case *FavorableMutation:
				allelesForThisIndiv.Favorable[id] = 1
			case *DeleteriousAllele:
				allelesForThisIndiv.DelInitialAlleles[id] = 1
			case *FavorableAllele:
				allelesForThisIndiv.FavInitialAlleles[id] = 1
			default:
				log.Fatalln("Error: unknown Mutator type found when counting alleles.")
			}
		}
		mutns = mutns.parentMutations

		for _, m := range mutns.dMutn {
			// Use the ptr to the mutation object as the key in the map.
			id := uintptr(unsafe.Pointer(m))
			allelesForThisIndiv.Deleterious[id] = 1
		}
		for _, m := range mutns.nMutn {
			id := uintptr(unsafe.Pointer(m))
			allelesForThisIndiv.Neutral[id] = 1
		}
		for _, m := range mutns.fMutn {
			id := uintptr(unsafe.Pointer(m))
			allelesForThisIndiv.Favorable[id] = 1
		}
		for _, a := range mutns.dAllele {
			id := uintptr(unsafe.Pointer(a))
			allelesForThisIndiv.DelInitialAlleles[id] = 1
		}
		for _, a := range mutns.fAllele {
			id := uintptr(unsafe.Pointer(a))
			allelesForThisIndiv.FavInitialAlleles[id] = 1
		}
	}
	*/
}
