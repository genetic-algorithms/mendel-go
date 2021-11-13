package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/genetic-algorithms/mendel-go/config"
)

const (
	IN_FILE_BASE  = "test/input/testcase"
	OUT_FILE_BASE = "test/output/testcase"
	EXP_FILE_BASE = "test/expected/testcase"
	//BIN_SUBDIR = "/allele-bins/"
	BIN_SUBDIR      = "/" + config.ALLELE_BINS_DIRECTORY
	NORM_SUBDIR     = "/" + config.NORMALIZED_ALLELE_BINS_DIRECTORY
	DIST_DEL_SUBDIR = "/" + config.DISTRIBUTION_DEL_DIRECTORY
	DIST_FAV_SUBDIR = "/" + config.DISTRIBUTION_FAV_DIRECTORY
)

// Typical small run
func TestMendelCase1(t *testing.T) {
	mendelCase(t, 1, 1)
}

// Same as TestMendelCase1 except that none of the mutations are tracked, but the results should be the same.
func TestMendelCase2(t *testing.T) {
	mendelCase(t, 2, 1)
}

// Same as TestMendelCase2 except crossover_model=partial and mean_num_crossovers=2
func TestMendelCase3(t *testing.T) {
	mendelCase(t, 3, 3)
}

// Same as TestMendelCase3 except with initial alleles
func TestMendelCase4(t *testing.T) {
	mendelCaseBin(t, 4, 4, "00000050.json", false, "", "")
}

// Same as TestMendelCase3 except with selection_model=ups, and heritability and non_scaling_noise back to default
func TestMendelCase5(t *testing.T) {
	mendelCase(t, 5, 5)
}

// Same as TestMendelCase5 except with selection_model=spps and omit_first_allele_bin=true
func TestMendelCase6(t *testing.T) {
	mendelCaseBin(t, 6, 6, "00000020.json", false, "", "")
}

// Same as TestMendelCase5 except with selection_model=partialtrunc
func TestMendelCase7(t *testing.T) {
	mendelCase(t, 7, 7)
}

// Same as TestMendelCase6 except with 4 threads
func TestMendelCase8(t *testing.T) {
	mendelCaseBin(t, 8, 8, "00000020.json", false, "", "")
}

// Same as TestMendelCase8 except with exponential pop growth
func TestMendelCase9(t *testing.T) {
	mendelCase(t, 9, 9)
}

// Same as TestMendelCase8 except with carrying capacity pop growth
func TestMendelCase10(t *testing.T) {
	mendelCaseBin(t, 10, 10, "00000010.json", false, "", "")
}

// Same as TestMendelCase8 except with founders pop growth with bottleneck, and weibull
func TestMendelCase11(t *testing.T) {
	mendelCase(t, 11, 11)
}

// Same as TestMendelCase3 except with initial alleles
func TestMendelCase12(t *testing.T) {
	mendelCaseBin(t, 12, 12, "00000100.json", true, "", "")
}

// Same as TestMendelCase11 except with multiple bottlenecks
func TestMendelCase13(t *testing.T) {
	mendelCase(t, 13, 13)
}

// Same as TestMendelCase12 except with multiple tribes
func TestMendelCase14(t *testing.T) {
	mendelCaseTribeBin(t, 14, 14, "00000050.json", true)
}

// Multiple tribes going extinct
/*todo: the results don't quite match the expected
func TestMendelCase15(t *testing.T) {
	mendelCaseTribeBin(t, 15, 15, "00000017.json", true)
}
*/

// Same as TestMendelCase4 except all initial alleles have 0 fitness effect
func TestMendelCase16(t *testing.T) {
	mendelCaseBin(t, 16, 16, "00000050.json", false, "", "")
}

// mendelCase runs a typical test case with an input file number and expected output file number.
func mendelCase(t *testing.T, num, expNum int) {
	numStr := strconv.Itoa(num)
	expNumStr := strconv.Itoa(expNum)
	//outputFileBase := "mendel"
	//testCase := "testcase" + strconv.Itoa(num)
	//expTestCase := "testcase" + strconv.Itoa(expNum)		// the number of the expected output file
	//outFileDir := "test/output/" + testCase
	inFileName := IN_FILE_BASE + numStr + ".ini"
	dataPath := OUT_FILE_BASE + numStr // we will set -O to this value
	if err := os.MkdirAll(OUT_FILE_BASE+numStr, 0755); err != nil {
		t.Errorf("Error creating %v: %v", OUT_FILE_BASE+numStr, err)
		return
	}

	cmdString := "./mendel-go"
	cmdFailed := false
	stdoutBytes, stderrBytes, err := runCmd(t, cmdString, "-f", inFileName, "-O", dataPath)
	if err != nil {
		t.Errorf("Error running command %v: %v", cmdString, err)
		cmdFailed = true
	}

	if stdoutBytes != nil && cmdFailed {
		t.Logf("stdout: %s", stdoutBytes)
	}
	if stderrBytes != nil && len(stderrBytes) > 0 {
		t.Logf("stderr: %s", stderrBytes)
	}
	if cmdFailed {
		return
	}

	// Open the actual and expected the files
	comparePlainFiles(t, numStr, expNumStr, "", "")
}

func comparePlainFiles(t *testing.T, numStr, expNumStr, outFileDir, expFileDir string) {
	if outFileDir == "" {
		outFileDir = OUT_FILE_BASE + numStr
	}
	if expFileDir == "" {
		expFileDir = EXP_FILE_BASE + expNumStr
	}
	compareFiles(t, outFileDir+"/mendel.fit", expFileDir+"/mendel.fit")
	compareFiles(t, outFileDir+"/mendel.hst", expFileDir+"/mendel.hst")
}

