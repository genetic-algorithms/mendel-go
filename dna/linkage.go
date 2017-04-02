package dna

// LinkageBlock represents 1 linkage block in the genome of an individual. It tracks the mutations in this LB
// and the cumulative fitness affect on the individual's fitness.
type LinkageBlock struct {
	//Fitness float64
	Mutn []*Mutation
	//Dmutn []*DeleteriousMutation  <-- not sure if we need to list these separately
	//Nmutn []*NeutralMutation
	//Fmutn []*FavorableMutation
}


func LinkageBlockFactory() *LinkageBlock {
	// Initially there are no mutations??
	return &LinkageBlock{}
}


// Copy makes a semi-deep copy (copies the array of pointers to mutations, but not the mutations themselves) and returns it
func (lb *LinkageBlock) Copy() *LinkageBlock {
	newLb := LinkageBlockFactory()
	// Assigning a slice does not copy all the array elements, so we have to make that happen
	newLb.Mutn = make([]*Mutation, len(lb.Mutn)) 	// allocate a new underlying array the same length as the source
	copy(newLb.Mutn, lb.Mutn) 		// this copies the array elements, which are ptrs to mutations, but it does not copy the mutations themselves (which are immutable, so we can reuse them)
	return newLb
}


// GetMutnCount returns the number of mutations currently on this LB
func (lb *LinkageBlock) GetMutnCount() int {
	return len(lb.Mutn)
}


// Append adds a mutation to this LB
func (lb *LinkageBlock) Append(mutn ...*Mutation) {
	lb.Mutn = append(lb.Mutn, mutn ...)
	//lb.calcFitness()
}


func (lb *LinkageBlock) GetFitness() (fitness float32) {
	//todo: the current implementation just sums all of the fitness factors
	fitness = 0.0
	for _, m := range lb.Mutn {
		if (m.GetExpressed()) { fitness += m.GetFitnessFactor() }
	}
	return
}


// GetNumMutations returns the number of deleterious, neutral, favorable mutations, respectively
func (lb *LinkageBlock) GetNumMutations() (deleterious, neutral, favorable int) {
	//todo: we maybe should store the different mutation types separately so we do not have to count them like this
	for _, m := range lb.Mutn {
		switch m.GetMType() {
		case DELETERIOUS:
			deleterious++
		case NEUTRAL:
			neutral++
		case FAVORABLE:
			favorable++
		}
	}
	return
}
