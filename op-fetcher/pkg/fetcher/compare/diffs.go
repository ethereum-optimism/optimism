package compare

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// ChainDiff contains the differences for a specific chain
type ChainDiff struct {
	AddressDiffs map[string]AddressDiff
	RoleDiffs    map[string]AddressDiff
}

// AddressDiff represents a difference in an address value
type AddressDiff struct {
	Expected string
	Actual   string
}

// CompareAddresses compares all fields in CombinedAddresses with corresponding fields in FetchOutput
func CompareAddresses(lgr log.Logger, cfg *CompareConfig) (map[uint64]ChainDiff, error) {
	result := make(map[uint64]ChainDiff)

	for chainID, actualInfo := range cfg.FetchOutput {
		lgr.Info("comparing chain info", "chainID", chainID)
		expectedInfo, exists := cfg.CombinedAddresses[chainID]
		if !exists {
			return result, fmt.Errorf("chain ID %d exists in CombinedAddresses but not in FetchOutput", chainID)
		}

		chainDiff := compareChainInfo(expectedInfo, actualInfo)
		if len(chainDiff.AddressDiffs) > 0 || len(chainDiff.RoleDiffs) > 0 {
			result[chainID] = chainDiff
		}
	}

	return result, nil
}

// compareChainInfo compares the expected ChainInfo with the actual ChainConfig
func compareChainInfo(expected ChainInfo, actual script.ChainConfig) ChainDiff {
	diff := ChainDiff{
		AddressDiffs: make(map[string]AddressDiff),
		RoleDiffs:    make(map[string]AddressDiff),
	}

	compareAddressFields(reflect.ValueOf(expected.Addresses), reflect.ValueOf(actual.Addresses), diff.AddressDiffs)
	compareAddressFields(reflect.ValueOf(expected.Roles), reflect.ValueOf(actual.Roles), diff.RoleDiffs)

	return diff
}

// compareAddressFields compares fields between two structs using reflection
func compareAddressFields(expected, actual reflect.Value, diffs map[string]AddressDiff) {
	expectedType := expected.Type()

	for i := 0; i < expectedType.NumField(); i++ {
		fieldName := expectedType.Field(i).Name

		expectedField := expected.Field(i)
		actualField := actual.FieldByName(fieldName)

		// Check if the field exists in the actual struct
		if !actualField.IsValid() {
			fmt.Printf("field %s does not exist in actual struct\n", fieldName)
			continue
		}

		// normalize addresses before comparison
		expectedAddr := getAddressString(expectedField)
		actualAddr := getAddressString(actualField)

		if !strings.EqualFold(expectedAddr, actualAddr) && expectedAddr != "" {
			diffs[fieldName] = AddressDiff{
				Expected: expectedAddr,
				Actual:   actualAddr,
			}
		}
	}
}

// getAddressString converts a reflect.Value to an Ethereum address string
func getAddressString(v reflect.Value) string {
	// Handle different types that might represent addresses
	switch v.Kind() {
	case reflect.String:
		// For string values, return the string directly
		return v.String()
	case reflect.Array:
		// Check if it's a common.Address ([20]byte)
		if v.Type() == reflect.TypeOf(common.Address{}) {
			// Convert to common.Address and get hex representation
			return common.Address(v.Interface().(common.Address)).Hex()
		}
	case reflect.Ptr:
		// For pointer values, dereference if not nil and try again
		if !v.IsNil() {
			return getAddressString(v.Elem())
		}
	case reflect.Interface:
		// For interface values, get the concrete value and try again
		if !v.IsNil() {
			return getAddressString(v.Elem())
		}
	}

	// If we can't determine the address, return empty string
	return ""
}

// FormatDiffs returns a human-readable string of the differences
func FormatDiffs(result map[uint64]ChainDiff) string {
	var sb strings.Builder

	for chainID, chainDiff := range result {
		sb.WriteString(fmt.Sprintf("Chain ID %d differences:\n", chainID))

		if len(chainDiff.AddressDiffs) > 0 {
			sb.WriteString("  Address differences:\n")
			for field, diff := range chainDiff.AddressDiffs {
				sb.WriteString(fmt.Sprintf("    %s:\n", field))
				sb.WriteString(fmt.Sprintf("      Expected: %s\n", diff.Expected))
				sb.WriteString(fmt.Sprintf("      Actual:   %s\n", diff.Actual))
			}
		}

		if len(chainDiff.RoleDiffs) > 0 {
			sb.WriteString("  Role differences:\n")
			for field, diff := range chainDiff.RoleDiffs {
				sb.WriteString(fmt.Sprintf("    %s:\n", field))
				sb.WriteString(fmt.Sprintf("      Expected: %s\n", diff.Expected))
				sb.WriteString(fmt.Sprintf("      Actual:   %s\n", diff.Actual))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
