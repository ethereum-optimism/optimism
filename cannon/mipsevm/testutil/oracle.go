package testutil

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

type TestOracle struct {
	hint        func(v []byte)
	getPreimage func(k [32]byte) []byte
}

var _ mipsevm.PreimageOracle = (*TestOracle)(nil)

func (t *TestOracle) Hint(v []byte) {
	t.hint(v)
}

func (t *TestOracle) GetPreimage(k [32]byte) []byte {
	return t.getPreimage(k)
}

func StaticOracle(t *testing.T, preimageData []byte) *TestOracle {
	return &TestOracle{
		hint: func(v []byte) {},
		getPreimage: func(k [32]byte) []byte {
			if k != preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey() {
				t.Fatalf("invalid preimage request for %x", k)
			}
			return preimageData
		},
	}
}

func StaticPrecompileOracle(t *testing.T, precompile common.Address, requiredGas uint64, input []byte, result []byte) *TestOracle {
	return &TestOracle{
		hint: func(v []byte) {},
		getPreimage: func(k [32]byte) []byte {
			requiredGasB := binary.BigEndian.AppendUint64(nil, requiredGas)
			keyData := append(precompile.Bytes(), requiredGasB...)
			keyData = append(keyData, input...)
			switch k[0] {
			case byte(preimage.Keccak256KeyType):
				if k != preimage.Keccak256Key(crypto.Keccak256Hash(keyData)).PreimageKey() {
					t.Fatalf("invalid preimage request for %x", k)
				}
				return keyData
			case byte(preimage.PrecompileKeyType):
				if k != preimage.PrecompileKey(crypto.Keccak256Hash(keyData)).PreimageKey() {
					t.Fatalf("invalid preimage request for %x", k)
				}
				return result
			}
			panic("unreachable")
		},
	}
}

func SelectOracleFixture(t *testing.T, programName string) mipsevm.PreimageOracle {
	if strings.HasPrefix(programName, "oracle_kzg") {
		precompile := common.BytesToAddress([]byte{0xa})
		input := common.FromHex("01e798154708fe7789429634053cbf9f99b619f9f084048927333fce637f549b564c0a11a0f704f4fc3e8acfe0f8245f0ad1347b378fbf96e206da11a5d3630624d25032e67a7e6a4910df5834b8fe70e6bcfeeac0352434196bdf4b2485d5a18f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7873033e038326e87ed3e1276fd140253fa08e9fc25fb2d9a98527fc22a2c9612fbeafdad446cbc7bcdbdcd780af2c16a")
		blobPrecompileReturnValue := common.FromHex("000000000000000000000000000000000000000000000000000000000000100073eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001")
		requiredGas := uint64(50_000)
		return StaticPrecompileOracle(t, precompile, requiredGas, input, append([]byte{0x1}, blobPrecompileReturnValue...))
	} else if strings.HasPrefix(programName, "oracle") {
		return StaticOracle(t, []byte("hello world"))
	} else {
		return nil
	}
}

type HintTrackingOracle struct {
	hints [][]byte
}

func (t *HintTrackingOracle) Hint(v []byte) {
	t.hints = append(t.hints, v)
}

func (t *HintTrackingOracle) GetPreimage(k [32]byte) []byte {
	return nil
}

func (t *HintTrackingOracle) Hints() [][]byte {
	return t.hints
}
