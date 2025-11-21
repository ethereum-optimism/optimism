package utils

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
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

// IdentifyELNodes returns the reth and geth EL nodes based on their IDs.
func IdentifyELNodes(el *dsl.L2ELNode, elB *dsl.L2ELNode) (opRethELNode *dsl.L2ELNode, opGethELNode *dsl.L2ELNode) {
	if strings.Contains(el.ID().Key(), "op-reth") {
		return el, elB
	}
	return elB, el
}

// IdentifyCLNodes returns the reth and geth CL nodes based on their IDs.
func IdentifyCLNodes(cl *dsl.L2CLNode, clB *dsl.L2CLNode) (opRethCLNode *dsl.L2CLNode, opGethCLNode *dsl.L2CLNode) {
	if strings.Contains(cl.ID().Key(), "op-reth") {
		return cl, clB
	}
	return clB, cl
}
