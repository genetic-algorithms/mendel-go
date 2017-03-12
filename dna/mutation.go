package dna

// Mutation represents 1 mutation in 1 individual.
type Mutation struct {

}

func MutationFactory() *Mutation {
	return &Mutation{}
}