package utils

import (
	"fmt"
	"log"
	//"os"
	//"path/filepath"
	//"io/ioutil"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"io/ioutil"
)

func Verbose(level int, msg string, args ...interface{}) {
	if level >= config.Cfg.Computation.Verbosity { log.Printf("V"+fmt.Sprint(level)+" "+msg, args...) }
}

func CopyFile(fromFile, toFile string) error {
	log.Printf("Copying %v to %v...\n", fromFile, toFile) 	// can not use verbosity here because we have not read the config file yet
	buf, err := ioutil.ReadFile(fromFile)
	if err != nil { return err }
	if err := ioutil.WriteFile(toFile, buf, 0644); err != nil { return err }
	return nil
}

