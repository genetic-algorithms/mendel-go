package config

import (
	"os"
	"regexp"
	"log"
	"strings"
)

// Supported file names. Do we need to make this a literal map to be able to check inputted file names??
const (
	HISTORY_FILENAME = "mendel.hst"
	FITNESS_FILENAME = "mendel.fit"		// this one is faster to produce than mendel.hst
	TOML_FILENAME = "mendel_go.toml"		// the input parameters
	OUTPUT_FILENAME = "mendel_go.out"		//todo: figure out how we can get our own output into this file
	ALLELE_BINS_DIRECTORY = "allele-bins/"
	NORMALIZED_ALLELE_BINS_DIRECTORY = "normalized-allele-bins/"
	DISTRIBUTION_DEL_DIRECTORY = "allele-distribution-del/"
	DISTRIBUTION_FAV_DIRECTORY = "allele-distribution-fav/"
)

// Not using buffered io because we need write to be flushed every generation to support restart
//type FileElem struct {
//	File *os.File
//	Writer *bufio.Writer
//}

// FileMgr is a simple object to manage all of the data files mendel writes.
type FileMgr struct {
	DataFilePath string                         // the directory in which output files should go
	Files        map[string]*os.File            // key is filename, value is file descriptor (nil if not opened yet)
	Dirs         map[string]map[string]*os.File // directories that hold a group of output files. Key is dir name, value is map in which key is filename, value is file descriptor (nil if not opened yet)
}

// FMgr is the singleton instance of FileMgr, created by FileMgrFactory.
var FMgr *FileMgr


// FileMgrFactory creates FMgr and initializes it. filesToOutput comes from the input file.
func FileMgrFactory(dataFilePath, filesToOutput string) *FileMgr {
	FMgr = &FileMgr{DataFilePath: dataFilePath, Files: make(map[string]*os.File), Dirs: make(map[string]map[string]*os.File) }
	if filesToOutput == "" { return FMgr }

	// Get the proper list of file names
	var VALID_FILE_NAMES = map[string]int{HISTORY_FILENAME: 1, FITNESS_FILENAME: 1, ALLELE_BINS_DIRECTORY: 1, NORMALIZED_ALLELE_BINS_DIRECTORY: 1, DISTRIBUTION_DEL_DIRECTORY: 1, DISTRIBUTION_FAV_DIRECTORY: 1,}
	if CmdArgs.SPCusername != "" {
		// Add the files that are relevant only when -u is specified (and we aren't running in spc)
		VALID_FILE_NAMES[TOML_FILENAME] = 1
		VALID_FILE_NAMES[OUTPUT_FILENAME] = 1
	}
	var fileNames []string
	if filesToOutput == "*" {
		// They want all files/dirs output
		fileNames = make([]string, 0, len(VALID_FILE_NAMES))
		for k := range VALID_FILE_NAMES {
			fileNames = append(fileNames, k)
		}
	} else {
		// They gave us a list of file/dir names
		fileNames = regexp.MustCompile(`,\s*`).Split(filesToOutput, -1)
	}

	// Open all of the files and put in the map
	Verbose(5, "Opening files for writing: %v", fileNames)
	if len(fileNames) > 0 {
		// Make sure output directory exists
		if err := os.MkdirAll(dataFilePath, 0755); err != nil { log.Fatalf("Error creating data_file_path %v: %v", dataFilePath, err) }
	}
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
			filePath := FMgr.DataFilePath + "/" + dirName + fileName // dirName already has / at the end of it
			file, err := os.Create(filePath)
			if err != nil { log.Fatal(err) } 	// for now, if we can't open a file, just bail
			dir[fileName] = file		// add it to our list so we can close it at the end
			return file
		}
	}
	return nil
}


// CloseDirFile closes a file under a directory.
func (fMgr *FileMgr) CloseDirFile(dirName, fileName string) {
	if dir, ok := fMgr.Dirs[dirName]; ok {
		// We have this dir entry, look for the file entry within it
		if file, ok := dir[fileName]; ok && file != nil {
			if err := file.Close(); err != nil {
				log.Printf("Error closing %v: %v", fileName, err)
			} else {
				dir[fileName] = nil
			}
		} else {
			log.Printf("Error: file %v in directory %v can not be closed because it is not open", fileName, dirName)
		}
	} else {
		log.Printf("Error: file %v in directory %v can not be closed because %v output was never requested", fileName, dirName, dirName)
	}
}


// CloseFile closes a file under FileMgr control.
func (fMgr *FileMgr) CloseFile(fileName string) {
	if file, ok := fMgr.Files[fileName]; ok && file != nil {
		if err := file.Close(); err != nil {
			log.Printf("Error closing %v: %v", fileName, err)
		} else {
			fMgr.Files[fileName] = nil
		}
	} else {
		log.Printf("Error: file %v can not be closed because it is not open", fileName)
	}
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
	for fileName, file := range fMgr.Files {
		if file != nil {
			if err := file.Close(); err != nil { log.Printf("Error closing %v: %v", fileName, err) }
			fMgr.Files[fileName] = nil		// in case CloseAllFiles() is called a 2nd time
		}
	}

	// Close all of the open files in our Dirs map
	for _, dir := range fMgr.Dirs {
		for fileName, file := range dir {
			if file != nil {
				if err := file.Close(); err != nil { log.Printf("Error closing %v: %v", fileName, err) }
				dir[fileName] = nil		// in case CloseAllFiles() is called a 2nd time
			}
		}
	}
}
