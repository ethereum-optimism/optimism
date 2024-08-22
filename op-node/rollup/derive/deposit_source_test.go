package derive

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// TestEcotone4788ContractSourceHash
// cast keccak $(cast concat-hex 0x0000000000000000000000000000000000000000000000000000000000000002 $(cast keccak "Ecotone: L1 Block Proxy Update"))
// # 0x18acb38c5ff1c238a7460ebc1b421fa49ec4874bdf1e0a530d234104e5e67dbc
func TestDeposit(t *testing.T) {
	source := UpgradeDepositSource{
		Intent: "Ecotone: L1 Block Proxy Update",
	}

	actual := source.SourceHash()
	expected := "0x18acb38c5ff1c238a7460ebc1b421fa49ec4874bdf1e0a530d234104e5e67dbc"

	assert.Equal(t, expected, actual.Hex())
}

// TestEcotone4788ContractSourceHash tests that the source-hash of the 4788 deposit deployment tx is correct.
// As per specs, the hash is computed as:
// cast keccak $(cast concat-hex 0x0000000000000000000000000000000000000000000000000000000000000002 $(cast keccak "Ecotone: beacon block roots contract deployment"))
// # 0x69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c
func TestEcotone4788ContractSourceHash(t *testing.T) {
	source := UpgradeDepositSource{
		Intent: "Ecotone: beacon block roots contract deployment",
	}

	actual := source.SourceHash()
	expected := "0x69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c"

	assert.Equal(t, expected, actual.Hex())
}

// TestAfterForceIncludeSourceHash
// cast keccak $(cast concat-hex 0x0000000000000000000000000000000000000000000000000000000000000003 $(cast keccak 0x01))
// # 0x8afb1c4a581d0e71ab65334e3365ba5511fb15c13fa212776f9d4dafc6287845
func TestAfterForceIncludeSource(t *testing.T) {
	source := AfterForceIncludeSource{
		L1BlockHash: common.Hash{0x01},
	}

	actual := source.SourceHash()
	expected := "0x8afb1c4a581d0e71ab65334e3365ba5511fb15c13fa212776f9d4dafc6287845"

	assert.Equal(t, expected, actual.Hex())
}
