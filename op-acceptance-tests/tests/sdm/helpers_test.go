package sdm

import (
	"encoding/json"
	"fmt"
	"strings"

	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/lmittmann/w3"
)

// ComputeHeavy: run(uint256 n) loops keccak256 n times (pure computation).
const computeHeavyBin = "6080604052348015600e575f5ffd5b506101908061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c8063a444f5e91461002d575b5f5ffd5b610047600480360381019061004291906100ec565b610049565b005b5f7f66a80b61b29ec044d14c4c8c613e762ba1fb8eeb0c454d1ee00ed6dedaa5b5c590505f5f90505b828110156100b0578160405160200161008b9190610140565b6040516020818303038152906040528051906020012091508080600101915050610072565b505050565b5f5ffd5b5f819050919050565b6100cb816100b9565b81146100d5575f5ffd5b50565b5f813590506100e6816100c2565b92915050565b5f60208284031215610101576101006100b5565b5b5f61010e848285016100d8565b91505092915050565b5f819050919050565b5f819050919050565b61013a61013582610117565b610120565b82525050565b5f61014b8284610129565b6020820191508190509291505056fea264697066735822122013cd314931f1991e7797e220c9553bb73dfef407d4d266dd8b2553907d5bc14364736f6c634300081c0033"

var (
	funcEmitLog = w3.MustNewFunc("emitLog(bytes32[],bytes)", "")
)

// setSDMEnabled toggles the local SDM PostExec production opt-in on an L2 EL via the
// admin_setSdmEnabled RPC. SDM is disabled by default on every process boot; tests that
// expect PostExec txs to flow must opt in explicitly on the sequencer's EL.
func setSDMEnabled(t devtest.T, l2EL *dsl.L2ELNode, enabled bool) {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	err := rpcClient.CallContext(t.Ctx(), nil, "admin_setSdmEnabled", enabled)
	t.Require().NoError(err, "admin_setSdmEnabled(%v) RPC failed", enabled)
}

// verifyOpReth checks the L2 execution layer client is op-reth by calling
// web3_clientVersion via the L2EthClient's RPC and asserting it contains "reth".
func verifyOpReth(t devtest.T, l2EL *dsl.L2ELNode) string {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var clientVersion string
	err := rpcClient.CallContext(t.Ctx(), &clientVersion, "web3_clientVersion")
	t.Require().NoError(err, "web3_clientVersion RPC failed — cannot verify EL client")

	lower := strings.ToLower(clientVersion)
	t.Require().True(
		strings.Contains(lower, "reth"),
		"FATAL: Expected op-reth execution client, but got: %q. "+
			"This test MUST run on op-reth. "+
			"Set DEVSTACK_L2EL_KIND=op-reth or ensure op-reth binary is available.",
		clientVersion,
	)
	t.Require().False(
		strings.Contains(lower, "geth"),
		"FATAL: Detected op-geth (%q) but this test requires op-reth.", clientVersion,
	)

	return clientVersion
}

// getOPGasRefund reads the opGasRefund field from a transaction receipt via
// raw JSON RPC. The boolean return value reports whether the field was present.
func getOPGasRefund(t devtest.T, l2EL *dsl.L2ELNode, txHash common.Hash) (uint64, bool) {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var raw json.RawMessage
	err := rpcClient.CallContext(t.Ctx(), &raw, "eth_getTransactionReceipt", txHash)
	t.Require().NoError(err, "eth_getTransactionReceipt RPC failed for tx %s", txHash)
	t.Require().NotNil(raw, "receipt %s not found", txHash)

	var result struct {
		OPGasRefund *hexutil.Uint64 `json:"opGasRefund"`
	}
	err = json.Unmarshal(raw, &result)
	t.Require().NoError(err, "failed to unmarshal receipt %s", txHash)
	if result.OPGasRefund == nil {
		return 0, false
	}
	return uint64(*result.OPGasRefund), true
}

func getBlockWithTxs(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) *sdmpkg.RPCBlock {
	block, err := sdmpkg.GetBlockWithTxs(t.Ctx(), l2EL.Escape().L2EthClient().RPC(), blockNum)
	t.Require().NoError(err, "eth_getBlockByNumber RPC failed for block %d", blockNum)
	return block
}

func findPostExecTransaction(block *sdmpkg.RPCBlock) (*sdmpkg.RPCTransaction, int) {
	return sdmpkg.FindPostExecTransaction(block)
}

func deployContract(t devtest.T, eoa *dsl.EOA, hexBytecode string) common.Address {
	tx := txplan.NewPlannedTx(eoa.Plan(), txplan.WithData(common.FromHex(hexBytecode)))
	res, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "failed to deploy contract")
	return res.ContractAddress
}

func encodeEmitLog(topicCount int, dataLen int) []byte {
	topics := make([][32]byte, topicCount)
	for i := range topics {
		topics[i] = [32]byte{byte(i + 1)}
	}
	opaqueData := make([]byte, dataLen)
	for i := range opaqueData {
		opaqueData[i] = byte(i % 256)
	}
	data, err := funcEmitLog.EncodeArgs(topics, opaqueData)
	if err != nil {
		panic(fmt.Sprintf("failed to encode emitLog: %v", err))
	}
	return data
}
