package config

import (
	"os"
	"regexp"
	"log"
	"strings"
)

// Supported file names. Do we need to make this a literal map to be able to check inputted file names??
const (
	HISTORY_FILENAME string = "mendel.hst"
	FITNESS_FILENAME string = "mendel.fit"		// this one is faster to produce than mendel.hst
	//ALLELES_COUNT_FILENAME string = "alleles-count.json"
	ALLELE_BINS_DIRECTORY string = "allele-bins/"
	DELETERIOUS_CSV string = "deleterious.csv"
	NEUTRAL_CSV string = "neutral.csv"
	FAVORABLE_CSV string = "favorable.csv"
	DEL_ALLELE_CSV string = "del_allele.csv"
	FAV_ALLELE_CSV string = "fav_allele.csv"
)
// Apparently this can't be a const because a map literal isn't a const in go
var VALID_FILE_NAMES = map[string]int{HISTORY_FILENAME: 1, FITNESS_FILENAME: 1, ALLELE_BINS_DIRECTORY: 1,}

// Not using buffered io because we need write to be flushed every generation to support restart
//type FileElem struct {
//	File *os.File
//	Writer *bufio.Writer
//}

// FileMgr is a simple object to manage all of the data files mendel writes.
type FileMgr struct {
	dataFilePath string 		// the directory in which output files should go
	Files map[string]*os.File 		// key is filename, value is file descriptor (nil if not opened yet)
	Dirs map[string]map[string]*os.File			// directories that hold a group of output files
}

// FMgr is the singleton instance of FileMgr, created by FileMgrFactory.
var FMgr *FileMgr


// FileMgrFactory creates FMgr and initializes it. filesToOutput comes from the input file.
func FileMgrFactory(dataFilePath, filesToOutput string) *FileMgr {
	FMgr = &FileMgr{dataFilePath: dataFilePath, Files: make(map[string]*os.File), Dirs: make(map[string]map[string]*os.File) }
	if filesToOutput == "" { return FMgr }

	// Open all of the files and put in the map
	fileNames := regexp.MustCompile(`,\s*`).Split(filesToOutput, -1)
	Verbose(5, "Opening files for writing: %v", fileNames)
	if len(fileNames) > 0 {
		// Make sure output directory exists
		if err := os.MkdirAll(dataFilePath, 0755); err != nil { log.Fatalf("Error creating data_file_path %v: %v", dataFilePath, err) }
	}
	//FMgr.Files = make(map[string]*os.File)
	for _, f := range fileNames {
		// Check the input file name against our list of valid ones
		if _, ok := VALID_FILE_NAMES[f]; !ok { log.Fatalf("Error: %v specified in files_to_output is not an output file or directory name we support.", f) }

		if strings.HasSuffix(f, "/") {
			// f is really a directory name, make sure it exists and then add it to our Dirs map. The actual files under that will get created later when GetDirFile() is called.
			dirPath := dataFilePath + "/" + f
			if err := os.MkdirAll(dirPath, 0755); err != nil { log.Fatalf("Error creating output directory %v: %v", dirPath, err) }
			FMgr.Dirs[f] = make(map[string]*os.File)
		} else {
			// f is a single file, open it
			filePath := dataFilePath + "/" + f
			//file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0664)
			file, err := os.Create(filePath)
			if err != nil { log.Fatal(err) } 	// for now, if we can't open a file, just bail
			//FMgr.Files[f] = FileElem{file, bufio.NewWriter(file)}
			FMgr.Files[f] = file
		}
	}
	return FMgr		// return the object created so we can chain other methods after this
}


// IsFile returns true if the specified file name was specified in the files_to_output config parameter and is open.
func (fMgr *FileMgr) IsFile(fileName string) bool {
	if file, ok := fMgr.Files[fileName]; ok && file != nil { return true }
	return false
}


// IsDir returns true if the specified dir name was specified in the files_to_output config parameter.
func (fMgr *FileMgr) IsDir(dirName string) bool {
	if dir, ok := fMgr.Dirs[dirName]; ok && dir != nil { return true }
	return false
}


// GetFile returns the specified file descriptor if we have it open.
func (fMgr *FileMgr) GetFile(fileName string) *os.File {
	if file, ok := fMgr.Files[fileName]; ok && file != nil { return file }
	return nil
}


// GetDirFile returns the specified file descriptor in the specified directory, creating/opening the file if necessary.
func (fMgr *FileMgr) GetDirFile(dirName, fileName string) *os.File {
	if dir, ok := fMgr.Dirs[dirName]; ok {
		// We have this dir entry, look for the file entry within it
		if file, ok := dir[fileName]; ok && file != nil {
			return file
		} else {
			// Not there yet, create the entry
			filePath := FMgr.dataFilePath + "/" + dirName + fileName		// dirName already has / at the end of it
			file, err := os.Create(filePath)
			if err != nil { log.Fatal(err) } 	// for now, if we can't open a file, just bail
			dir[fileName] = file		// add it to our list so we can close it at the end
			return file
		}
	}
	return nil
}


/* Not currently used...
// GetFileBuffer returns a buffered file descriptor if we have it open
func (fMgr *FileMgr) GetFileBuffer(fileName string) *bufio.Writer {
	if file, ok := fMgr.Files[fileName]; ok && file.Writer != nil { return file.Writer }
	return nil
}
*/


// CloseAllFiles closes all of the open files.
func (fMgr *FileMgr) CloseAllFiles() {
	// Close all of the open files in our Files map
	for name, file := range fMgr.Files {
		if file != nil {
			if err := file.Close(); err != nil { log.Printf("Error closing %v: %v", name, err) }
		}
	}

	// Close all of the open files in our Dirs map
	for _, dir := range fMgr.Dirs {
		for fileName, file := range dir {
			if file != nil {
				if err := file.Close(); err != nil { log.Printf("Error closing %v: %v", fileName, err) }
			}
		}
	}
}
