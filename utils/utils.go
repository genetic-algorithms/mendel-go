package utils

import (
	"log"
	"io/ioutil"
)

func CopyFile(fromFile, toFile string) error {
	log.Printf("Copying %v to %v...\n", fromFile, toFile) 	// can not use verbosity here because we have not read the config file yet
	buf, err := ioutil.ReadFile(fromFile)
	if err != nil { return err }
	if err := ioutil.WriteFile(toFile, buf, 0644); err != nil { return err }
	return nil
}

func Min(a, b int) int {
	if a < b { return a }
	return b
}

func Max(a, b int) int {
	if a > b { return a }
	return b
}

func NotImplementedYet(what string) { log.Fatalf("Not implemented yet: %v", what) }
