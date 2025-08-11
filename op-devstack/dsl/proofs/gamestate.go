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

type GameStateMove struct {
	ParentIdx *big.Int
	Claim     common.Hash
	Attack    bool
}

type GameStateArtifactData struct {
	Bytecode []byte
	ABI      abi.ABI
}

type GameState struct {
	t            devtest.T
	require      *require.Assertions
	contractAddr common.Address
	abi          abi.ABI
}

func DeployGameState(t devtest.T, deployer *dsl.EOA) *GameState {
	req := require.New(t)

	artifactData := getGameStateArtifactData(t)

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
	req.NoError(err, "Failed to deploy GameState contract")

	req.Equal(types.ReceiptStatusSuccessful, receipt.Status, "GameState deployment failed")
	req.NotEqual(common.Address{}, receipt.ContractAddress, "GameState contract address not set in receipt")

	contractAddr := receipt.ContractAddress
	t.Logf("GameState contract deployed at: %s", contractAddr.Hex())

	return &GameState{
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

func getGameStateArtifactData(t devtest.T) *GameStateArtifactData {
	req := require.New(t)
	artifactPath := getGameStateArtifactPath(t)

	fileData, err := os.ReadFile(artifactPath)
	req.NoError(err, "Failed to read GameState artifact file")

	var artifactJSON ArtifactJSON
	err = json.Unmarshal(fileData, &artifactJSON)
	req.NoError(err, "Failed to parse GameState artifact JSON")

	req.NotEmpty(artifactJSON.Bytecode.Object, "Bytecode object not found in GameState artifact")

	bytecode := common.FromHex(artifactJSON.Bytecode.Object)

	parsedABI, err := abi.JSON(bytes.NewReader(artifactJSON.ABI))
	req.NoError(err, "Failed to parse ABI")

	return &GameStateArtifactData{
		Bytecode: bytecode,
		ABI:      parsedABI,
	}
}

func getGameStateArtifactPath(t devtest.T) string {
	req := require.New(t)
	wd, err := os.Getwd()
	req.NoError(err, "Failed to get current working directory")

	monorepoRoot, err := opservice.FindMonorepoRoot(wd)
	req.NoError(err, "Failed to find monorepo root")

	contractsBedrock := filepath.Join(monorepoRoot, "packages", "contracts-bedrock")
	return filepath.Join(contractsBedrock, "forge-artifacts", "GameState.sol", "GameState.json")
}

func (gs *GameState) CreateGameWithClaims(
	eoa *dsl.EOA,
	factory *DisputeGameFactory,
	gameType challengerTypes.GameType,
	rootClaim common.Hash,
	extraData []byte,
	moves []GameStateMove,
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

func (gs *GameState) PerformMoves(eoa *dsl.EOA, game *FaultDisputeGame, moves []GameStateMove) {
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
	receipt, err := tx.Included.Eval(gs.t.Ctx())
	gs.require.NoError(err)
	gs.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

func (gs *GameState) totalMoveBonds(game *FaultDisputeGame, moves []GameStateMove) eth.ETH {
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

func Move(parentIdx int64, claim common.Hash, attack bool) GameStateMove {
	return GameStateMove{
		ParentIdx: big.NewInt(parentIdx),
		Claim:     claim,
		Attack:    attack,
	}
}
