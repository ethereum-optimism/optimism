package testutil

import "github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"

func SignExtend(value arch.Word, bitToCheck int) arch.Word {
	mostSigBit := value >> (bitToCheck - 1) & 0x1
	if mostSigBit == 1 {
		signBits := ^arch.Word(0) << bitToCheck
		return signBits | value
	}

	return value
}
