package config

import (
	"os"
	"regexp"
	"log"
)

// Supported file names. Do we need to make this a literal map to be able to check inputted file names??
const (
	HISTORY_FILENAME string = "mendel.hst"
)

// Not using buffered io because we need write to be flushed every generation to support restart
//type FileElem struct {
//	File *os.File
//	Writer *bufio.Writer
//}

// FileMgr is a simple object to manage all of the data files mendel writes.
type FileMgr struct {
	Files map[string]*os.File 		// key is filename, value is file descriptor (nil if not opened yet)
}

// FMgr is the singleton instance of FileMgr, created by FileMgrFactory.
var FMgr *FileMgr


// FileMgrFactory creates FMgr and initializes it. filesToOutput comes from the input file.
func FileMgrFactory(dataFilePath, filesToOutput string) {
	FMgr = &FileMgr{}

	// Open all of the files and put in the map
	fileNames := regexp.MustCompile(`,\s*`).Split(filesToOutput, -1)
	Verbose(5, "Opening files for writing: %v", fileNames)
	if len(fileNames) > 0 {
		// Make sure output directory exists
		if err := os.MkdirAll(dataFilePath, 0755); err != nil { log.Fatalf("Error creating data_file_path %v: %v", dataFilePath, err) }
	}
	FMgr.Files = make(map[string]*os.File)
	for _, f := range fileNames {
		//todo: check the input file name against the const/enum at the top of this file
		filePath := dataFilePath + "/" + f
		//file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0664)
		file, err := os.Create(filePath)
		if err != nil { log.Fatal(err) } 	// for now, if we can't open a file, just bail
		//FMgr.Files[f] = FileElem{file, bufio.NewWriter(file)}
		FMgr.Files[f] = file
	}
}


// GetFile returns the specified file descriptor if we have it open.
func (fMgr *FileMgr) GetFile(fileName string) *os.File {
	if file, ok := fMgr.Files[fileName]; ok && file != nil { return file }
	return nil
}
//func (fMgr *FileMgr) GetFileWriter(fileName string) *bufio.Writer {
//	if file, ok := fMgr.Files[fileName]; ok && file.Writer != nil { return file.Writer }
//	return nil
//}


// CloseFiles closes all of the open files.
func (fMgr *FileMgr) CloseFiles() {
	for name, file := range fMgr.Files {
		if file != nil {
			if err := file.Close(); err != nil { log.Printf("Error closing %v: %v", name, err) }
		}
	}
}
