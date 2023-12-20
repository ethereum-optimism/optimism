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
func TestUpgradeDepositIntent(t *testing.T) {
	source := UpgradeDepositSource{
		Intent: "Eclipse: L1Block upgrade",
	}

	actual := source.SourceHash()
	expected := "0x7dc74874297a8937186fdbec57ad344647a522de456088557e5fdeda88f66ddd"

	assert.Equal(t, expected, actual.Hex())
}
