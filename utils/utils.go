package utils

import (
	"log"
	"io/ioutil"
	"math"
	"runtime"
	"encoding/hex"
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/random"
	"os"
	"archive/zip"
	"path/filepath"
	"strings"
	"io"
)

func CopyFile(fromFile, toFile string) error {
	log.Printf("Copying %v to %v...\n", fromFile, toFile) 	// can not use verbosity here because we have not read the config file yet
	buf, err := ioutil.ReadFile(fromFile)
	if err != nil { return err }
	if err := ioutil.WriteFile(toFile, buf, 0644); err != nil { return err }
	return nil
}

func CopyFromFileName2Writer(fromFileName string, toFile *os.File) error {
	buf, err := ioutil.ReadFile(fromFileName)
	if err != nil { return err }
	if _, err := io.WriteString(toFile, string(buf)); err != nil { return err }
	return nil
}

func CanonicalPathsEqual(filePath1, filePath2 string) (isEqual bool, err error) {
	var cPath1, cPath2 string
	if cPath1, err = filepath.EvalSymlinks(filePath1); err != nil { return }
	if cPath1, err = filepath.Abs(cPath1); err != nil { return }
	if cPath2, err = filepath.EvalSymlinks(filePath2); err != nil { return }
	if cPath2, err = filepath.Abs(cPath2); err != nil { return }
	isEqual = cPath1 == cPath2
	return
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

func MaxUint64(a, b uint64) uint64 {
	if a > b { return a }
	return b
}

func MinFloat64(a, b float64) float64 {
	if a < b { return a }
	return b
}

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


// CollectGarbage runs the Go GC and times it
func CollectGarbage() {
	Measure.Start("GC")
	runtime.GC()
	Measure.Stop("GC")
}


// RandomSlug returns a random 6 char id
func RandomSlug() string {
	b := make([]byte, 3) // equals 6 characters
	//rand.Read(b)
	rand.New(rand.NewSource(random.GetSeed())).Read(b)		// GetSeed() uses a truly random seed
	return hex.EncodeToString(b)
}


/*
 CreatePrefixedZip creates a zip file of a directory (pathToZip). As it puts each file in the zip,
 if the path of the file starts with prefixToReplace it will be changed to newPrefix.
 Note: this is not a full-featured zip creation function, only what we need. Specifically, it doesn't
		follow sym links during its recursion (because filepath.Walk() doesn't), and it doesn't add the
		dirs as their own entries like the zip cmd does (SPC can't handle that).
*/
func CreatePrefixedZip(pathToZipUp, zipFilePath, prefixToReplace, newPrefix string) error {
	cleanPrefixToReplace := filepath.Clean(prefixToReplace)
	//fmt.Printf("CreatePrefixedZip: pathToZipUp=%s, zipFilePath=%s, prefixToReplace=%s, cleanPrefixToReplace=%s, newPrefix=%s\n", pathToZipUp, zipFilePath, prefixToReplace, cleanPrefixToReplace, newPrefix)
	zipFile, err := os.Create(zipFilePath)
	if err != nil {	return err }
	myZip := zip.NewWriter(zipFile)
	err = filepath.Walk(pathToZipUp, func(filePath string, info os.FileInfo, err error) error {
		if err != nil { return err }	// Walk() had a problem with this file
		if info.IsDir() { return nil }	// SPC can't handle the dirs themselves in the zip
		if err != nil { return err }
		var zipFile string
		if strings.HasPrefix(filePath, cleanPrefixToReplace) {
			zipFile = newPrefix + strings.TrimPrefix(filePath, cleanPrefixToReplace)
		} else {
			zipFile = filePath
		}
		//fmt.Printf("Walk: changing %s to %s and adding it...\n", filePath, zipFile)
		zipFileWriter, err := myZip.Create(zipFile)
		if err != nil { return err }
		fsFile, err := os.Open(filePath)
		if err != nil { return err }
		_, err = io.Copy(zipFileWriter, fsFile)
		if err != nil { return err }
		return nil
	})
	if err != nil {	return err }
	err = myZip.Close()
	if err != nil {	return err }
	return nil
}


func NotImplementedYet(what string) { log.Fatalf("Not implemented yet: %v", what) }
