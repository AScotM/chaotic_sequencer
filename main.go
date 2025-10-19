package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"time"
)

// ChaoticConfig holds configuration for chaotic sequence generation
type ChaoticConfig struct {
	Volatility    float64 // 0.0 to 1.0 - how chaotic the sequence is
	TrendStrength float64 // 0.0 to 1.0 - tendency to follow trends
	MeanReversion float64 // 0.0 to 1.0 - tendency to revert to mean
	MinValue      int
	MaxValue      int
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() ChaoticConfig {
	return ChaoticConfig{
		Volatility:    0.7,
		TrendStrength: 0.3,
		MeanReversion: 0.2,
		MinValue:      1,
		MaxValue:      1000,
	}
}

// secureRandIntn generates cryptographically secure random numbers
func secureRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	num, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// Fallback to time-based seeding if crypto fails
		var fallback int64
		if err := binary.Read(rand.Reader, binary.BigEndian, &fallback); err != nil {
			return int(time.Now().UnixNano() % int64(n))
		}
		if fallback < 0 {
			fallback = -fallback
		}
		return int(fallback % int64(n))
	}
	return int(num.Int64())
}

// secureRandFloat64 generates cryptographically secure random float between 0 and 1
func secureRandFloat64() float64 {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return float64(secureRandIntn(1<<53)) / (1 << 53)
	}
	return float64(binary.LittleEndian.Uint64(buf[:])&((1<<53)-1)) / (1 << 53)
}

// ChaoticTransactionSequence generates a chaotic transaction sequence of n steps
func ChaoticTransactionSequence(n int, config ChaoticConfig) ([]map[string]interface{}, error) {
	if n <= 0 {
		return nil, errors.New("the number of steps must be a positive integer")
	}
	if n < 2 {
		return nil, errors.New("sequence length must be at least 2 for proper chaotic behavior")
	}

	sequence := make([]int, n)
	log := make([]map[string]interface{}, n)

	// Initialize with random starting value
	sequence[0] = secureRandIntn(config.MaxValue-config.MinValue+1) + config.MinValue
	log[0] = map[string]interface{}{
		"step":  0,
		"value": sequence[0],
		"type":  "initial",
	}

	// Generate second value
	sequence[1] = clamp(
		sequence[0]+secureRandIntn(21)-10,
		config.MinValue,
		config.MaxValue,
	)
	log[1] = map[string]interface{}{
		"step":  1,
		"value": sequence[1],
		"type":  "random_walk",
	}

	runningMean := float64(sequence[0]+sequence[1]) / 2.0

	for i := 2; i < n; i++ {
		prev1 := sequence[i-1]
		prev2 := sequence[i-2]
		var nextValue int

		randomChoice := secureRandFloat64()
		chaosFactor := secureRandFloat64()*2 - 1 // -1 to 1

		switch {
		case randomChoice < 0.25: // Trend following
			trend := prev1 - prev2
			nextValue = prev1 + int(float64(trend)*config.TrendStrength) + int(chaosFactor*float64(prev1)*0.5)

		case randomChoice < 0.5: // Mean reversion
			deviation := float64(prev1) - runningMean
			nextValue = prev1 - int(deviation*config.MeanReversion) + int(chaosFactor*float64(prev1)*0.3)

		case randomChoice < 0.75: // Multiplicative change
			factors := []float64{0.3, 0.7, 1.3, 1.7, 2.0, -0.5}
			factor := factors[secureRandIntn(len(factors))]
			nextValue = int(float64(prev1)*factor) + int(chaosFactor*10)

		default: // Additive noise with memory
			noise := secureRandIntn(21) - 10
			nextValue = prev1 + (prev1-prev2)/2 + noise
		}

		// Apply volatility
		volatilityEffect := int(chaosFactor * float64(nextValue) * config.Volatility)
		nextValue += volatilityEffect

		// Clamp to valid range
		nextValue = clamp(nextValue, config.MinValue, config.MaxValue)

		sequence[i] = nextValue
		runningMean = (runningMean*float64(i) + float64(nextValue)) / float64(i+1)

		log[i] = map[string]interface{}{
			"step":  i,
			"value": nextValue,
			"type":  getStepType(randomChoice),
		}
	}

	return log, nil
}

// clamp ensures value stays within min-max range
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// getStepType returns a descriptive type for the generation step
func getStepType(randomChoice float64) string {
	switch {
	case randomChoice < 0.25:
		return "trend_following"
	case randomChoice < 0.5:
		return "mean_reversion"
	case randomChoice < 0.75:
		return "multiplicative"
	default:
		return "additive_noise"
	}
}

// ComputeStatistics computes comprehensive statistics for the transaction sequence
func ComputeStatistics(sequence []map[string]interface{}) (map[string]interface{}, error) {
	if len(sequence) == 0 {
		return nil, errors.New("empty sequence")
	}

	// Extract values safely
	values := make([]int, len(sequence))
	for i, entry := range sequence {
		if val, ok := entry["value"].(int); ok {
			values[i] = val
		} else {
			return nil, fmt.Errorf("invalid value type at step %d", i)
		}
	}

	// Calculate basic statistics
	stats := calculateBasicStats(values)
	
	// Calculate advanced statistics
	stats["variance"] = stats["stdev"].(float64) * stats["stdev"].(float64)
	stats["coefficient_of_variation"] = stats["stdev"].(float64) / stats["mean"].(float64)
	stats["q1"] = calculateQuantile(values, 0.25)
	stats["q3"] = calculateQuantile(values, 0.75)
	stats["iqr"] = stats["q3"].(int) - stats["q1"].(int)

	// Trend analysis
	stats["trend_strength"] = calculateTrendStrength(values)
	stats["volatility"] = calculateVolatility(values)

	return stats, nil
}

