package feature

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var EnableCustomizeL1BlockBaseFee bool
var EnableCustomizeSuggestedL1BaseFee bool
var EnableCustomizeL1Label bool
var EnableCustomizeCraftL1Transaction bool
var EnableCustomizeProposeL1BlockHash bool

// EnableCoordinator  is true when the driver should request permission from op-coordinator before building new blocks.
var EnableCoordinator bool

// CoordinatorAddr is the address of the Coordinator JSON-RPC endpoint to use.
var CoordinatorAddr string

// CoordinatorSequencerID is the identifier of the sequencer node to request blocks from.
// It must be unique and same as the name of the sequencer node configured in the Coordinator service.
var CoordinatorSequencerID string
var Coordinator *CoordinatorClient

var DefaultCustomizedBaseFee = big.NewInt(5000000000)
var DefaultCustomizedSuggestedL1BaseFee = big.NewInt(0)
var DefaultCustomizedL1LabelSub = uint64(15)
var DefaultCustomizedProposeL1BlockHash = common.Hash{}

func init() {
	EnableCustomizeL1BlockBaseFee = os.Getenv("OP_FEATURE_ENABLE_CUSTOMIZE_L1_BLOCK_BASE_FEE") == "true"
	EnableCustomizeSuggestedL1BaseFee = os.Getenv("OP_FEATURE_ENABLE_CUSTOMIZE_SUGGESTED_L1_BASE_FEE") == "true"
	EnableCustomizeL1Label = os.Getenv("OP_FEATURE_ENABLE_CUSTOMIZE_L1_LABEL") == "true"
	EnableCustomizeCraftL1Transaction = os.Getenv("OP_FEATURE_ENABLE_CUSTOMIZE_CRAFT_L1_TRANSACTION") == "true"
	EnableCustomizeProposeL1BlockHash = os.Getenv("OP_FEATURE_ENABLE_CUSTOMIZE_PROPOSE_L1_BLOCK_HASH") == "true"
	EnableCoordinator = os.Getenv("OP_FEATURE_ENABLE_COORDINATOR") == "true"
	CoordinatorAddr = os.Getenv("OP_FEATURE_COORDINATOR_ADDR")
	CoordinatorSequencerID = os.Getenv("OP_FEATURE_COORDINATOR_SEQUENCER_ID")

	// Initialize coordinator client
	if EnableCoordinator {
		if CoordinatorAddr == "" || CoordinatorSequencerID == "" {
			panic("CoordinatorAddr and CoordinatorSequencerID must be set when EnableCoordinator is true")
		}

		coord, err := NewCoordinatorClient(CoordinatorAddr, CoordinatorSequencerID)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize coordinator client: %v", err))
		}
		Coordinator = coord
	}
}

func CustomizeL1BaseFeeByTransactions(originBaseFee *big.Int, transactions types.Transactions) *big.Int {
	if EnableCustomizeL1BlockBaseFee {
		return calcAvgGasPriceByBlockTransactions(transactions)
	} else {
		return originBaseFee
	}
}

func CustomizeL1BlockInfoByReceipts(originInfo eth.BlockInfo, receipts types.Receipts) eth.BlockInfo {
	if EnableCustomizeL1BlockBaseFee {
		return &customizedBlockInfo{
			BlockInfo:   originInfo,
			avgGasPrice: calcAvgGasPriceByBlockReceipts(receipts),
		}
	} else {
		return originInfo
	}
}

func CustomizeSuggestedL1BaseFee(originBaseFee *big.Int) *big.Int {
	if EnableCustomizeSuggestedL1BaseFee && originBaseFee == nil {
		return DefaultCustomizedSuggestedL1BaseFee
	} else {
		return originBaseFee
	}
}

