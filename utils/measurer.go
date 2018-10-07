package utils

import (
	"time"
	"log"
	"strconv"
	"sort"
	"runtime"
)

// Measure keeps track of how much execution time was spent in various parts of the code, and the maximum amount of memory
type Measurer struct {
	DeltaTime map[string]time.Time		// holds the start time for 1 duration measurement
	TotalTime map[string]int64
	MaxTotalMemory uint64		// the amount of memory this program used
	Track bool
}

// Measure is a singleton instance of Measure to track execution time
var Measure *Measurer


// MeasurerFactory creates an instance of Measurer
func MeasurerFactory(verbosity uint32) {
	Measure = &Measurer{}

	if verbosity >= 1 { Measure.Track = true }		// don't bother tracking the times if we are not going to print them
	Measure.DeltaTime = make(map[string]time.Time)
	Measure.TotalTime = make(map[string]int64)
}


// Start starts the time measuring of a section of code. Returns the Measurer instance to Start and Stop can be chained in a defer statement.
func (m *Measurer) Start(codeName string) *Measurer {
	//fmt.Printf("Measurer: starting %s\n", codeName)
	if !m.Track { return m }
	if !m.DeltaTime[codeName].IsZero() {
		// We were in the middle of a measurement
		log.Fatalf("Error: Measurer.Start(%v) called before the previous measurement was stopped.", codeName)
	}
	m.DeltaTime[codeName] = time.Now()
	return m
}


// GetInterimTime returns the total number of seconds (as a float64) so far for this codeName.
func (m *Measurer) GetInterimTime(codeName string) float64 {
	var delta int64
	if !m.DeltaTime[codeName].IsZero() {
		// We are in the middle of a measurement, so add it to the total
		delta = int64(time.Since(m.DeltaTime[codeName]))
	}
	total := delta + m.TotalTime[codeName]
	return float64(total)/float64(time.Second)
}


// Stop stops the time measuring of a section of code and adds this amount to the total for the run
func (m *Measurer) Stop(codeName string) (delta float64) {
	//fmt.Printf("Measurer: stopping %s\n", codeName)
	if !m.Track { return }
	if m.DeltaTime[codeName].IsZero() {
		// We did not start a measurement
		log.Fatalf("Error: Measurer.Stop(%v) called without previously calling Start.", codeName)
	}
	// DeltaTime[codeName] currently holds the start time
	deltaInt := int64(time.Since(m.DeltaTime[codeName]))
	m.TotalTime[codeName] += deltaInt
	delete(m.DeltaTime, codeName)		// zero it out so we know we are not in the middle of a measurement
	delta = float64(deltaInt)/float64(time.Second)
	return
}


func (m *Measurer) CheckAmountMemoryUsed() {
	if !m.Track { return }
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	m.MaxTotalMemory = MaxUint64(m.MaxTotalMemory, mem.Sys)
}


// GetAmountMemoryUsed returns the max memory used so far in MB
func (m *Measurer) GetAmountMemoryUsed() float32 {
	return float32(m.MaxTotalMemory)/1000000.0
}


// LogSummary prints to the log all of the total times.
func (m *Measurer) LogSummary() {
	if !m.Track { return }
	// We want the map output to be sorted by key to make comparisons between runs easier, so create an array of the keys
	var codeNames []string
	for codeName := range m.TotalTime { codeNames = append(codeNames, codeName) }
	sort.Strings(codeNames)

	separator := ""
	var str string
	//for codeName, dur := range m.TotalTime {
	for _, codeName := range codeNames {
		dur := m.TotalTime[codeName]
		if !m.DeltaTime[codeName].IsZero() {
			// We did not finish a measurement
			log.Fatalf("Error: measurement for %v was not completed when Measurer.LogSummary() was called.", codeName)
		}
		str += separator + codeName + ": " + strconv.FormatFloat(float64(dur)/float64(time.Second), 'f', 4, 64) + "s"
		if separator == "" { separator = ", " }
	}
	log.Printf("Time measurements: %v", str)

	m.CheckAmountMemoryUsed()
	log.Printf("Maximum amount of memory used: %.3f MB\n", m.GetAmountMemoryUsed())
}
