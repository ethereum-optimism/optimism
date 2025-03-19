package compare

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/common"
)

// ChainDiff contains the differences for a specific chain
type AddressDiffs struct {
	Addresses map[string]string `json:"addresses,omitempty"`
	Roles     map[string]string `json:"roles,omitempty"`
}

// CompareAddresses compares all fields in CombinedAddresses with corresponding fields in FetchOutput
func (c *Comparator) CompareAddresses() (map[uint64]AddressDiffs, error) {
	result := make(map[uint64]AddressDiffs)

	for chainID, actualInfo := range c.FetchOutput {
		c.lgr.Info("comparing chain info", "chainName", actualInfo.ChainName, "chainID", chainID)
		expectedInfo, exists := c.CombinedAddresses[chainID]
		if !exists {
			return result, fmt.Errorf("chain ID %d exists in CombinedAddresses but not in FetchOutput", chainID)
		}

		chainDiff := compareChainInfo(expectedInfo, actualInfo)
		if len(chainDiff.Addresses) > 0 || len(chainDiff.Roles) > 0 {
			result[chainID] = chainDiff
		}
	}

	return result, nil
}

// compareChainInfo compares the expected ChainInfo with the actual ChainConfig
func compareChainInfo(expected ChainInfo, actual script.ChainConfig) AddressDiffs {
	diff := AddressDiffs{
		Addresses: make(map[string]string),
		Roles:     make(map[string]string),
	}

	compareAddressFields(reflect.ValueOf(expected.Addresses), reflect.ValueOf(actual.Addresses), diff.Addresses)
	compareAddressFields(reflect.ValueOf(expected.Roles), reflect.ValueOf(actual.Roles), diff.Roles)

	return diff
}

// compareAddressFields compares fields between two structs using reflection
func compareAddressFields(expected, actual reflect.Value, diffs map[string]string) {
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
			diffs[fieldName] = actualAddr
		}
	}
}

// getAddressString converts a reflect.Value to an Ethereum address string. Works for
// string, *string, [20]byte, and *[20]byte values.
func getAddressString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Array:
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
	return ""
}
