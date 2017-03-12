package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"bitbucket.org/geneticentropy/mendel-go/config"
)

// Population tracks the tribes and global info about the population.
type Population struct {
	Size int
	Indivs []*Individual
}

func PopulationFactory() *Population {
	p := &Population{
		Size: config.Cfg.Basic.Pop_size,
		Indivs: make([]*Individual, config.Cfg.Basic.Pop_size),
	}

	//todo: there is probably a faster way to initilize these arrays
	for i := range p.Indivs { p.Indivs[i] = IndividualFactory() }

	return p
}

func (p *Population) Mate() {
	utils.Verbose(9, "Mating the population of %d individuals...\n", p.Size)
}

func (p *Population) Select() {
	utils.Verbose(9, "Selecting individuals to maintain a population of %d...\n", p.Size)
}
