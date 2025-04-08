package operatorfee

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"
)

// init seeds the global PRNG using the current time.
var seededRand *rand.Rand // Added package-level generator

func init() {
	// rand.Seed(time.Now().UTC().UnixNano()) // Deprecated
	seededRand = rand.New(rand.NewSource(time.Now().UTC().UnixNano())) // Use new generator
}

type TestParams struct {
	ID                  string
	OperatorFeeScalar   uint32
	OperatorFeeConstant uint64
	L1BaseFeeScalar     uint32
	L1BlobBaseFeeScalar uint32
}

func GenerateAllTestParamsCases(numGeneratedValues int) []TestParams {
	// Specific values for testing edge cases
	operatorFeeScalarSpecificValues := []uint32{0, math.MaxUint32}
	operatorFeeConstantSpecificValues := []uint64{0, math.MaxUint64}
	l1BaseFeeScalarSpecificValues := []uint32{0, math.MaxUint32}
	l1BlobBaseFeeScalarSpecificValues := []uint32{0, math.MaxUint32}

	// Generate random values for broader test coverage
	operatorFeeScalarGeneratedValues := GenerateUint32s(numGeneratedValues)
	operatorFeeConstantGeneratedValues := GenerateUint64s(numGeneratedValues)
	l1BaseFeeScalarGeneratedValues := GenerateUint32s(numGeneratedValues)
	l1BlobBaseFeeScalarGeneratedValues := GenerateUint32s(numGeneratedValues)

	specificValues := GenerateTestParamsCases(
		"specific",
		operatorFeeScalarSpecificValues,
		operatorFeeConstantSpecificValues,
		l1BaseFeeScalarSpecificValues,
		l1BlobBaseFeeScalarSpecificValues,
	)

	generatedValues := GenerateTestParamsCases(
		"generated",
		operatorFeeScalarGeneratedValues,
		operatorFeeConstantGeneratedValues,
		l1BaseFeeScalarGeneratedValues,
		l1BlobBaseFeeScalarGeneratedValues,
	)
	return append(specificValues, generatedValues...)
}

func GenerateTestParamsCases(
	idPrefix string,
	operatorFeeScalarValues []uint32,
	operatorFeeConstantValues []uint64,
	l1FeeScalarValues []uint32,
	l1FeeConstantValues []uint32,
) []TestParams {
	indexCombinations := GenerateIndexCombinations([]int{
		len(operatorFeeScalarValues),
		len(operatorFeeConstantValues),
		len(l1FeeScalarValues),
		len(l1FeeConstantValues),
	})
	results := make([]TestParams, len(indexCombinations))
	for i := 0; i < len(indexCombinations); i++ {
		results[i] = TestParams{
			ID:                  fmt.Sprintf("%s_case_%d", idPrefix, i),
			OperatorFeeScalar:   operatorFeeScalarValues[indexCombinations[i][0]],
			OperatorFeeConstant: operatorFeeConstantValues[indexCombinations[i][1]],
			L1BaseFeeScalar:     l1FeeScalarValues[indexCombinations[i][2]],
			L1BlobBaseFeeScalar: l1FeeConstantValues[indexCombinations[i][3]],
		}
	}
	return results
}

func GenerateUint64s(n int) []uint64 {
	results := make([]uint64, n)
	for i := 0; i < n; i++ {
		// results[i] = rand.Uint64() // Use package generator
		results[i] = seededRand.Uint64()
	}
	return results
}

func GenerateUint32s(n int) []uint32 {
	results := make([]uint32, n)
	for i := 0; i < n; i++ {
		// results[i] = rand.Uint32() // Use package generator
		results[i] = seededRand.Uint32()
	}
	return results
}

func GenerateBigInts(n int, min *big.Int, max *big.Int) []*big.Int {
	results := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		diff := new(big.Int).Sub(max, min)
		diff = diff.Add(diff, big.NewInt(1))
		if diff.Sign() == 0 { // if overflowed (don't think this can happen)
			diff = max
		}
		n, err := cryptoRand.Int(cryptoRand.Reader, diff)
		if err != nil {
			panic(fmt.Errorf("could not generate a random big.Int: %w", err).Error())
		}
		results[i] = new(big.Int).Add(n, min)
	}
	return results
}

func GenerateIndexCombinations(lengths []int) [][]int {
	if len(lengths) == 0 {
		return [][]int{}
	}

	// Calculate the total number of combinations
	totalCombinations := 1
	for _, length := range lengths {
		totalCombinations *= length
	}

	// Initialize the result slice
	result := make([][]int, totalCombinations)
	for i := range result {
		result[i] = make([]int, len(lengths))
	}

	// Generate all combinations
	divisor := 1
	for i := len(lengths) - 1; i >= 0; i-- {
		for j := 0; j < totalCombinations; j++ {
			result[j][i] = (j / divisor) % lengths[i]
		}
		divisor *= lengths[i]
	}

	return result
}
