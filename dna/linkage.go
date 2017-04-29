package dna

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB
// and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	//Fitness float64
	//Mutn []*Mutation
	DMutn []*Mutation
	NMutn []*Mutation
	FMutn []*Mutation
}


func LinkageBlockFactory() *LinkageBlock {
	// Initially there are no mutations??
	// We do not need to initialize the mutn slices, go automatically makes them ready for append()
	return &LinkageBlock{}
}


// Copy makes a semi-deep copy (copies the array of pointers to mutations, but not the mutations themselves) and returns it
func (lb *LinkageBlock) Copy() *LinkageBlock {
	newLb := LinkageBlockFactory()
	// Assigning a slice does not copy all the array elements, so we have to make that happen
	newLb.DMutn = make([]*Mutation, len(lb.DMutn)) 	// allocate a new underlying array the same length as the source
	if len(lb.DMutn) > 0 { copy(newLb.DMutn, lb.DMutn) } 		// this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)
	newLb.NMutn = make([]*Mutation, len(lb.NMutn))
	if len(lb.NMutn) > 0 { copy(newLb.NMutn, lb.NMutn) }
	newLb.FMutn = make([]*Mutation, len(lb.FMutn))
	if len(lb.FMutn) > 0 { copy(newLb.FMutn, lb.FMutn) }
	return newLb
}


// GetMutnCount returns the number of mutations currently on this LB
func (lb *LinkageBlock) GetTotalMutnCount() uint32 {
	return uint32(len(lb.DMutn)+len(lb.NMutn)+len(lb.FMutn))
}


// Append adds a mutations to this LB
func (lb *LinkageBlock) Append(mutn ...*Mutation) {
	for _, m := range mutn {
		switch m.GetMType() {
		case DELETERIOUS:
			lb.DMutn = append(lb.DMutn, m)
		case NEUTRAL:
			lb.NMutn = append(lb.NMutn, m)
		case FAVORABLE:
			lb.FMutn = append(lb.FMutn, m)
		}
	}
	//lb.calcFitness()
}


/* Not using this, so we can apply the fitness factor aggregation at the individual level...
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
	for _, m := range lb.DMutn { avDelFit += m.GetFitnessFactor() }
	if deleterious > 0 { avDelFit = avDelFit / float64(deleterious) } 		// else avDelFit is already 0.0

	neutral = uint32(len(lb.NMutn))

	favorable = uint32(len(lb.FMutn))
	for _, m := range lb.FMutn { avFavFit += m.GetFitnessFactor() }
	if favorable > 0 { avFavFit = avFavFit / float64(favorable) } 		// else avFavFit is already 0.0
	return
}
