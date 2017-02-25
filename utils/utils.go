package utils

import (
	"fmt"
	"log"
	//"os"
	//"path/filepath"
	//"io/ioutil"
	"bitbucket.org/geneticentropy/mendel-go/config"
)

func Verbose(level uint, msg string, args ...interface{}) {
	if level >= config.Cfg.General.Verbose { log.Printf("Verbose "+fmt.Sprint(level)+": "+msg, args...) }
}
