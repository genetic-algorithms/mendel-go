package dna

import "bitbucket.org/geneticentropy/mendel-go/config"

// Genome represents an individual's genome, but really just tracks the diffs from the common genome (mutations and alleles).
type Genome struct {
	Chromos []*Chromosome
	Linkages []*LinkageBlock
}

func GenomeFactory() *Genome {
	gen := &Genome{
		Chromos: make([]*Chromosome, config.Cfg.Population.Haploid_chromosome_number),
		Linkages: make([]*LinkageBlock, config.Cfg.Population.Num_linkage_subunits),
	}

	//todo: there is probably a faster way to initilize these arrays
	for i := range gen.Chromos { gen.Chromos[i] = ChromosomeFactory() }
	for i := range gen.Linkages { gen.Linkages[i] = LinkageBlockFactory() }

	return gen
}
