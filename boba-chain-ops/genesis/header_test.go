package genesis

import (
	"testing"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	bobachain "github.com/ledgerwatch/erigon/boba-chain-ops/chain"
	"github.com/ledgerwatch/erigon/core/types"
)

func TestCreateHeader(t *testing.T) {
	genesis := types.Genesis{
		Config: &chain.Config{
			ChainID: common.Big1,
		},
	}
	if _, err := CreateHeader(&genesis); err == nil {
		t.Fatal("expected error")
	}
	genesis.Config.ChainID = bobachain.BobaGoerliChainId
	if _, err := CreateHeader(&genesis); err != nil {
		t.Fatal(err)
	}
}