// calculateBasicStats computes mean, median, standard deviation, min, max
func calculateBasicStats(values []int) map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Sort copy for median calculation
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	// Calculate mean and min/max
	sum := 0
	minVal, maxVal := sorted[0], sorted[0]
	for _, v := range values {
		sum += v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	mean := float64(sum) / float64(len(values))
	
	// Calculate standard deviation
	var variance float64
	for _, v := range values {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)
	stdev := math.Sqrt(variance)

	// Calculate median
	median := 0
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		median = sorted[len(sorted)/2]
	}

	stats["mean"] = mean
	stats["median"] = median
	stats["stdev"] = stdev
	stats["min"] = minVal
	stats["max"] = maxVal
	stats["count"] = len(values)

	return stats
}

// calculateQuantile computes the specified quantile (0.0 to 1.0)
func calculateQuantile(values []int, quantile float64) int {
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	pos := quantile * float64(len(sorted)-1)
	lower := int(pos)
	upper := lower + 1
	weight := pos - float64(lower)

	if upper >= len(sorted) {
		return sorted[lower]
	}
	return int(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// calculateTrendStrength measures how trending the sequence is
func calculateTrendStrength(values []int) float64 {
	if len(values) < 2 {
		return 0.0
	}

	up, down := 0, 0
	for i := 1; i < len(values); i++ {
		if values[i] > values[i-1] {
			up++
		} else if values[i] < values[i-1] {
			down++
		}
	}

	total := up + down
	if total == 0 {
		return 0.0
	}
	return math.Abs(float64(up-down)) / float64(total)
}

// calculateVolatility measures the sequence volatility
func calculateVolatility(values []int) float64 {
	if len(values) < 2 {
		return 0.0
	}

	var sum float64
	for i := 1; i < len(values); i++ {
		change := math.Abs(float64(values[i]) - float64(values[i-1]))
		sum += change
	}
	return sum / float64(len(values)-1)
}

// EnhancedChaoticLogic applies sophisticated chaotic transformations
func EnhancedChaoticLogic(value int, step int) int {
	chaos := secureRandFloat64()
	
	switch {
	case value%11 == 0:
		// Major transformation for values divisible by 11
		return value*3 + secureRandIntn(41) - 20
	case value%7 == 0:
		// Moderate transformation
		return value*2 + secureRandIntn(21) - 10
	case value%5 == 0:
		// Minor transformation
		return value/2 + secureRandIntn(11) - 5
	case step%13 == 0:
		// Periodic major disruption
		return value + secureRandIntn(101) - 50
	case chaos < 0.1:
		// Random major event (10% chance)
		return value + secureRandIntn(201) - 100
	default:
		// Normal chaotic adjustment
		return value + secureRandIntn(21) - 10
	}
}

// ChaoticTransactionSequenceExtended generates sequence with enhanced chaotic logic
func ChaoticTransactionSequenceExtended(n int, config ChaoticConfig) ([]map[string]interface{}, error) {
	log, err := ChaoticTransactionSequence(n, config)
	if err != nil {
		return nil, err
	}

	for i, entry := range log {
		value := entry["value"].(int)
		enhancedValue := EnhancedChaoticLogic(value, i)
		entry["enhanced_value"] = clamp(enhancedValue, config.MinValue, config.MaxValue*2) // Allow larger range for enhanced
		entry["enhancement_delta"] = enhancedValue - value
		log[i] = entry
	}

	return log, nil
}

// SaveToJson saves data to a JSON file with proper error handling
func SaveToJson(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func main() {
	config := DefaultConfig()
	config.Volatility = 0.8 // More chaotic
	config.MaxValue = 500   // Smaller range for better visualization

	log, err := ChaoticTransactionSequenceExtended(50, config) // Smaller sample for demo
	if err != nil {
		fmt.Printf("Error generating sequence: %v\n", err)
		return
	}

	stats, err := ComputeStatistics(log)
	if err != nil {
		fmt.Printf("Error computing statistics: %v\n", err)
		return
	}

	// Print summary
	fmt.Printf("Chaotic Sequence Analysis\n")
	fmt.Printf("========================\n")
	fmt.Printf("Generated %d transactions\n", len(log))
	fmt.Printf("Value Range: %d - %d\n", stats["min"].(int), stats["max"].(int))
	fmt.Printf("Mean: %.2f, Median: %d\n", stats["mean"].(float64), stats["median"].(int))
	fmt.Printf("Std Dev: %.2f, Volatility: %.2f\n", stats["stdev"].(float64), stats["volatility"].(float64))
	fmt.Printf("Trend Strength: %.2f\n", stats["trend_strength"].(float64))
	fmt.Printf("IQR: %d (Q1: %d, Q3: %d)\n", stats["iqr"].(int), stats["q1"].(int), stats["q3"].(int))

	// Save detailed data
	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"config":       config,
			"sequence_length": len(log),
		},
		"statistics": stats,
		"sequence":   log,
	}

	if err := SaveToJson(output, "chaotic_transaction_analysis.json"); err != nil {
		fmt.Printf("Error saving JSON: %v\n", err)
		return
	}
	fmt.Println("\nDetailed analysis saved to chaotic_transaction_analysis.json")

	// Print first 10 entries as sample
	fmt.Println("\nFirst 10 transactions:")
	sample, _ := json.MarshalIndent(log[:10], "", "  ")
	fmt.Println(string(sample))
}
