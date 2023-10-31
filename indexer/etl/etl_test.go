package etl

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGetStartingBlock(t *testing.T) {
	// (1) Test when dbHeader is nil

	testHeader := &types.Header{
		Number:     big.NewInt(100),
		ParentHash: common.HexToHash("0x69"),
	}

	testHeaderDB := &database.BlockHeader{
		RLPHeader: &database.RLPHeader{
			Number: big.NewInt(99),
		},
		Number: big.NewInt(99),
	}

	start := getStartingBlock(testHeaderDB, testHeader)
	require.Equal(t, start.Number, testHeaderDB.Number)

	// (2) Test when dbHeader is nil
	start = getStartingBlock(nil, testHeader)
	require.Equal(t, start.Number, testHeader.Number)

	// (3) Test when dbHeader is nil and testHeader == 0
	start = getStartingBlock(nil, &types.Header{Number: big.NewInt(0)})
	require.Nil(t, start)

}
