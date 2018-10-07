package utils

import (
	"log"
	"math"
)

// ResDivide splits a resource quantity among a number of recipient objects, handling the remainder for the last object.
type ResDivide struct {
	numObjects uint64
	perObject uint64		// number of resources each object gets (except for the last object)
	remainder uint64		// number of resources the last object gets
	nextIndex uint64		// the next object it should return an amount for (1-based)
}


// ResDivideFactory creates an instance of ResDivide
func ResDivideFactory(numResources, numObjects uint64) *ResDivide {
	perObject := uint64(math.Ceil(float64(numResources) / float64(numObjects)))
	remainder := numResources - ((numObjects-1) * perObject)
	return &ResDivide{numObjects: numObjects, perObject: perObject, remainder: remainder, nextIndex: 1}
}


// NextAmount returns the number of resources the next object in the iteration should receive
func (r *ResDivide) NextAmount() (nextAmount uint64) {
	if r.nextIndex > r.numObjects {
		log.Fatalf("Error: ResDivide nextIndex %d is > numObjects %d", r.nextIndex, r.numObjects)
	} else if r.nextIndex == r.numObjects {
		nextAmount = r.remainder
	} else {
		nextAmount = r.perObject
	}
	r.nextIndex++
	return
}
