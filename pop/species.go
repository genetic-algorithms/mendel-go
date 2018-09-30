package pop

import (
	"github.com/genetic-algorithms/mendel-go/config"
	"github.com/genetic-algorithms/mendel-go/random"
	"math/rand"
)

// Species tracks all of the populations (tribes) and holds attributes common to the whole species.
type Species struct {
	Populations []*Population		// the tribes that make up this species
}

func SpeciesFactory() *Species {
	s := &Species{
		Populations: make([]*Population, config.Cfg.Tribes.Num_tribes),
	}
	return s
}

// Initialize inits the populations for gen 0
func (s *Species) Initialize(maxGenNum uint32, uniformRandom *rand.Rand) *Species {
	for i := range s.Populations {
		var newRandom *rand.Rand
		if i == 0 {
			// Let the 1st pop use the main uniformRandom generator so tribes=1 is the same as pre-tribes
			newRandom = uniformRandom
		} else {
			newRandom = random.RandFactory()
		}
		s.Populations[i] = PopulationFactory(nil, 0) 		// genesis population
		Mdl.GenerateInitialAlleles(s.Populations[i], newRandom)
		s.Populations[i].ReportInitial(maxGenNum)
	}
	return s 		// so we can chain calls
}

// GetNumPopulations returns the number of populations in this species
func (s *Species) GetNumPopulations() uint32 {
	return uint32(len(s.Populations))
}

// GetCurrentSize returns the sum of all of the pop sizes
func (s *Species) GetCurrentSize() (size uint32) {
	for _, p := range s.Populations {
		size += p.GetCurrentSize()
	}
	return
}

// GetNextGeneration prepares all of the populations for the next gen and returns them in a new Species object
func (parentS *Species) GetNextGeneration(gen uint32) (childrenS *Species) {
	random.NextSeed = config.Cfg.Computation.Random_number_seed + 1		// reset the seed to 1 above our initial seed, so when we call RandFactory() in Mate() for additional threads it will work like it did before
	childrenS = SpeciesFactory()
	for i := range parentS.Populations {
		childrenS.Populations[i] = PopulationFactory(parentS.Populations[i], gen)	// this creates the PopulationParts too
	}
	return
}

// Mate mates all of the populations
func (parentS *Species) Mate(childrenS *Species, uniformRandom *rand.Rand) {
	for i := range parentS.Populations {
		var newRandom *rand.Rand
		if i == 0 {
			// Let the 1st pop use the main uniformRandom generator so tribes=1 is the same as pre-tribes
			newRandom = uniformRandom
		} else {
			newRandom = random.RandFactory()
		}
		parentS.Populations[i].Mate(childrenS.Populations[i], newRandom)
	}
}

// Select does selection on all of the populations
func (s *Species) Select(uniformRandom *rand.Rand) {
	for i, p := range s.Populations {
		var newRandom *rand.Rand
		if i == 0 {
			// Let the 1st pop use the main uniformRandom generator so tribes=1 is the same as pre-tribes
			newRandom = uniformRandom
		} else {
			newRandom = random.RandFactory()
		}
		p.Select(newRandom)
	}
}

// ReportEachGen reports stats on each population
func (s *Species) ReportEachGen(gen uint32, lastGen bool) {
	for _, p := range s.Populations {
		p.ReportEachGen(gen, lastGen)
	}
}

// ReportEachGen reports stats on each population
func (s *Species) GetAverageFitness() (averageFitness float64) {
	if s.GetNumPopulations() == 0 { return }
	for _, p := range s.Populations {
		aveFit, _, _, _, _ := p.GetFitnessStats()
		averageFitness += aveFit
	}
	return averageFitness / float64(s.GetNumPopulations())
}

/* don't think this is useful...
func (parentS *Species) MoveToNextGeneration(childrenS *Species, gen uint32, lastGen bool) *Species {
	for i := range parentS.Populations {
		childrenS.Populations[i].ReportEachGen(gen, lastGen)
	}
	return childrenS
}
*/
