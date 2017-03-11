package dna

// Genome represents an individual's genome, but really just tracks the diffs from the common genome (mutations and alleles).
type Genome struct {

}

func GenomeFactory() *Genome {
	return &Genome{}
}
