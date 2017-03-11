package pop

import "bitbucket.org/geneticentropy/mendel-go/utils"

// Population tracks the tribes and global info about the population.
type Population struct {

}

func PopulationFactory() *Population {
	return &Population{}
}

func (p *Population) Mate() {
	utils.Verbose(9, "Mating the population...\n")
}

func (p *Population) Select() {
	utils.Verbose(9, "Selecting the population...\n")
}
