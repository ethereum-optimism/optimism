package genesis

import (
	"testing"

	bobachain "github.com/bobanetwork/boba/boba-chain-ops/chain"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestCreateHeader(t *testing.T) {
	genesis := types.Genesis{
		Config: &chain.Config{
			ChainID: common.Big1,
		},
	}

	_, err := CreateHeader(&genesis, &types.Header{}, &DeployConfig{})
	require.EqualError(t, err, bobachain.ErrInvalidChainID.Error())

	genesis.Config.ChainID = bobachain.BobaSepoliaChainId
	if _, err := CreateHeader(&genesis, &types.Header{}, &DeployConfig{
		L2OutputOracleStartingBlockNumber: 0,
		L2OutputOracleStartingTimestamp:   0,
	}); err != nil {
		t.Fatal(err)
	}
}
