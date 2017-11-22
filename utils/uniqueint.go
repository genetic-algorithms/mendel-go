package utils

import (
	"log"
)

const MAXUINT64 = uint64(18446744073709551615)

// UniqueInt hands out unique integer numbers, with the ability to also give a range to another object
type UniqueInt struct {
	nextInt uint64		// the next int to hand out
	lastInt uint64		// the max int we can hand out
}

// GlobalUniqueInt is a singleton instance of UniqueInt to hand out integers. It is *not* thread safe.
var GlobalUniqueInt *UniqueInt


// UniqueIntFactory creates the global instance of UniqueInt
func GlobalUniqueIntFactory() {
	GlobalUniqueInt = &UniqueInt{lastInt: MAXUINT64}
}


// DonateRange create a new instance of UniqueInt, donating a range of int's from the existing object.
func (u *UniqueInt) DonateRange(rangeSize uint64) (newU *UniqueInt) {
	donatedNextInt := u.nextInt
	u.nextInt += rangeSize 		// move our own nextInt past this range so we do not use it
	if u.nextInt > u.lastInt {		// Note: really should at 1 to lastInt in this check, but if lastInt is the max uint64 value that makes it wrap
		log.Fatalf("Error: UniqueInt rangeSize %d requested at nextInt %d results in nextInt %d which exceeds lastInt %d", rangeSize, donatedNextInt, u.nextInt, u.lastInt)
	}
	newU = &UniqueInt{nextInt: donatedNextInt, lastInt: donatedNextInt + rangeSize - 1}
	return
}


// Start starts the time measuring of a section of code. Returns the UniqueInt instance to Start and Stop can be chained in a defer statement.
func (u *UniqueInt) NextInt() uint64 {
	if u.nextInt > u.lastInt {
		//todo: for now don't kill a run because of a few duplicate id's
		log.Printf("Error: UniqueInt nextInt %d exceeds lastInt %d", u.nextInt, u.lastInt)
		return u.lastInt
	}
	i := u.nextInt
	u.nextInt++
	return i
}
