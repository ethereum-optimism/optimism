package derive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
# compute intent hash:
cast keccak "Eclipse: L1Block upgrade"
# 0x831b745c7397f93704ae55eb0100bf3c56fe9e304d3f21c1a93ec25f736fea26
# source hash type:
# 0x0000000000000000000000000000000000000000000000000000000000000002
# compute source hash:
cast keccak 0x0000000000000000000000000000000000000000000000000000000000000002831b745c7397f93704ae55eb0100bf3c56fe9e304d3f21c1a93ec25f736fea26
# 0x7dc74874297a8937186fdbec57ad344647a522de456088557e5fdeda88f66ddd
*/
func TestUpgradeDepositIntentEclipseBlockUpgrade(t *testing.T) {
	source := UpgradeDepositSource{
		Intent: "Eclipse: L1Block upgrade",
	}

	actual := source.SourceHash()
	expected := "0x7dc74874297a8937186fdbec57ad344647a522de456088557e5fdeda88f66ddd"

	assert.Equal(t, expected, actual.Hex())
}

/*
# compute intent hash:
cast keccak "Ecotone: L1Block upgrade"
# 0xaf2b20ee05be9fc3f0712050591a5f8988f94b56cdf48842863a773b76634fde
# source hash type:
# 0x0000000000000000000000000000000000000000000000000000000000000002
# compute source hash:
cast keccak 0x0000000000000000000000000000000000000000000000000000000000000002af2b20ee05be9fc3f0712050591a5f8988f94b56cdf48842863a773b76634fde
# 0x7795a90486cc207315616d57bdbe5ca0ad63b22b5ba7fe087d11774f5de6e10b
*/
func TestUpgradeDepositIntentEcotoneBlockUpgrade(t *testing.T) {
	source := UpgradeDepositSource{
		Intent: "Ecotone: L1Block upgrade",
	}

	actual := source.SourceHash()
	expected := "0x7795a90486cc207315616d57bdbe5ca0ad63b22b5ba7fe087d11774f5de6e10b"

	assert.Equal(t, expected, actual.Hex())
}

/*
# compute intent hash:
cast keccak "Ecotone: beacon block roots contract deployment"
# 0x4e73a20ffe4a8330eb1f726862f4b062301e73d081c6d3824a6e0bd6428697fe
# source hash type:
# 0x0000000000000000000000000000000000000000000000000000000000000002
# compute source hash:
cast keccak 0x0000000000000000000000000000000000000000000000000000000000000002ab0dfc96b47739a0ae1d415bbfaae79ebb1111861a3b7cfbbaa6ca4a9e618357
# 0x69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c
*/
func TestUpgradeDepositIntentEcotonContractUpgrade(t *testing.T) {
	source := UpgradeDepositSource{
		Intent: "Ecotone: beacon block roots contract deployment",
	}

	actual := source.SourceHash()
	expected := "0x69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c"

	assert.Equal(t, expected, actual.Hex())
}
