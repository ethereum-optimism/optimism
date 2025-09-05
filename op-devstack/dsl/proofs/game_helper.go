package proofs

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

type GameHelperMove struct {
	ParentIdx *big.Int
	Claim     common.Hash
	Attack    bool
}

type contractArtifactData struct {
	Bytecode []byte
	ABI      abi.ABI
}

type GameHelper struct {
	t            devtest.T
	require      *require.Assertions
	contractAddr common.Address
	abi          abi.ABI
}

func DeployGameHelper(t devtest.T, deployer *dsl.EOA) *GameHelper {
	req := require.New(t)

	artifactData := getGameHelperArtifactData(t)

	constructorABI := artifactData.ABI

	encodedArgs, err := constructorABI.Pack("")
	req.NoError(err, "Failed to encode constructor arguments")

	deploymentData := append(artifactData.Bytecode, encodedArgs...)

	deployTxOpts := txplan.Combine(
		deployer.Plan(),
		txplan.WithData(deploymentData),
	)

	deployTx := txplan.NewPlannedTx(deployTxOpts)
	receipt, err := deployTx.Included.Eval(t.Ctx())
	req.NoError(err, "Failed to deploy GameHelper contract")

	req.Equal(types.ReceiptStatusSuccessful, receipt.Status, "GameHelper deployment failed")
	req.NotEqual(common.Address{}, receipt.ContractAddress, "GameHelper contract address not set in receipt")

	contractAddr := receipt.ContractAddress
	t.Logf("GameHelper contract deployed at: %s", contractAddr.Hex())

	return &GameHelper{
		t:            t,
		require:      require.New(t),
		contractAddr: contractAddr,
		abi:          artifactData.ABI,
	}
}

type ArtifactBytecode struct {
	Object string `json:"object"`
}

type ArtifactJSON struct {
	Bytecode ArtifactBytecode `json:"bytecode"`
	ABI      json.RawMessage  `json:"abi"`
}

func getGameHelperArtifactData(t devtest.T) *contractArtifactData {
	req := require.New(t)
	artifactPath := getGameHelperArtifactPath(t)

	fileData, err := os.ReadFile(artifactPath)
	req.NoError(err, "Failed to read GameHelper artifact file")

	var artifactJSON ArtifactJSON
	err = json.Unmarshal(fileData, &artifactJSON)
	req.NoError(err, "Failed to parse GameHelper artifact JSON")

	req.NotEmpty(artifactJSON.Bytecode.Object, "Bytecode object not found in GameHelper artifact")

	bytecode := common.FromHex(artifactJSON.Bytecode.Object)

	parsedABI, err := abi.JSON(bytes.NewReader(artifactJSON.ABI))
	req.NoError(err, "Failed to parse ABI")

	return &contractArtifactData{
		Bytecode: bytecode,
		ABI:      parsedABI,
	}
}

func getGameHelperArtifactPath(t devtest.T) string {
	req := require.New(t)
	wd, err := os.Getwd()
	req.NoError(err, "Failed to get current working directory")

	monorepoRoot, err := opservice.FindMonorepoRoot(wd)
	req.NoError(err, "Failed to find monorepo root")

	contractsBedrock := filepath.Join(monorepoRoot, "packages", "contracts-bedrock")
	return filepath.Join(contractsBedrock, "forge-artifacts", "GameHelper.sol", "GameHelper.json")
}

func (gs *GameHelper) AuthEOA(eoa *dsl.EOA) *GameHelper {
	tx := txplan.NewPlannedTx(eoa.PlanAuth(gs.contractAddr))
	receipt, err := tx.Included.Eval(gs.t.Ctx())
	gs.require.NoError(err)
	gs.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	return &GameHelper{
		t:            gs.t,
		require:      require.New(gs.t),
		contractAddr: eoa.Address(),
		abi:          gs.abi,
	}
}

func (gs *GameHelper) CreateGameWithClaims(
	eoa *dsl.EOA,
	factory *DisputeGameFactory,
	gameType challengerTypes.GameType,
	rootClaim common.Hash,
	extraData []byte,
	moves []GameHelperMove,
) common.Address {
	data, err := gs.abi.Pack("createGameWithClaims", factory.Address(), gameType, rootClaim, extraData, moves)
	gs.require.NoError(err)

	gameImpl := factory.gameImpl(gameType)
	bonds := factory.initBond(gameType)
	bonds = bonds.Add(gs.totalMoveBonds(gameImpl, moves))

	tx := txplan.NewPlannedTx(
		txplan.Combine(
			eoa.Plan(),
			txplan.WithValue(bonds),
			txplan.WithTo(&gs.contractAddr),
			txplan.WithData(data),
		),
	)
	receipt, err := tx.Included.Eval(gs.t.Ctx())
	gs.require.NoError(err)
	gs.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return receipt.ContractAddress
}

func (gs *GameHelper) PerformMoves(eoa *dsl.EOA, game *FaultDisputeGame, moves []GameHelperMove) []*Claim {
	data, err := gs.abi.Pack("performMoves", game.Address, moves)
	gs.require.NoError(err)

	tx := txplan.NewPlannedTx(
		txplan.Combine(
			eoa.Plan(),
			txplan.WithValue(gs.totalMoveBonds(game, moves)),
			txplan.WithTo(&gs.contractAddr),
			txplan.WithData(data),
		),
	)
	preClaimCount := game.claimCount()
	receipt, err := tx.Included.Eval(gs.t.Ctx())
	gs.require.NoError(err)
	gs.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	postClaimCount := game.claimCount()

	// While all claims are performed within one transaction, it's possible another transaction also added claims
	// between the calls to get claim count above (e.g. by a challenger running in parallel).
	// So iterate to find the claims we added rather than just assuming the claim indices.
	// Assumes that claims added by this helper contract are only added by this thread,
	// which is safe because we deployed this particular instance of GameHelper.
	claims := make([]*Claim, 0, len(moves))
	for claimIdx := preClaimCount; claimIdx < postClaimCount; claimIdx++ {
		claim := game.ClaimAtIndex(claimIdx)
		if claim.claim.Claimant != gs.contractAddr {
			continue
		}
		claims = append(claims, claim)
	}
	gs.require.Equal(len(claims), len(moves), "Did not find claims for all moves")
	return claims
}

func (gs *GameHelper) totalMoveBonds(game *FaultDisputeGame, moves []GameHelperMove) eth.ETH {
	claimPositions := map[uint64]challengerTypes.Position{
		0: challengerTypes.RootPosition,
	}
	totalBond := eth.Ether(0)
	for i, move := range moves {
		parentPos := claimPositions[move.ParentIdx.Uint64()]
		gs.require.NotEmpty(parentPos, "Move references non-existent parent - may be out of order")
		childPos := parentPos.Defend()
		if move.Attack {
			childPos = parentPos.Attack()
		}
		claimPositions[uint64(i)+1] = childPos
		bond := game.requiredBond(childPos)
		totalBond = totalBond.Add(bond)
	}
	return totalBond
}

func Move(parentIdx int64, claim common.Hash, attack bool) GameHelperMove {
	return GameHelperMove{
		ParentIdx: big.NewInt(parentIdx),
		Claim:     claim,
		Attack:    attack,
	}
}
