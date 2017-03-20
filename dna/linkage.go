package dna

// LinkageBlock represents 1 linkage block in the genome.
type LinkageBlock struct {
	Mutn_count int
	Fitness float64
	Dmutn []*DeleteriousMutation
	Nmutn []*NeutralMutation
	Fmutn []*FavorableMutation
}

func LinkageBlockFactory() *LinkageBlock {
	// Initially there are no mutations??
	return &LinkageBlock{}
}