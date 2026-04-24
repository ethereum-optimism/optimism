package eip1559

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	HoloceneExtraDataVersionByte = uint8(0x00)
	JovianExtraDataVersionByte   = uint8(0x01)
)

type ForkChecker interface {
	IsHolocene(time uint64) bool
	IsJovian(time uint64) bool
}

// DecodeOptimismExtraData decodes the Optimism extra data.
// It uses the config and parent time to determine how to do the decoding.
// The extraData is expected to already be valid.
func DecodeOptimismExtraData(fc ForkChecker, time uint64, extraData []byte) (uint64, uint64, *uint64) {
	if fc.IsJovian(time) {
		return DecodeJovianExtraData(extraData)
	} else if fc.IsHolocene(time) && len(extraData) == 9 {
		denominator, elasticity := DecodeHolocene1559Params(extraData[1:])
		return denominator, elasticity, nil
	}
	return 0, 0, nil
}

// DecodeHolocene1559Params extracts the Holcene 1559 parameters from the encoded form defined here:
// https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#eip-1559-parameters-in-payloadattributesv3
//
// Returns 0,0 if the format is invalid, though [ValidateHolocene1559Params] should be used instead of this function for
// validity checking.
func DecodeHolocene1559Params(params []byte) (uint64, uint64) {
	if len(params) != 8 {
		return 0, 0
	}
	denominator := binary.BigEndian.Uint32(params[:4])
	elasticity := binary.BigEndian.Uint32(params[4:])
	return uint64(denominator), uint64(elasticity)
}

// EncodeHolocene1559Params encodes the eip-1559 parameters into 'PayloadAttributes.EIP1559Params' format. Will panic if
// either value is outside uint32 range.
func EncodeHolocene1559Params(denom, elasticity uint64) []byte {
	r := make([]byte, 8)
	if denom > math.MaxUint32 || elasticity > math.MaxUint32 {
		panic("eip-1559 parameters out of uint32 range")
	}
	binary.BigEndian.PutUint32(r[:4], uint32(denom))
	binary.BigEndian.PutUint32(r[4:], uint32(elasticity))
	return r
}

// EncodeHoloceneExtraData encodes the eip-1559 parameters into the header 'ExtraData' format. Will panic if either
// value is outside uint32 range.
func EncodeHoloceneExtraData(denom, elasticity uint64) []byte {
	r := make([]byte, 9)
	if denom > math.MaxUint32 || elasticity > math.MaxUint32 {
		panic("eip-1559 parameters out of uint32 range")
	}
	// leave version byte 0
	binary.BigEndian.PutUint32(r[1:5], uint32(denom))
	binary.BigEndian.PutUint32(r[5:], uint32(elasticity))
	return r
}

// ValidateHolocene1559Params checks if the encoded parameters of the payload attributes are valid
// according to the Holocene rules: the encoded denominator and elasticity must both be either
// zero or non-zero.
// Note the difference to the extraData validation, where both values must be non-zero.
func ValidateHolocene1559Params(params []byte) error {
	if len(params) != 8 {
		return fmt.Errorf("holocene eip-1559 params should be 8 bytes, got %d", len(params))
	}
	d, e := DecodeHolocene1559Params(params)
	if e != 0 && d == 0 {
		return errors.New("holocene params cannot have a 0 denominator unless elasticity is also 0")
	} else if e == 0 && d != 0 {
		return errors.New("holocene params cannot have a 0 elasticity unless denominator is also 0")
	}
	return nil
}

// validateHoloceneExtraDataPart validates the Holocene 8-bytes part of extraData:
// the encoded denominator and elasticity must both be non-zero.
// The passed bytes are not checked for correct length, so the caller must ensure it has 8 bytes.
func validateHoloceneExtraDataPart(extra []byte) error {
	d, e := DecodeHolocene1559Params(extra)
	if d == 0 {
		return errors.New("holocene extraData must encode a non-zero denominator")
	} else if e == 0 {
		return errors.New("holocene extraData must encode a non-zero elasticity")
	}
	return nil
}

// ValidateHoloceneExtraData checks if the header extraData is valid according to the Holocene
// rules: the encoded denominator and elasticity must both be non-zero.
// Note the difference to the payload attributes validation, where both values may also be zero
// (at the same time).
func ValidateHoloceneExtraData(extra []byte) error {
	if len(extra) != 9 {
		return fmt.Errorf("holocene extraData should be 9 bytes, got %d", len(extra))
	}
	if extra[0] != HoloceneExtraDataVersionByte {
		return fmt.Errorf("holocene extraData version byte should be %d, got %d", HoloceneExtraDataVersionByte, extra[0])
	}
	return validateHoloceneExtraDataPart(extra[1:])
}

// DecodeJovianExtraData decodes the extraData parameters from the encoded form defined here:
// https://specs.optimism.io/protocol/jovian/exec-engine.html
//
// Returns 0,0,nil if the format is invalid, and d, e, nil for the Holocene length, to provide best effort behavior for non-MinBaseFee extradata, though ValidateJovianExtraData should be used instead of this function for
// validity checking.
func DecodeJovianExtraData(extra []byte) (uint64, uint64, *uint64) {
	// Best effort to decode the extraData for every block in the chain's history,
	// including blocks before the minimum base fee feature was enabled.
	if len(extra) == 9 {
		// This is Holocene extraData
		denominator, elasticity := DecodeHolocene1559Params(extra[1:9])
		return denominator, elasticity, nil
	} else if len(extra) == 17 {
		// Decode extraData when the minimum base fee fork is enabled
		denominator, elasticity := DecodeHolocene1559Params(extra[1:9])
		minBaseFee := binary.BigEndian.Uint64(extra[9:])
		return denominator, elasticity, &minBaseFee
	}
	return 0, 0, nil
}

// EncodeJovianExtraData encodes the eip-1559 and minBaseFee parameters into the header 'ExtraData' format.
// Will panic if eip-1559 parameters are outside uint32 range.
func EncodeJovianExtraData(denom, elasticity, minBaseFee uint64) []byte {
	r := make([]byte, 17)
	if denom > math.MaxUint32 || elasticity > math.MaxUint32 {
		panic("eip-1559 parameters out of uint32 range")
	}
	r[0] = JovianExtraDataVersionByte
	binary.BigEndian.PutUint32(r[1:5], uint32(denom))
	binary.BigEndian.PutUint32(r[5:9], uint32(elasticity))
	binary.BigEndian.PutUint64(r[9:], minBaseFee)
	return r
}

// ValidateJovianExtraData checks if the header extraData is valid according to the Jovian rules:
// the Holocene rules apply to the 8 bytes encoding the eip-1559 parameters and the minBaseFee can
// be set arbitrarily.
func ValidateJovianExtraData(extra []byte) error {
	if len(extra) != 17 {
		return fmt.Errorf("Jovian extraData should be 17 bytes, got %d", len(extra))
	}
	if extra[0] != JovianExtraDataVersionByte {
		return fmt.Errorf("Jovian extraData version byte should be %d, got %d", JovianExtraDataVersionByte, extra[0])
	}
	// Note that the encoded minBaseFee can be set arbitrarily, so no additional validation is done.
	return validateHoloceneExtraDataPart(extra[1:9])
}
