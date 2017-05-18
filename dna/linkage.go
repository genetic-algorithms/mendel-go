package dna

import (
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/config"
)

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB
// and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	//Fitness float64
	//Mutn []*Mutation
	DMutn []*DeleteriousMutation
	NMutn []*NeutralMutation
	NumNeutrals uint16		// this is used instead of the array above if track_neutrals==false
	FMutn []*FavorableMutation
}


func LinkageBlockFactory() *LinkageBlock {
	// Initially there are no mutations.
	// We do not need to initialize the mutn slices, go automatically makes them ready for append()
	return &LinkageBlock{}
}


// Copy makes a semi-deep copy (copies the array of pointers to mutations, but not the mutations themselves) and returns it
func (lb *LinkageBlock) Copy() *LinkageBlock {
	newLb := LinkageBlockFactory()
	// Assigning a slice does not copy all the array elements, so we have to make that happen
	newLb.DMutn = make([]*DeleteriousMutation, len(lb.DMutn)) 	// allocate a new underlying array the same length as the source
	if len(lb.DMutn) > 0 { copy(newLb.DMutn, lb.DMutn) } 		// this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)

	newLb.NMutn = make([]*NeutralMutation, len(lb.NMutn))
	if len(lb.NMutn) > 0 { copy(newLb.NMutn, lb.NMutn) }
	newLb.NumNeutrals = lb.NumNeutrals
	//if len(lb.NMutn) > 0 || lb.NumNeutrals > 0 { config.Verbose(3, "inheriting %d neutral mutations and %d num neutral", len(lb.NMutn), lb.NumNeutrals) }

	newLb.FMutn = make([]*FavorableMutation, len(lb.FMutn))
	if len(lb.FMutn) > 0 { copy(newLb.FMutn, lb.FMutn) }
	return newLb
}


// GetTotalMutnCount returns the number of mutations currently on this LB
func (lb *LinkageBlock) GetTotalMutnCount() uint32 {
	numNeuts := lb.NumNeutrals
	if numNeuts == 0 { numNeuts = uint16(len(lb.NMutn)) }		// maybe we are tracking neutrals
	return uint32(len(lb.DMutn)+int(numNeuts)+len(lb.FMutn))
}


// AppendMutation creates and adds a mutations to this LB
func (lb *LinkageBlock) AppendMutation(uniformRandom *rand.Rand) {
	mType := CalcMutationType(uniformRandom)
	switch mType {
	case DELETERIOUS:
		lb.DMutn = append(lb.DMutn, DeleteriousMutationFactory(uniformRandom))
	case NEUTRAL:
		if config.Cfg.Computation.Track_neutrals {
			//config.Verbose(3, "adding a neutral mutation")
			lb.NMutn = append(lb.NMutn, NeutralMutationFactory(uniformRandom))
		} else {
			//config.Verbose(3, "adding to the neutral mutation count")
			lb.NumNeutrals++
		}
	case FAVORABLE:
		lb.FMutn = append(lb.FMutn, FavorableMutationFactory(uniformRandom))
	}
}


/* Not using this, so we can apply the fitness factor aggregation at the individual level, using the appropriate model...
func (lb *LinkageBlock) GetFitness() (fitness float64) {
	fitness = 0.0
	for _, m := range lb.Mutn {
		if (m.GetExpressed()) { fitness += m.GetFitnessFactor() }
	}
	return
}
*/


// GetMutationStats returns the number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (lb *LinkageBlock) GetMutationStats() (deleterious, neutral, favorable uint32, avDelFit, avFavFit float64) {
	deleterious = uint32(len(lb.DMutn))
	for _, m := range lb.DMutn { avDelFit += m.GetFitnessEffect() }
	if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) } 		// else avDelFit is already 0.0

	neutral = uint32(lb.NumNeutrals)
	if neutral == 0 { neutral = uint32(len(lb.NMutn)) }		// maybe we are tracking neutrals

	favorable = uint32(len(lb.FMutn))
	for _, m := range lb.FMutn { avFavFit += m.GetFitnessEffect() }
	if favorable > 0 { avFavFit = avFavFit / float64(favorable) } 		// else avFavFit is already 0.0
	return
}
