package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/dna"
)

// Individual represents 1 organism in the population, tracking its mutations and alleles.
type Individual struct {
	Gen *dna.Genome
}

func IndividualFactory() *Individual{
	return &Individual{Gen: dna.GenomeFactory()}
}