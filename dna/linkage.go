package dna

import (
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/config"
	//"log"
	//"unsafe"
	"github.com/genetic-algorithms/mendel-go/utils"
	"log"
)

// Note: with a typical 10K population (30K during mating) and 989 LBs per individual there are a lot of LBs, so saving
//		space in them is important.

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	mutn []Mutation		// holds deleterious, neutral, favorable, initial deleterious, initial favorable
	// Note: instead of adding the space of another LB member var, we could always make sure the mutn array is barely big enough so the builtin append() would naturally copy it
	IsPtrToParent bool		// whether or not the mutn slice is still a reference to its parents mutn array. We don't copy it until we add a mutation. During create of a new LB, this will naturally be set to false.
	fitnessEffect float32
	numDeleterious         uint16
	numFavorable           uint16
	numNeutrals            uint16               // this is used instead of the array above if track_neutrals==false
	numDelAllele uint16
	numFavAllele uint16
}


// GetNumMutations returns the current total number of mutations and initial alleles
func (lb *LinkageBlock) GetNumMutations() uint32 {
	return uint32(lb.numDeleterious + lb.numFavorable + lb.numNeutrals + lb.numDelAllele + lb.numFavAllele)
}


// AppendMutation creates and adds a mutation to this LB.
func (lb *LinkageBlock) AppendMutation(mutId uint64, uniformRandom *rand.Rand) (mType MutationType, fitnessEffect float32) {
	mType = CalcMutationType(uniformRandom)
	switch mType {
	case DELETERIOUS_DOMINANT:
		fallthrough
	case DELETERIOUS_RECESSIVE:
		fitnessEffect = calcDelMutationAttrs(mType, uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || fitnessEffect < -config.Cfg.Computation.Tracking_threshold {
			// We are tracking this mutation, so create it and append
			lb.appendMutn(Mutation{Id: mutId, Type: mType, FitnessEffect: fitnessEffect})
		}
		lb.numDeleterious++
		lb.fitnessEffect += fitnessEffect		// currently only the additive combination model is supported, so this is appropriate
	case NEUTRAL:
		if config.Cfg.Computation.Track_neutrals {
			lb.appendMutn(Mutation{Id: mutId, Type: NEUTRAL})
		}
		lb.numNeutrals++
	case FAVORABLE_DOMINANT:
		fallthrough
	case FAVORABLE_RECESSIVE:
		fitnessEffect = calcFavMutationAttrs(mType, uniformRandom)
		if config.Cfg.Computation.Tracking_threshold == 0.0 || fitnessEffect > config.Cfg.Computation.Tracking_threshold {
			// We are tracking this mutation, so create it and append
			lb.appendMutn(Mutation{Id: mutId, Type: mType, FitnessEffect: fitnessEffect})
		}
		lb.numFavorable++
		lb.fitnessEffect += fitnessEffect	// currently only the additive combination model is supported, so this is appropriate
	}
	return
}


// appendMutn adds a mutation to the LB slice, but only adds 2 elements (instead of Go's default of doubling) if it needs to be made bigger
// because for typical input parameters usually 0 or 1 mutation gets added to an LB in a generation.
func (lb *LinkageBlock) appendMutn(mutn Mutation) {
	origLen := len(lb.mutn)
	newLen := origLen + 1
	// If the mutn backing array is already ours but not big enough, or if we are still referring to our parents array, make a new/bigger array
	//if newLen > cap(lb.mutn) || (config.Cfg.Computation.Perf_option == 2 && lb.IsPtrToParent) {
	if newLen > cap(lb.mutn) || lb.IsPtrToParent {
		// current backing array is not big enough, so allocate a new one
		newCap := origLen + 2 	// make the capacity of the new backing array 2 bigger, in case we add another mutn later
		newSlice := make([]Mutation, newLen, newCap)
		copy(newSlice, lb.mutn)		// will only copy the number of elements slice has (the smaller one)
		newSlice[origLen] = mutn
		lb.mutn = newSlice
	} else {
		// current backing array is big enough
		lb.mutn = lb.mutn[0:newLen]		// increase the len of the slice
		lb.mutn[origLen] = mutn
	}
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
	fitnessEffect1 = float32(fitnessEffect)
	lb1.mutn = append(lb1.mutn, Mutation{Id: uniqueInt.NextInt(), Type: FAV_ALLELE, FitnessEffect: fitnessEffect1})
	lb1.numFavAllele++
	lb1.fitnessEffect += fitnessEffect1

	// Add a deleterious allele to the 2nd LB
	fitnessEffect2 = float32(-fitnessEffect)
	lb2.mutn = append(lb2.mutn, Mutation{Id: uniqueInt.NextInt()+1, Type: DEL_ALLELE, FitnessEffect: fitnessEffect2})
	lb2.numDelAllele++
	lb2.fitnessEffect += fitnessEffect2
	return
}


// SumFitness combines the fitness effect of all of its mutations in the additive method
func (lb *LinkageBlock) SumFitness() (fitness float32) {
	fitness = lb.fitnessEffect
	return
}


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and deleterious and favorable initial alleles.
func (lb *LinkageBlock) GetMutationStats() (deleterious, neutral, favorable, delAllele, favAllele uint32) {
	// Note: this is only valid for the additive combination method
	deleterious = uint32(lb.numDeleterious)
	neutral = uint32(lb.numNeutrals)
	favorable = uint32(lb.numFavorable)
	delAllele = uint32(lb.numDelAllele)
	favAllele = uint32(lb.numFavAllele)
	return
}


// CountAlleles counts all of this LB's alleles (both mutations and initial alleles) and adds them to the given struct
func (lb *LinkageBlock) CountAlleles(allelesForThisIndiv *AlleleCount) {
	// We are getting the alleles for just this individual so we don't want to double count the same allele from both parents,
	// so we only ever set the value to 1 for a particular allele id.
	for _, m := range lb.mutn {
		id := m.Id
		switch m.Type {
		case DELETERIOUS_DOMINANT:
			if allele, ok := allelesForThisIndiv.DeleteriousDom[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		case DELETERIOUS_RECESSIVE:
			if allele, ok := allelesForThisIndiv.DeleteriousRec[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.DeleteriousRec[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.DeleteriousRec[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		case NEUTRAL:
			if allele, ok := allelesForThisIndiv.Neutral[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.Neutral[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.Neutral[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		case FAVORABLE_DOMINANT:
			if allele, ok := allelesForThisIndiv.FavorableDom[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.FavorableDom[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.FavorableDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		case FAVORABLE_RECESSIVE:
			if allele, ok := allelesForThisIndiv.FavorableRec[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.FavorableRec[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.FavorableRec[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		case DEL_ALLELE:
			if allele, ok := allelesForThisIndiv.DelInitialAlleles[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.DelInitialAlleles[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.DelInitialAlleles[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		case FAV_ALLELE:
			if allele, ok := allelesForThisIndiv.FavInitialAlleles[id]; ok {
				// It already exists, update it
				if config.Cfg.Computation.Count_duplicate_alleles {
					allelesForThisIndiv.FavInitialAlleles[id] = Allele{Count: allele.Count+1, FitnessEffect: m.FitnessEffect}
				}
				// else we already did this: allelesForThisIndiv.DeleteriousDom[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			} else {
				allelesForThisIndiv.FavInitialAlleles[id] = Allele{Count: 1, FitnessEffect: m.FitnessEffect}
			}
		default:
			log.Fatalf("Error: unknown Mutation type %v found when counting alleles.", m.Type)
		}
	}
}
