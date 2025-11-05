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

const (
	MaxFloatPrecision = 1 << 53
	TrendThreshold    = 0.25
	MeanReversionThreshold = 0.5
	MultiplicativeThreshold = 0.75
)

type ChaoticConfig struct {
	Volatility    float64
	TrendStrength float64
	MeanReversion float64
	MinValue      int
	MaxValue      int
	Seed          int64
}

type TransactionStep struct {
	Step            int    `json:"step"`
	Value           int    `json:"value"`
	Type            string `json:"type"`
	EnhancedValue   int    `json:"enhanced_value,omitempty"`
	EnhancementDelta int   `json:"enhancement_delta,omitempty"`
}

type SequenceStatistics struct {
	Count                 int     `json:"count"`
	Mean                  float64 `json:"mean"`
	Median                int     `json:"median"`
	Stdev                 float64 `json:"stdev"`
	Min                   int     `json:"min"`
	Max                   int     `json:"max"`
	Variance              float64 `json:"variance"`
	CoefficientOfVariation float64 `json:"coefficient_of_variation"`
	Q1                    int     `json:"q1"`
	Q3                    int     `json:"q3"`
	IQR                   int     `json:"iqr"`
	TrendStrength         float64 `json:"trend_strength"`
	Volatility            float64 `json:"volatility"`
}

type GenerationMetadata struct {
	GeneratedAt     string       `json:"generated_at"`
	Config          ChaoticConfig `json:"config"`
	SequenceLength  int          `json:"sequence_length"`
}

type OutputData struct {
	Metadata   GenerationMetadata `json:"metadata"`
	Statistics SequenceStatistics `json:"statistics"`
	Sequence   []TransactionStep  `json:"sequence"`
}

func DefaultConfig() ChaoticConfig {
	return ChaoticConfig{
		Volatility:    0.7,
		TrendStrength: 0.3,
		MeanReversion: 0.2,
		MinValue:      1,
		MaxValue:      1000,
		Seed:          time.Now().UnixNano(),
	}
}

func (c ChaoticConfig) Validate() error {
	if c.MinValue >= c.MaxValue {
		return errors.New("MinValue must be less than MaxValue")
	}
	if c.Volatility < 0 || c.Volatility > 1 {
		return errors.New("Volatility must be between 0 and 1")
	}
	if c.TrendStrength < 0 || c.TrendStrength > 1 {
		return errors.New("TrendStrength must be between 0 and 1")
	}
	if c.MeanReversion < 0 || c.MeanReversion > 1 {
		return errors.New("MeanReversion must be between 0 and 1")
	}
	return nil
}

func secureRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	num, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
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

func secureRandFloat64() float64 {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return float64(secureRandIntn(MaxFloatPrecision)) / float64(MaxFloatPrecision)
	}
	return float64(binary.LittleEndian.Uint64(buf[:])&(MaxFloatPrecision-1)) / float64(MaxFloatPrecision)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func getStepType(randomChoice float64) string {
	switch {
	case randomChoice < TrendThreshold:
		return "trend_following"
	case randomChoice < MeanReversionThreshold:
		return "mean_reversion"
	case randomChoice < MultiplicativeThreshold:
		return "multiplicative"
	default:
		return "additive_noise"
	}
}

func ChaoticTransactionSequence(n int, config ChaoticConfig) ([]TransactionStep, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, errors.New("the number of steps must be a positive integer")
	}
	if n < 2 {
		return nil, errors.New("sequence length must be at least 2 for proper chaotic behavior")
	}

	sequence := make([]int, n)
	log := make([]TransactionStep, n)

	sequence[0] = secureRandIntn(config.MaxValue-config.MinValue+1) + config.MinValue
	log[0] = TransactionStep{
		Step:  0,
		Value: sequence[0],
		Type:  "initial",
	}

	sequence[1] = clamp(
		sequence[0]+secureRandIntn(21)-10,
		config.MinValue,
		config.MaxValue,
	)
	log[1] = TransactionStep{
		Step:  1,
		Value: sequence[1],
		Type:  "random_walk",
	}

	runningMean := float64(sequence[0]+sequence[1]) / 2.0

	for i := 2; i < n; i++ {
		prev1 := sequence[i-1]
		prev2 := sequence[i-2]
		var nextValue int

		randomChoice := secureRandFloat64()
		chaosFactor := secureRandFloat64()*2 - 1

		switch {
		case randomChoice < TrendThreshold:
			trend := prev1 - prev2
			nextValue = prev1 + int(float64(trend)*config.TrendStrength) + int(chaosFactor*float64(prev1)*0.5)

		case randomChoice < MeanReversionThreshold:
			deviation := float64(prev1) - runningMean
			nextValue = prev1 - int(deviation*config.MeanReversion) + int(chaosFactor*float64(prev1)*0.3)

		case randomChoice < MultiplicativeThreshold:
			factors := []float64{0.3, 0.7, 1.3, 1.7, 2.0, -0.5}
			factor := factors[secureRandIntn(len(factors))]
			nextValue = int(float64(prev1)*factor) + int(chaosFactor*10)

		default:
			noise := secureRandIntn(21) - 10
			nextValue = prev1 + (prev1-prev2)/2 + noise
		}

		volatilityEffect := int(chaosFactor * float64(nextValue) * config.Volatility)
		nextValue += volatilityEffect
		nextValue = clamp(nextValue, config.MinValue, config.MaxValue)

		sequence[i] = nextValue
		runningMean = (runningMean*float64(i) + float64(nextValue)) / float64(i+1)

		log[i] = TransactionStep{
			Step:  i,
			Value: nextValue,
			Type:  getStepType(randomChoice),
		}
	}

	return log, nil
}

