package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
)

func getContractBundleFromState(filepath string) (map[string]common.Address, error) {
	_, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("input file not found: %s", filepath)
	}

	bundleData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file %s: %w", filepath, err)
	}

	var st state.State
	if err := json.Unmarshal(bundleData, &st); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	bundle := make(map[string]common.Address)

	if st.SuperchainDeployment != nil {
		addContractsFromStruct("superchain", st.SuperchainDeployment, bundle)
	}

	if st.ImplementationsDeployment != nil {
		addContractsFromStruct("implementations", st.ImplementationsDeployment, bundle)
	}

	for _, chain := range st.Chains {
		chainPrefix := fmt.Sprintf("opchain_%s", chain.ID.Hex())
		addContractsFromStruct(chainPrefix, &chain.OpChainContracts, bundle)
	}

	return bundle, nil
}

func addContractsFromStruct(prefix string, structPtr interface{}, bundle map[string]common.Address) {
	val := reflect.ValueOf(structPtr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if field.Type() == reflect.TypeOf(common.Address{}) {
			addr := field.Interface().(common.Address)
			if addr != (common.Address{}) {
				contractName := fieldNameToContractName(fieldType.Name)
				key := fmt.Sprintf("%s_%s", prefix, contractName)
				bundle[key] = addr
			}
		} else if field.Kind() == reflect.Struct {
			addContractsFromStruct(prefix, field.Addr().Interface(), bundle)
		}
	}
}

func fieldNameToContractName(fieldName string) string {
	parts := []string{}
	currentWord := ""

	for i, c := range fieldName {
		if i > 0 && c >= 'A' && c <= 'Z' {
			if currentWord != "" {
				parts = append(parts, currentWord)
			}
			currentWord = string(c)
		} else {
			currentWord += string(c)
		}
	}

	if currentWord != "" {
		parts = append(parts, currentWord)
	}

	return strings.ToLower(strings.Join(parts, "_"))
}

func isStateFile(filepath string) bool {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return false
	}

	var st state.State
	if err := json.Unmarshal(data, &st); err != nil {
		return false
	}

	return st.SuperchainDeployment != nil || st.ImplementationsDeployment != nil || len(st.Chains) > 0
}

func GetBundleFromFile(filepath string) (map[string]common.Address, error) {
	if isStateFile(filepath) {
		return getContractBundleFromState(filepath)
	}

	return getContractBundle(filepath)
}

func getContractBundle(filepath string) (map[string]common.Address, error) {
	_, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("input file not found: %s", filepath)
	}

	bundleData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundle file %s: %w", filepath, err)
	}

	var bundle map[string]common.Address
	if err := json.Unmarshal(bundleData, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse superchain bundle: %w", err)
	}

	return bundle, nil
}
