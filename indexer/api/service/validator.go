package service

import (
	"errors"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

// Validator ... Validates API user request parameters
type Validator struct{}

// ParseValidateAddress ... Validates and parses the address query parameter
func (v *Validator) ParseValidateAddress(addr string) (common.Address, error) {
	if !common.IsHexAddress(addr) {
		return common.Address{}, errors.New("address must be represented as a valid hexadecimal string")
	}

	parsedAddr := common.HexToAddress(addr)
	if parsedAddr == common.HexToAddress("0x0") {
		return common.Address{}, errors.New("address cannot be the zero address")
	}

	return parsedAddr, nil
}

// ValidateCursor ... Validates and parses the cursor query parameter
func (v *Validator) ValidateCursor(cursor string) error {
	if cursor == "" {
		return nil
	}

	if len(cursor) != 66 { // 0x + 64 chars
		return errors.New("cursor must be a 32 byte hex string")
	}

	if cursor[:2] != "0x" {
		return errors.New("cursor must begin with 0x")
	}

	return nil
}

// ParseValidateLimit ... Validates and parses the limit query parameters
func (v *Validator) ParseValidateLimit(limit string) (int, error) {
	if limit == "" {
		return 100, nil
	}

	val, err := strconv.Atoi(limit)
	if err != nil {
		return 0, errors.New("limit must be an integer value")
	}

	if val <= 0 {
		return 0, errors.New("limit must be greater than 0")
	}

	// TODO - Add a check against a max limit value
	return val, nil
}
