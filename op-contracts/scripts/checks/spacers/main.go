package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// directoryPath is the path to the artifacts directory.
// It can be configured as the first argument to the script or
// defaults to the forge-artifacts directory.
var directoryPath string

func init() {
	if len(os.Args) > 1 {
		directoryPath = os.Args[1]
	} else {
		currentDir, _ := os.Getwd()
		directoryPath = filepath.Join(currentDir, "forge-artifacts")
	}
}

// skipped returns true if the contract should be skipped when inspecting its storage layout.
func skipped(contractName string) bool {
	return strings.Contains(contractName, "CrossDomainMessengerLegacySpacer")
}

// variableInfo represents the parsed variable information.
type variableInfo struct {
	name   string
	slot   int
	offset int
	length int
}

// parseVariableInfo parses out variable info from the variable structure in standard compiler json output.
func parseVariableInfo(variable map[string]interface{}) (variableInfo, error) {
	var info variableInfo
	var err error

	info.name = variable["label"].(string)
	info.slot, err = strconv.Atoi(variable["slot"].(string))
	if err != nil {
		return info, err
	}
	info.offset = int(variable["offset"].(float64))

	variableType := variable["type"].(string)
	if strings.HasPrefix(variableType, "t_mapping") {
		info.length = 32
	} else if strings.HasPrefix(variableType, "t_uint") {
		re := regexp.MustCompile(`uint(\d+)`)
		matches := re.FindStringSubmatch(variableType)
		if len(matches) > 1 {
			bitSize, _ := strconv.Atoi(matches[1])
			info.length = bitSize / 8
		}
	} else if strings.HasPrefix(variableType, "t_bytes_") {
		info.length = 32
	} else if strings.HasPrefix(variableType, "t_bytes") {
		re := regexp.MustCompile(`bytes(\d+)`)
		matches := re.FindStringSubmatch(variableType)
		if len(matches) > 1 {
			info.length, _ = strconv.Atoi(matches[1])
		}
	} else if strings.HasPrefix(variableType, "t_address") {
		info.length = 20
	} else if strings.HasPrefix(variableType, "t_bool") {
		info.length = 1
	} else if strings.HasPrefix(variableType, "t_array") {
		re := regexp.MustCompile(`^t_array\((\w+)\)(\d+)`)
		matches := re.FindStringSubmatch(variableType)
		if len(matches) > 2 {
			innerType := matches[1]
			size, _ := strconv.Atoi(matches[2])
			innerInfo, err := parseVariableInfo(map[string]interface{}{
				"label":  variable["label"],
				"offset": variable["offset"],
				"slot":   variable["slot"],
				"type":   innerType,
			})
			if err != nil {
				return info, err
			}
			info.length = innerInfo.length * size
		}
	} else {
		return info, fmt.Errorf("%s: unsupported type %s, add it to the script", info.name, variableType)
	}

	return info, nil
}

func main() {
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.Contains(path, "t.sol") {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var artifact map[string]interface{}
		err = json.Unmarshal(raw, &artifact)
		if err != nil {
			return err
		}

		storageLayout, ok := artifact["storageLayout"].(map[string]interface{})
		if !ok {
			return nil
		}

		storage, ok := storageLayout["storage"].([]interface{})
		if !ok {
			return nil
		}

		for _, v := range storage {
			variable := v.(map[string]interface{})
			fqn := variable["contract"].(string)

			if skipped(fqn) {
				continue
			}

			label := variable["label"].(string)
			if strings.HasPrefix(label, "spacer_") {
				parts := strings.Split(label, "_")
				if len(parts) != 4 {
					return fmt.Errorf("invalid spacer name format: %s", label)
				}

				slot, _ := strconv.Atoi(parts[1])
				offset, _ := strconv.Atoi(parts[2])
				length, _ := strconv.Atoi(parts[3])

				variableInfo, err := parseVariableInfo(variable)
				if err != nil {
					return err
				}

				if slot != variableInfo.slot {
					return fmt.Errorf("%s %s is in slot %d but should be in %d", fqn, label, variableInfo.slot, slot)
				}

				if offset != variableInfo.offset {
					return fmt.Errorf("%s %s is at offset %d but should be at %d", fqn, label, variableInfo.offset, offset)
				}

				if length != variableInfo.length {
					return fmt.Errorf("%s %s is %d bytes long but should be %d", fqn, label, variableInfo.length, length)
				}

				fmt.Printf("%s.%s is valid\n", fqn, label)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
