package genesis

import (
	"testing"

	bobachain "github.com/bobanetwork/v3-anchorage/boba-chain-ops/chain"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
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
