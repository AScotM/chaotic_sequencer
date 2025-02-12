package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"math/rand"
	"time"
)

// ChaoticTransactionSequence generates a chaotic transaction sequence of n steps
func ChaoticTransactionSequence(n int) ([]map[string]interface{}, error) {
	if n <= 0 {
		return nil, errors.New("the number of steps must be a positive integer")
	}

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	sequence := []int{rand.Intn(10) + 1}
	log := []map[string]interface{}{
		{"step": 0, "value": sequence[0]},
	}

	for i := 1; i < n; i++ {
		prev := sequence[len(sequence)-1]
		var nextValue int

		if rand.Float64() < 0.3 {
			nextValue = prev + rand.Intn(prev*2+1) - prev/2
		} else if rand.Float64() < 0.5 {
			factor := []float64{0.5, 1.5, 2, -1}[rand.Intn(4)]
			nextValue = int(float64(prev) * factor)
		} else {
			noise := rand.Intn(11) - 5
			nextValue = prev + sequence[len(sequence)-2] + noise
		}

		if nextValue < 1 {
			nextValue = 1
		}

		sequence = append(sequence, nextValue)
		log = append(log, map[string]interface{}{"step": i, "value": nextValue})
	}

	return log, nil
}

// ComputeStatistics computes statistics like mean, median, stdev, min, max for the transaction sequence
func ComputeStatistics(sequence []map[string]interface{}) (map[string]interface{}, error) {
	if len(sequence) == 0 {
		return nil, errors.New("empty sequence")
	}

	var values []int
	for _, entry := range sequence {
		values = append(values, entry["value"].(int))
	}

	// Calculate mean
	mean := float64(0)
	for _, v := range values {
		mean += float64(v)
	}
	mean /= float64(len(values))

	// Calculate median
	median := values[len(values)/2]

	// Calculate standard deviation
	var stdev float64
	for _, v := range values {
		stdev += (float64(v) - mean) * (float64(v) - mean)
	}
	stdev = stdev / float64(len(values)-1)

	// Prepare and return stats
	stats := map[string]interface{}{
		"mean":   mean,
		"median": median,
		"stdev":  stdev,
		"min":    min(values),
		"max":    max(values),
	}

	return stats, nil
}

func min(values []int) int {
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(values []int) int {
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// SaveToJson saves the log data to a JSON file
func SaveToJson(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}

// EnhancedChaoticLogic applies an enhanced logic to a value
func EnhancedChaoticLogic(value int) int {
	switch {
	case value%7 == 0:
		return value*2 + rand.Intn(21) - 10
	case value%5 == 0:
		return value/2 + rand.Intn(7) - 3
	default:
		return value + rand.Intn(11) - 5
	}
}

// ChaoticTransactionSequenceExtended generates a chaotic transaction sequence with enhanced logic
func ChaoticTransactionSequenceExtended(n int) ([]map[string]interface{}, error) {
	log, err := ChaoticTransactionSequence(n)
	if err != nil {
		return nil, err
	}

	for i, entry := range log {
		entry["enhanced_value"] = EnhancedChaoticLogic(entry["value"].(int))
		log[i] = entry
	}

	return log, nil
}

func main() {
	n := 200
	log, err := ChaoticTransactionSequenceExtended(n)
	if err != nil {
		fmt.Printf("Error generating sequence: %v\n", err)
		return
	}

	stats, err := ComputeStatistics(log)
	if err != nil {
		fmt.Printf("Error computing statistics: %v\n", err)
		return
	}

	// Print log as JSON
	fmt.Println("\nTransaction Log:")
	logJson, err := json.MarshalIndent(log, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling log: %v\n", err)
		return
	}
	fmt.Println(string(logJson))

	// Print statistics as JSON
	fmt.Println("\nStatistics:")
	statsJson, err := json.MarshalIndent(stats, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling stats: %v\n", err)
		return
	}
	fmt.Println(string(statsJson))

	// Save log to JSON
	if err := SaveToJson(log, "transaction_log.json"); err != nil {
		fmt.Printf("Error saving JSON: %v\n", err)
		return
	}
	fmt.Println("Log saved to transaction_log.json")
}

