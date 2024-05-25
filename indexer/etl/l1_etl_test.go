package etl

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestL1ETLConstruction(t *testing.T) {
	etlMetrics := NewMetrics(metrics.NewRegistry(), "l1")

	type testSuite struct {
		db        *database.MockDB
		client    *node.MockEthClient
		start     *big.Int
		contracts config.L1Contracts
	}

	var tests = []struct {
		name         string
		construction func() *testSuite
		assertion    func(*L1ETL, error)
	}{
		{
			name: "Start from L1 config height",
			construction: func() *testSuite {
				client := new(node.MockEthClient)
				db := database.NewMockDB()

				testStart := big.NewInt(100)
				db.MockBlocks.On("L1LatestBlockHeader").Return(nil, nil)

				client.On("BlockHeaderByNumber", mock.MatchedBy(
					bigint.Matcher(100))).Return(
					&types.Header{
						ParentHash: common.HexToHash("0x69"),
					}, nil)

				client.On("GethEthClient").Return(nil)

				return &testSuite{
					db:     db,
					client: client,
					start:  testStart,

					// utilize sample l1 contract configuration (optimism)
					contracts: config.Presets[10].ChainConfig.L1Contracts,
				}
			},
			assertion: func(etl *L1ETL, err error) {
				require.NoError(t, err)
				require.Equal(t, etl.headerTraversal.LastTraversedHeader().ParentHash, common.HexToHash("0x69"))
			},
		},
		{
			name: "Start from recent height stored in DB",
			construction: func() *testSuite {
				client := new(node.MockEthClient)
				db := database.NewMockDB()

				testStart := big.NewInt(100)

				db.MockBlocks.On("L1LatestBlockHeader").Return(
					&database.L1BlockHeader{
						BlockHeader: database.BlockHeader{
							RLPHeader: &database.RLPHeader{
								Number: big.NewInt(69),
							},
						}}, nil)

				client.On("GethEthClient").Return(nil)

				return &testSuite{
					db:     db,
					client: client,
					start:  testStart,

					// utilize sample l1 contract configuration (optimism)
					contracts: config.Presets[10].ChainConfig.L1Contracts,
				}
			},
			assertion: func(etl *L1ETL, err error) {
				require.NoError(t, err)
				header := etl.headerTraversal.LastTraversedHeader()

				require.True(t, header.Number.Cmp(big.NewInt(69)) == 0)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := test.construction()

			logger := testlog.Logger(t, log.LvlInfo)
			cfg := Config{StartHeight: ts.start}

			etl, err := NewL1ETL(cfg, logger, ts.db.DB, etlMetrics, ts.client, ts.contracts, func(cause error) {
				t.Fatalf("crit error: %v", cause)
			})
			test.assertion(etl, err)
		})
	}
}
