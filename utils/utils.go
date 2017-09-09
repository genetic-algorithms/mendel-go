package utils

import (
	"log"
	"io/ioutil"
	"math"
)

func CopyFile(fromFile, toFile string) error {
	log.Printf("Copying %v to %v...\n", fromFile, toFile) 	// can not use verbosity here because we have not read the config file yet
	buf, err := ioutil.ReadFile(fromFile)
	if err != nil { return err }
	if err := ioutil.WriteFile(toFile, buf, 0644); err != nil { return err }
	return nil
}

func MinInt(a, b int) int {
	if a < b { return a }
	return b
}

/*
func MaxInt(a, b int) int {
	if a > b { return a }
	return b
}
*/

func MinUint32(a, b uint32) uint32 {
	if a < b { return a }
	return b
}

func MaxUint32(a, b uint32) uint32 {
	if a > b { return a }
	return b
}

/*
func MinFloat64(a, b float64) float64 {
	if a < b { return a }
	return b
}
*/

func MaxFloat64(a, b float64) float64 {
	if a > b { return a }
	return b
}


// RoundIntDiv divides a/b and returns the nearest int result.
func RoundIntDiv(a, b float64) int {
	return RoundInt(a / b)
}


// RoundInt returns the nearest int result. To get more exact, look at https://github.com/a-h/round
func RoundInt(f float64) int {
	if f < 0 {
		return int(math.Ceil(f - 0.5))
	}
	return int(math.Floor(f + 0.5))
}


// RoundToEven returns the nearest even integer. Note: only works for positive numbers
func RoundToEven(f float64) int {
	remainder := math.Mod(f, 2.0)
	switch {
	case remainder > 1.0:
		return int(math.Ceil(f))
	case remainder == 1.0:
		return int(f) + 1
	default: 	// remainder < 1.0
		return int(math.Floor(f))
	}
}

func NotImplementedYet(what string) { log.Fatalf("Not implemented yet: %v", what) }