func calculateBasicStats(values []int) SequenceStatistics {
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

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
	
	var variance float64
	for _, v := range values {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)
	stdev := math.Sqrt(variance)

	median := 0
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		median = sorted[len(sorted)/2]
	}

	return SequenceStatistics{
		Mean:   mean,
		Median: median,
		Stdev:  stdev,
		Min:    minVal,
		Max:    maxVal,
		Count:  len(values),
	}
}

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

func ComputeStatistics(sequence []TransactionStep) (SequenceStatistics, error) {
	if len(sequence) == 0 {
		return SequenceStatistics{}, errors.New("empty sequence")
	}

	values := make([]int, len(sequence))
	for i, entry := range sequence {
		values[i] = entry.Value
	}

	stats := calculateBasicStats(values)
	
	stats.Variance = stats.Stdev * stats.Stdev
	stats.CoefficientOfVariation = stats.Stdev / stats.Mean
	stats.Q1 = calculateQuantile(values, 0.25)
	stats.Q3 = calculateQuantile(values, 0.75)
	stats.IQR = stats.Q3 - stats.Q1
	stats.TrendStrength = calculateTrendStrength(values)
	stats.Volatility = calculateVolatility(values)

	return stats, nil
}

func EnhancedChaoticLogic(value int, step int) int {
	chaos := secureRandFloat64()
	
	switch {
	case value%11 == 0:
		return value*3 + secureRandIntn(41) - 20
	case value%7 == 0:
		return value*2 + secureRandIntn(21) - 10
	case value%5 == 0:
		return value/2 + secureRandIntn(11) - 5
	case step%13 == 0:
		return value + secureRandIntn(101) - 50
	case chaos < 0.1:
		return value + secureRandIntn(201) - 100
	default:
		return value + secureRandIntn(21) - 10
	}
}

func ChaoticTransactionSequenceExtended(n int, config ChaoticConfig) ([]TransactionStep, error) {
	log, err := ChaoticTransactionSequence(n, config)
	if err != nil {
		return nil, err
	}

	for i := range log {
		value := log[i].Value
		enhancedValue := EnhancedChaoticLogic(value, i)
		log[i].EnhancedValue = clamp(enhancedValue, config.MinValue, config.MaxValue*2)
		log[i].EnhancementDelta = enhancedValue - value
	}

	return log, nil
}

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
	config.Volatility = 0.8
	config.MaxValue = 500

	log, err := ChaoticTransactionSequenceExtended(50, config)
	if err != nil {
		fmt.Printf("Error generating sequence: %v\n", err)
		return
	}

	stats, err := ComputeStatistics(log)
	if err != nil {
		fmt.Printf("Error computing statistics: %v\n", err)
		return
	}

	fmt.Printf("Chaotic Sequence Analysis\n")
	fmt.Printf("========================\n")
	fmt.Printf("Generated %d transactions\n", len(log))
	fmt.Printf("Value Range: %d - %d\n", stats.Min, stats.Max)
	fmt.Printf("Mean: %.2f, Median: %d\n", stats.Mean, stats.Median)
	fmt.Printf("Std Dev: %.2f, Volatility: %.2f\n", stats.Stdev, stats.Volatility)
	fmt.Printf("Trend Strength: %.2f\n", stats.TrendStrength)
	fmt.Printf("IQR: %d (Q1: %d, Q3: %d)\n", stats.IQR, stats.Q1, stats.Q3)

	output := OutputData{
		Metadata: GenerationMetadata{
			GeneratedAt:    time.Now().Format(time.RFC3339),
			Config:         config,
			SequenceLength: len(log),
		},
		Statistics: stats,
		Sequence:   log,
	}

	if err := SaveToJson(output, "chaotic_transaction_analysis.json"); err != nil {
		fmt.Printf("Error saving JSON: %v\n", err)
		return
	}
	fmt.Println("\nDetailed analysis saved to chaotic_transaction_analysis.json")

	fmt.Println("\nFirst 10 transactions:")
	sample, _ := json.MarshalIndent(log[:10], "", "  ")
	fmt.Println(string(sample))
}
