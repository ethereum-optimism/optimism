package operatorfee

import (
	"fmt"
)

type TestParams struct {
	ID                  string
	OperatorFeeScalar   uint32
	OperatorFeeConstant uint64
	L1BaseFeeScalar     uint32
	L1BlobBaseFeeScalar uint32
}

func GenerateAllTestParamsCases(numGeneratedValues int) []TestParams {
	// Specific values for testing edge cases
	operatorFeeScalarSpecificValues := []uint32{0, 100}
	operatorFeeConstantSpecificValues := []uint64{0, 100}
	l1BaseFeeScalarSpecificValues := []uint32{0, 100}
	l1BlobBaseFeeScalarSpecificValues := []uint32{0, 100}

	specificValues := GenerateTestParamsCases(
		"specific",
		operatorFeeScalarSpecificValues,
		operatorFeeConstantSpecificValues,
		l1BaseFeeScalarSpecificValues,
		l1BlobBaseFeeScalarSpecificValues,
	)
	return specificValues
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
