package derive

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// l1InfoVectorsFile holds the L1-info deposit vectors that kona asserts too, in
// test_l1_info_shared_vectors (rust/kona/crates/protocol/protocol/src/info/variant.rs).
const l1InfoVectorsFile = "../../../rust/kona/crates/protocol/protocol/testdata/l1_info_vectors.json"

type l1InfoVector struct {
	Name    string `json:"name"`
	Variant string `json:"variant"`
	L1Info  struct {
		Number               uint64         `json:"number"`
		Time                 uint64         `json:"time"`
		BaseFee              *hexutil.Big   `json:"baseFee"`
		BlockHash            common.Hash    `json:"blockHash"`
		SequenceNumber       uint64         `json:"sequenceNumber"`
		BatcherAddr          common.Address `json:"batcherAddr"`
		L1FeeOverhead        eth.Bytes32    `json:"l1FeeOverhead"`
		L1FeeScalar          eth.Bytes32    `json:"l1FeeScalar"`
		BlobBaseFee          *hexutil.Big   `json:"blobBaseFee"`
		BaseFeeScalar        uint32         `json:"baseFeeScalar"`
		BlobBaseFeeScalar    uint32         `json:"blobBaseFeeScalar"`
		OperatorFeeScalar    uint32         `json:"operatorFeeScalar"`
		OperatorFeeConstant  uint64         `json:"operatorFeeConstant"`
		DAFootprintGasScalar uint16         `json:"daFootprintGasScalar"`
	} `json:"l1Info"`
	Calldata   hexutil.Bytes `json:"calldata"`
	SourceHash common.Hash   `json:"sourceHash"`
	TxHash     common.Hash   `json:"txHash"`
}

// TestL1InfoDepositSharedVectors pins, for each L1-info calldata format, the calldata, source hash
// and deposit tx hash that L1InfoDeposit builds from the vector's L1 block and system config
// values, and the values that L1BlockInfoFromBytes decodes from the calldata.
func TestL1InfoDepositSharedVectors(t *testing.T) {
	data, err := os.ReadFile(l1InfoVectorsFile)
	require.NoError(t, err)
	var vectors []l1InfoVector
	require.NoError(t, json.Unmarshal(data, &vectors))

	// The Lagoon (interop) L1 info format is the Jovian one.
	variantForks := map[string][]rollup.ForkName{
		"bedrock": {forks.Regolith},
		"ecotone": {forks.Ecotone},
		"isthmus": {forks.Isthmus},
		"jovian":  {forks.Jovian, forks.Lagoon},
	}
	seen := make(map[string]bool)
	for _, v := range vectors {
		activeForks, ok := variantForks[v.Variant]
		require.True(t, ok, "unknown variant %q", v.Variant)
		seen[v.Variant] = true
		for _, fork := range activeForks {
			t.Run(v.Name+"/"+string(fork), func(t *testing.T) {
				rollupCfg := rollup.Config{BlockTime: 2, Genesis: rollup.Genesis{L2Time: 1000}}
				rollupCfg.ActivateAtGenesis(fork)
				// The block after genesis, so the fork's own format applies.
				l2Time := rollupCfg.Genesis.L2Time + rollupCfg.BlockTime

				in := v.L1Info
				want := L1BlockInfo{
					Number:         in.Number,
					Time:           in.Time,
					BaseFee:        in.BaseFee.ToInt(),
					BlockHash:      in.BlockHash,
					SequenceNumber: in.SequenceNumber,
					BatcherAddr:    in.BatcherAddr,
				}
				sysCfg := eth.SystemConfig{BatcherAddr: in.BatcherAddr}
				block := &testutils.MockBlockInfo{
					InfoHash:    in.BlockHash,
					InfoNum:     in.Number,
					InfoTime:    in.Time,
					InfoBaseFee: in.BaseFee.ToInt(),
				}
				if v.Variant == "bedrock" {
					want.L1FeeOverhead, want.L1FeeScalar = in.L1FeeOverhead, in.L1FeeScalar
					sysCfg.Overhead, sysCfg.Scalar = in.L1FeeOverhead, in.L1FeeScalar
				} else {
					want.BlobBaseFee = in.BlobBaseFee.ToInt()
					want.BaseFeeScalar, want.BlobBaseFeeScalar = in.BaseFeeScalar, in.BlobBaseFeeScalar
					want.OperatorFeeScalar, want.OperatorFeeConstant = in.OperatorFeeScalar, in.OperatorFeeConstant
					want.DAFootprintGasScalar = in.DAFootprintGasScalar
					sysCfg.Scalar = eth.EncodeScalar(eth.EcotoneScalars{
						BaseFeeScalar:     in.BaseFeeScalar,
						BlobBaseFeeScalar: in.BlobBaseFeeScalar,
					})
					sysCfg.OperatorFeeParams = eth.EncodeOperatorFeeParams(eth.OperatorFeeParams{
						Scalar:   in.OperatorFeeScalar,
						Constant: in.OperatorFeeConstant,
					})
					sysCfg.DAFootprintGasScalar = in.DAFootprintGasScalar
					block.InfoBlobBaseFee = in.BlobBaseFee.ToInt()
				}

				dep, err := L1InfoDeposit(&rollupCfg, params.MergedTestChainConfig, sysCfg, in.SequenceNumber, block, l2Time)
				require.NoError(t, err)
				require.Equal(t, hexutil.Bytes(v.Calldata), hexutil.Bytes(dep.Data), "calldata")
				require.Equal(t, v.SourceHash, dep.SourceHash, "source hash")
				require.Equal(t, v.TxHash, dep.Hash(), "deposit tx hash")

				got, err := L1BlockInfoFromBytes(&rollupCfg, l2Time, v.Calldata)
				require.NoError(t, err)
				requireEqual(t, want, *got, "decoded L1 info")
			})
		}
	}
	for variant := range variantForks {
		require.True(t, seen[variant], "no vector for variant %q", variant)
	}
}