// CustomizeL1Label customize the L1 labels "safe" and "finalized" when the feature "EnableCustomizeL1Label" is enabled:
// - keep the "unsafe" label as it is
// - redefine the "safe" label and "finalized" to be the block with 15 blocks less than the "unsafe" block
func CustomizeL1Label(ctx context.Context, l1Source l1BlockInfoSource, label eth.BlockLabel) (eth.BlockInfo, error) {
	if EnableCustomizeL1Label && label != eth.Unsafe {
		unsafeInfo, err := l1Source.InfoByLabel(ctx, eth.Unsafe)
		if err != nil {
			return nil, err
		}

		if unsafeInfo.NumberU64() <= DefaultCustomizedL1LabelSub {
			return l1Source.InfoByNumber(ctx, 0)
		} else {
			return l1Source.InfoByNumber(ctx, unsafeInfo.NumberU64()-DefaultCustomizedL1LabelSub)
		}
	} else {
		return l1Source.InfoByLabel(ctx, label)
	}
}

func CustomizeCraftL1Transaction(dynRawTx *types.DynamicFeeTx) types.TxData {
	if EnableCustomizeCraftL1Transaction {
		return &types.LegacyTx{
			Nonce:    dynRawTx.Nonce,
			GasPrice: dynRawTx.GasFeeCap,
			Gas:      dynRawTx.Gas,
			To:       dynRawTx.To,
			Value:    dynRawTx.Value,
			Data:     dynRawTx.Data,
		}
	} else {
		return dynRawTx
	}
}

func CustomizeCraftL1CallMsg(callMsg ethereum.CallMsg) ethereum.CallMsg {
	if EnableCustomizeCraftL1Transaction {
		return ethereum.CallMsg{
			From:     callMsg.From,
			To:       callMsg.To,
			GasPrice: callMsg.GasFeeCap,
			Data:     callMsg.Data,
		}
	} else {
		return callMsg
	}
}

func CustomizeProposeL1BlockHash(originHash common.Hash) common.Hash {
	if EnableCustomizeProposeL1BlockHash {
		return DefaultCustomizedProposeL1BlockHash
	} else {
		return originHash
	}
}

// calcAvgGasPriceByBlockTransactions calculates the average gas price of the non-zero-gas-price transactions in the block.
// If there is no non-zero-gas-price transaction in the block, it returns DefaultCustomizedBaseFee.
func calcAvgGasPriceByBlockTransactions(transactions types.Transactions) *big.Int {
	nonZeroTxsCnt := big.NewInt(0)
	nonZeroTxsSum := big.NewInt(0)
	for _, tx := range transactions {
		if tx.GasPrice().Cmp(common.Big0) > 0 {
			nonZeroTxsCnt.Add(nonZeroTxsCnt, big.NewInt(1))
			nonZeroTxsSum.Add(nonZeroTxsSum, tx.GasPrice())
		}
	}

	if nonZeroTxsCnt.Cmp(big.NewInt(0)) == 0 {
		return DefaultCustomizedBaseFee
	}
	return nonZeroTxsSum.Div(nonZeroTxsSum, nonZeroTxsCnt)
}

// calcAvgGasPriceByBlockReceipts calculates the average gas price of the non-zero-gas-price transactions in the block.
// If there is no non-zero-gas-price transaction in the block, it returns DefaultCustomizedBaseFee.
func calcAvgGasPriceByBlockReceipts(receipts types.Receipts) *big.Int {
	nonZeroTxsCnt := big.NewInt(0)
	nonZeroTxsSum := big.NewInt(0)
	for _, tx := range receipts {
		if tx.L1GasPrice.Cmp(common.Big0) > 0 {
			nonZeroTxsCnt.Add(nonZeroTxsCnt, big.NewInt(1))
			nonZeroTxsSum.Add(nonZeroTxsSum, tx.L1GasPrice)
		}
	}

	if nonZeroTxsCnt.Cmp(big.NewInt(0)) == 0 {
		return DefaultCustomizedBaseFee
	}
	return nonZeroTxsSum.Div(nonZeroTxsSum, nonZeroTxsCnt)
}

// l1BlockInfoSource is a helper interface to avoid circular dependencies when we need to use op-node.source.EthClient
// to get block info.
type l1BlockInfoSource interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, num uint64) (eth.BlockInfo, error)
}

type customizedBlockInfo struct {
	eth.BlockInfo
	avgGasPrice *big.Int
}

var _ eth.BlockInfo = (*customizedBlockInfo)(nil)

func (c *customizedBlockInfo) BaseFee() *big.Int {
	return c.avgGasPrice
}
