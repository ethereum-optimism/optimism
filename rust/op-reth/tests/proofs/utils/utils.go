package utils

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// minimal parts of artifact
type Artifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

// LoadArtifact reads the forge artifact JSON at artifactPath and returns the parsed ABI
// and the creation bytecode (as bytes). It prefers bytecode.object (creation) and falls
// back to deployedBytecode.object if needed.
func LoadArtifact(t devtest.T, artifactPath string) (abi.ABI, []byte) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		require.NoError(t, err, "failed to read artifact file")
	}

	var art Artifact
	if err := json.Unmarshal(data, &art); err != nil {
		require.NoError(t, err, "failed to unmarshal artifact JSON")
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(art.ABI)))
	if err != nil {
		require.NoError(t, err, "failed to parse contract ABI")
	}

	binHex := strings.TrimSpace(art.Bytecode.Object)
	if binHex == "" {
		require.NoError(t, err, "artifact has no bytecode")
	}

	return parsedABI, common.FromHex(binHex)
}

// WaitForProofsStoreBlock polls the op-reth debug_proofsSyncStatus RPC until the
// proofs ExEx store has indexed at least up to targetBlock. The ExEx processes
// ChainCommitted notifications asynchronously, so the EL head can advance before
// the proofs store has caught up. Any RPC that depends on OpStateProviderFactory
// (e.g. debug_executePayload) will fail with "no state found" if called before
// the store is ready.
func WaitForProofsStoreBlock(t devtest.T, client apis.EthClient, targetBlock uint64) {
	type syncStatus struct {
		Earliest *uint64 `json:"earliest"`
		Latest   *uint64 `json:"latest"`
	}
	require.Eventually(t, func() bool {
		var status syncStatus
		err := client.RPC().CallContext(t.Ctx(), &status, "debug_proofsSyncStatus")
		if err != nil {
			t.Logf("debug_proofsSyncStatus call failed (retrying): %v", err)
			return false
		}
		if status.Latest == nil {
			t.Logf("proofs store not yet initialized, waiting...")
			return false
		}
		t.Logf("proofs store status: latest=%d target=%d", *status.Latest, targetBlock)
		return *status.Latest >= targetBlock
	}, 30*time.Second, 200*time.Millisecond, "proofs store did not index block %d in time", targetBlock)
}

// DeployContract deploys the contract creation bytecode from the given artifact.
// user must provide a Plan() method compatible with txplan.NewPlannedTx (kept generic).
func DeployContract(t devtest.T, user *dsl.EOA, bin []byte) (common.Address, *types.Receipt) {
	tx := txplan.NewPlannedTx(user.Plan(), txplan.WithData(bin))
	res, err := tx.Included.Eval(t.Ctx())
	if err != nil {
		require.NoError(t, err, "contract deployment tx failed")
	}

	if res.Status != types.ReceiptStatusSuccessful {
		require.NoError(t, err, "contract deployment transaction failed")
	}

	return res.ContractAddress, res
}
