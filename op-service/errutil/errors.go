package errutil

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type errWithData interface {
	ErrorData() interface{}
}

// TryAddRevertReason attempts to extract the revert reason from geth RPC client errors and adds it to the error message.
// This is most useful when attempting to execute gas, as if the transaction reverts this will then show the reason.
func TryAddRevertReason(err error) error {
	var errData errWithData
	if !errors.As(err, &errData) {
		return err
	}
	reason := DecodeRevertReason(errData.ErrorData())
	return fmt.Errorf("%w, reason: %s", err, reason)
}

// DecodeRevertReason attempts to decode raw EVM revert data into a human-readable string.
// It handles:
//   - Standard Error(string) reverts (selector 0x08c379a0)
//   - Standard Panic(uint256) reverts (selector 0x4e487b71)
//   - Unknown selectors: displayed as hex
//
// The data parameter is expected to be a hex-encoded string (with or without 0x prefix),
// as returned by geth's ErrorData() interface.
func DecodeRevertReason(data interface{}) string {
	hexStr, ok := data.(string)
	if !ok {
		return fmt.Sprintf("%v", data)
	}

	trimmed := strings.TrimPrefix(hexStr, "0x")
	if len(trimmed) == 0 {
		return hexStr
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return hexStr
	}
	if len(raw) < 4 {
		return hexStr
	}

	selector := [4]byte(raw[:4])

	// Standard Error(string) — selector 0x08c379a0
	if selector == [4]byte{0x08, 0xc3, 0x79, 0xa0} {
		strType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: strType}}
		vals, uErr := args.Unpack(raw[4:])
		if uErr == nil && len(vals) > 0 {
			return fmt.Sprintf("Error(%q)", vals[0])
		}
	}

	// Standard Panic(uint256) — selector 0x4e487b71
	if selector == [4]byte{0x4e, 0x48, 0x7b, 0x71} {
		uint256Type, _ := abi.NewType("uint256", "", nil)
		args := abi.Arguments{{Type: uint256Type}}
		vals, uErr := args.Unpack(raw[4:])
		if uErr == nil && len(vals) > 0 {
			code := vals[0].(*big.Int)
			return fmt.Sprintf("Panic(%s: %s)", code, panicReason(code))
		}
	}

	// Unknown custom error — show the raw hex as before
	return hexStr
}

// DecodeRevertReasonWithABIs is like DecodeRevertReason but additionally attempts
// to decode custom errors defined in the provided ABIs. If a matching error selector
// is found, the error name and decoded arguments are returned.
func DecodeRevertReasonWithABIs(data interface{}, abis ...*abi.ABI) string {
	hexStr, ok := data.(string)
	if !ok {
		return fmt.Sprintf("%v", data)
	}

	trimmed := strings.TrimPrefix(hexStr, "0x")
	if len(trimmed) == 0 {
		return hexStr
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return hexStr
	}
	if len(raw) < 4 {
		return hexStr
	}

	// Try ABI custom errors first
	var selector [4]byte
	copy(selector[:], raw[:4])
	for _, a := range abis {
		if a == nil {
			continue
		}
		abiErr, lookupErr := a.ErrorByID(selector)
		if lookupErr != nil {
			continue
		}
		vals, uErr := abiErr.Inputs.Unpack(raw[4:])
		if uErr != nil {
			return fmt.Sprintf("%s(<decode failed>)", abiErr.Name)
		}
		return formatABIError(abiErr, vals)
	}

	// Fall back to standard error decoding
	return DecodeRevertReason(data)
}

// formatABIError formats a decoded ABI error with its argument names and values.
func formatABIError(abiErr *abi.Error, vals []interface{}) string {
	if len(vals) == 0 {
		return fmt.Sprintf("%s()", abiErr.Name)
	}
	parts := make([]string, len(vals))
	for i, v := range vals {
		name := fmt.Sprintf("arg%d", i)
		if i < len(abiErr.Inputs) && abiErr.Inputs[i].Name != "" {
			name = abiErr.Inputs[i].Name
		}
		parts[i] = fmt.Sprintf("%s=%v", name, v)
	}
	return fmt.Sprintf("%s(%s)", abiErr.Name, strings.Join(parts, ", "))
}

// ExtractRevertData extracts the revert data from an error chain, if present.
func ExtractRevertData(err error) (interface{}, bool) {
	var errData errWithData
	if errors.As(err, &errData) {
		return errData.ErrorData(), true
	}
	return nil, false
}

// panicReason returns a human-readable description for standard Solidity panic codes.
func panicReason(code *big.Int) string {
	if !code.IsUint64() {
		return "unknown panic code"
	}
	switch code.Uint64() {
	case 0x00:
		return "generic/compiler-inserted panic"
	case 0x01:
		return "assertion failure"
	case 0x11:
		return "arithmetic overflow/underflow"
	case 0x12:
		return "division or modulo by zero"
	case 0x21:
		return "invalid enum value"
	case 0x22:
		return "invalid storage byte array"
	case 0x31:
		return "pop on empty array"
	case 0x32:
		return "array index out of bounds"
	case 0x41:
		return "out of memory"
	case 0x51:
		return "call to zero-initialized function"
	default:
		return "unknown panic code"
	}
}
