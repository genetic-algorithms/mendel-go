package config

import (
	//"fmt"
	"os"
	//"path/filepath"
	"github.com/naoina/toml"
	"io/ioutil"
	//"bitbucket.org/geneticentropy/mendel-go/utils"
	"log"
)

type Config struct {
	Name string
	General struct {
		Verbose uint
	}
}

var Cfg *Config 		// singleton config

func (c *Config) ReadFromFile(filename string) error {
	log.Printf("Reading %v...\n", filename) 	// can not use Verbose here because we have not read the config file yet
	f, err := os.Open(filename)
	if err != nil { return err }
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil { return err }
	if err := toml.Unmarshal(buf, c); err != nil { return err }
	Cfg = c 		// set the singleton config
	return nil
}