func mendelCaseBin(t *testing.T, num, expNum int, binFile string, andDistBins bool, outFileDir, expFileDir string) {
	mendelCase(t, num, expNum)
	// Also compare the allele-bins files
	numStr := strconv.Itoa(num)
	expNumStr := strconv.Itoa(expNum)
	compareBinFiles(t, numStr, expNumStr, binFile, andDistBins, outFileDir, expFileDir)
}

func compareBinFiles(t *testing.T, numStr, expNumStr, binFile string, andDistBins bool, outFileDir, expFileDir string) {
	if outFileDir == "" {
		outFileDir = OUT_FILE_BASE + numStr
	}
	if expFileDir == "" {
		expFileDir = EXP_FILE_BASE + expNumStr
	}
	compareFiles(t, outFileDir+BIN_SUBDIR+binFile, expFileDir+BIN_SUBDIR+binFile)
	compareFiles(t, outFileDir+NORM_SUBDIR+binFile, expFileDir+NORM_SUBDIR+binFile)
	if andDistBins {
		compareFiles(t, outFileDir+DIST_DEL_SUBDIR+binFile, expFileDir+DIST_DEL_SUBDIR+binFile)
		compareFiles(t, outFileDir+DIST_FAV_SUBDIR+binFile, expFileDir+DIST_FAV_SUBDIR+binFile)
	}
}

func mendelCaseTribeBin(t *testing.T, num, expNum int, binFile string, andDistBins bool) {
	mendelCase(t, num, expNum) // compare the summary files in the top dir
	numStr := strconv.Itoa(num)
	expNumStr := strconv.Itoa(expNum)
	// Go into each of the tribe dirs can compare everything in there
	for _, tribeDir := range getTribeDirs(t, OUT_FILE_BASE+numStr) {
		outFileDir := OUT_FILE_BASE + numStr + "/" + tribeDir
		expFileDir := EXP_FILE_BASE + numStr + "/" + tribeDir
		comparePlainFiles(t, numStr, expNumStr, outFileDir, expFileDir)
		compareBinFiles(t, numStr, expNumStr, binFile, andDistBins, outFileDir, expFileDir)
	}
}

// Run a command with args, and return stdout, stderr
func runCmd(t *testing.T, commandString string, args ...string) ([]byte, []byte, error) {
	// For debug, build the full cmd string
	cmdStr := commandString
	for _, a := range args {
		cmdStr += " " + a
	}
	t.Logf("Running: %v\n", cmdStr)

	// Create the command object with its args
	cmd := exec.Command(commandString, args...)
	if cmd == nil {
		return nil, nil, errors.New("did not return a command object, returned nil")
	}
	// Create the stdout pipe to hold the output from the command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.New("Error retrieving output from command, error: " + err.Error())
	}
	// Create the stderr pipe to hold the errors from the command
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, errors.New("Error retrieving stderr from command, error: " + err.Error())
	}
	// Start the command, which will block for input to std in
	err = cmd.Start()
	if err != nil {
		return nil, nil, errors.New("Unable to start command, error: " + err.Error())
	}
	err = error(nil)
	// Read the output from stdout and stderr into byte arrays
	// stdoutBytes, err := readPipe(stdout)
	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, nil, errors.New("Error reading stdout, error: " + err.Error())
	}
	// stderrBytes, err := readPipe(stderr)
	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		return nil, nil, errors.New("Error reading stderr, error: " + err.Error())
	}
	// Now block waiting for the command to complete
	err = cmd.Wait()
	if err != nil {
		return stdoutBytes, stderrBytes, errors.New("Error waiting for command: " + err.Error())
	}

	return stdoutBytes, stderrBytes, error(nil)
}

// Compare the actual output file with the expected output
func compareFiles(t *testing.T, outputFilename, expectedFilename string) {
	if outputFile, err := ioutil.ReadFile(outputFilename); err != nil {
		t.Errorf("Unable to open %v file, error: %v", outputFilename, err)
		// Read the file into it's own byte array
	} else if expectedFile, err := ioutil.ReadFile(expectedFilename); err != nil {
		t.Errorf("Unable to open %v file, error: %v", expectedFilename, err)
		// Compare the bytes of both files. If there is a difference, then we have a problem so a bunch
		// of diagnostics will be written out.
	} else if bytes.Compare(outputFile, expectedFile) != 0 {
		t.Errorf("Newly created %v file does not match %v file.", outputFilename, expectedFilename)
		// if err := ioutil.WriteFile("./test/new_governor.sls", out2, 0644); err != nil {
		//     t.Errorf("Unable to write ./test/new_governor.sls file, error: %v", err)
		// }
		for idx, val := range outputFile {
			if val == expectedFile[idx] {
				continue
			} else {
				t.Errorf("Found difference in files at index %v", idx)
				padding := 10
				beginIdx := maxInt(idx-padding, 0)
				endIdx := minInt(idx+padding, len(outputFile)-1)
				t.Errorf("bytes around diff in output   file: %v", string(outputFile[beginIdx:endIdx]))
				endIdx = minInt(idx+padding, len(expectedFile)-1)
				t.Errorf("bytes around diff in expected file: %v", string(expectedFile[beginIdx:endIdx]))
				break
			}
		}
	}
}

// Get the non-fully-qualified tribe dirs in the given dir
func getTribeDirs(t *testing.T, dir string) (tribeDirs []string) {
	dirEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Errorf("Error reading dir %s: %v", dir, err)
	}
	for _, d := range dirEntries {
		if d.IsDir() && strings.HasPrefix(d.Name(), "tribe-") {
			tribeDirs = append(tribeDirs, d.Name())
		}
	}
	return
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
