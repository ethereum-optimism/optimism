package testutil

import "github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"

func SignExtend(value arch.Word, valueBitLength int) arch.Word {
	mostSigBit := value >> (valueBitLength - 1) & 0x1
	if mostSigBit == 1 {
		signBits := ^arch.Word(0) << valueBitLength
		return signBits | value
	}

	return value
}
