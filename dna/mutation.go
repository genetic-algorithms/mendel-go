package dna

type MutationType uint8
const (
	DELETERIOUS MutationType = iota
	NEUTRAL MutationType = iota
	FAVORABLE MutationType = iota
)

// Mutation is the base class for all mutation types. It represents 1 mutation in 1 individual.
type Mutation struct {
	Mtype MutationType
	Dominant bool 		// dominant or recessive mutation
}

//func MutationFactory(mtype MutationType) *Mutation {
//	return &Mutation{Mtype: mtype}
//}

// DeleteriousMutation represents 1 deleterious mutation in 1 individual.
type DeleteriousMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func DeleteriousMutationFactory(dominant bool) *DeleteriousMutation {
	return &DeleteriousMutation{Mutation{Mtype: DELETERIOUS, Dominant: dominant}}
}

// NeutralMutation represents 1 neutral mutation in 1 individual.
type NeutralMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func NeutralMutationFactory(dominant bool) *NeutralMutation {
	return &NeutralMutation{Mutation{Mtype: NEUTRAL, Dominant: dominant}}
}

// FavorableMutation represents 1 favorable mutation in 1 individual.
type FavorableMutation struct {
	Mutation 		// this anonymous reference includes the base class's struct here
}

func FavorableMutationFactory(dominant bool) *FavorableMutation {
	return &FavorableMutation{Mutation{Mtype: FAVORABLE, Dominant: dominant}}
}
