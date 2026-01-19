package proofs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
	t                   devtest.T
	require             *require.Assertions
	contractAddr        common.Address
	abi                 abi.ABI
	honestTraceProvider func(game *FaultDisputeGame) challengerTypes.TraceAccessor
}

func DeployGameHelper(t devtest.T, deployer *dsl.EOA, honestTraceProvider func(game *FaultDisputeGame) challengerTypes.TraceAccessor) *GameHelper {
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
		t:                   t,
		require:             require.New(t),
		contractAddr:        contractAddr,
		abi:                 artifactData.ABI,
		honestTraceProvider: honestTraceProvider,
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
		t:                   gs.t,
		require:             require.New(gs.t),
		contractAddr:        eoa.Address(),
		abi:                 gs.abi,
		honestTraceProvider: gs.honestTraceProvider,
	}
}

func (gs *GameHelper) CreateGameWithClaims(
	eoa *dsl.EOA,
	factory *DisputeGameFactory,
	gameType gameTypes.GameType,
	rootClaim common.Hash,
	extraData []byte,
	moves []GameHelperMove,
) common.Address {
	data, err := gs.abi.Pack("createGameWithClaims", factory.Address(), gameType, rootClaim, extraData, moves)
	gs.require.NoError(err)

	gameImpl := factory.GameImpl(gameType)
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

func (gs *GameHelper) DisputeL2SequenceNumber(eoa *dsl.EOA, game *FaultDisputeGame, startClaim *Claim, l2SequenceNumber uint64) *Claim {
	splitDepth := game.SplitDepth()
	startingSeqNumber := game.StartingL2SequenceNumber()
	gs.require.Greater(l2SequenceNumber, startingSeqNumber, "Cannot dispute things at or prior to the starting block")
	seqNumAtPosition := func(pos challengerTypes.Position) uint64 {
		return pos.TraceIndex(splitDepth).Uint64() + startingSeqNumber + 1
	}
	shouldMoveLeftFrom := func(pos challengerTypes.Position) bool {
		// Move left when equal to the sequence number so that we disagree with it
		return seqNumAtPosition(pos) >= l2SequenceNumber
	}
	finalClaim, gameState := gs.disputeTo(eoa, game, startClaim, splitDepth, shouldMoveLeftFrom)

	// Check that we landed in the right place
	// We can only land on every second sequence number, starting from startingSeqNumber+1
	// And we want to land on the sequence number that's either equal to or one before l2SequenceNumber
	// If it's equal to we would attack, if it's the one before we would defend to ensure the bottom
	// half of the game is executing l2SequenceNumber.
	finalPosition := seqNumAtPosition(finalClaim.Position())
	if l2SequenceNumber%2 == startingSeqNumber%2 {
		gs.require.Equal(l2SequenceNumber-1, finalPosition)
		// When defending the required status code depends on whether we provided the next trace or not.
		disputedClaim, found := gameState.AncestorWithTraceIndex(finalClaim.asChallengerClaim(), finalClaim.Position().MoveRight().TraceIndex(game.MaxDepth()))
		gs.require.True(found, "Did not find ancestor at target trace index")
		statusCode := byte(0x00)
		if disputedClaim.Position.Depth()%2 == splitDepth%2 {
			statusCode = byte(0x01)
		}
		gs.t.Logf("Defend at split depth with status code %v", statusCode)
		finalClaim = finalClaim.Defend(eoa, common.Hash{statusCode, 0xba, 0xd0})
	} else {
		gs.t.Log("Attack at split depth")
		gs.require.Equal(l2SequenceNumber, finalPosition)
		// When attacking, the final block must be invalid so always use 0x01 as status code
		finalClaim = finalClaim.Attack(eoa, common.Hash{0x01, 0xba, 0xd0})
	}
	return finalClaim
}

func (gs *GameHelper) DisputeToStep(eoa *dsl.EOA, game *FaultDisputeGame, startClaim *Claim, traceIndex uint64) *Claim {
	splitDepth := game.SplitDepth()
	maxDepth := game.MaxDepth()
	if startClaim.Depth() < splitDepth {
		gs.require.Greater(startClaim.Depth(), splitDepth, "Start claim must be past the game split depth")
	}
	traceIndexAtPosition := func(pos challengerTypes.Position) uint64 {
		relativeFinalPosition, err := pos.RelativeToAncestorAtDepth(splitDepth + 1)
		gs.require.NoError(err, "Failed to calculate relative position")
		return relativeFinalPosition.TraceIndex(maxDepth - splitDepth - 1).Uint64()
	}
	shouldMoveLeftFrom := func(pos challengerTypes.Position) bool {
		// Move left when equal to the trace index so that we disagree with it
		return traceIndexAtPosition(pos) >= traceIndex
	}
	finalClaim, _ := gs.disputeTo(eoa, game, startClaim, maxDepth, shouldMoveLeftFrom)
	// Check that we landed in the right place
	// We can only land on every second sequence number, starting from startingSeqNumber+1
	// And we want to land on the sequence number that's either equal to or one before l2SequenceNumber
	// If it's equal to we would attack, if it's the one before we would defend to ensure the bottom
	// half of the game is executing l2SequenceNumber.
	finalTraceIndex := traceIndexAtPosition(finalClaim.Position())
	if traceIndex%2 == 1 {
		gs.require.Equal(traceIndex-1, finalTraceIndex)
	} else {
		gs.require.Equal(traceIndex, finalTraceIndex)
	}
	return finalClaim
}

func (gs *GameHelper) disputeTo(eoa *dsl.EOA, game *FaultDisputeGame, startClaim *Claim, targetDepth challengerTypes.Depth, shouldMoveLeftFrom func(pos challengerTypes.Position) bool) (*Claim, challengerTypes.Game) {
	honestTrace := gs.honestTraceProvider(game)
	maxDepth := game.MaxDepth()
	parentIdx := int64(startClaim.Index)
	moves := make([]GameHelperMove, 0, targetDepth)
	currentPos := startClaim.Position()
	claims := allChallengerClaims(game)
	gameState := challengerTypes.NewGameState(claims, maxDepth)
	honestRootClaim, err := honestTrace.Get(gs.t.Ctx(), gameState, claims[0], challengerTypes.RootPosition)
	gs.require.NoError(err, "Failed to get honest root claim")
	agreeWithRoot := claims[0].Value == honestRootClaim
	for currentPos.Depth() < targetDepth {
		shouldAttack := shouldMoveLeftFrom(currentPos)
		nextPos := currentPos.Defend()
		if shouldAttack {
			nextPos = currentPos.Attack()
		}
		claimValue := common.Hash{0xba, 0xd0}
		gs.t.Logf("Disputing claim %v at depth %v with claim at depth %v", parentIdx, currentPos.Depth(), nextPos.Depth())
		if !shouldMoveLeftFrom(nextPos) || !gameState.AgreeWithClaimLevel(claims[len(claims)-1], agreeWithRoot) {
			// Either we needed the honest actor to move right (defend) after this move or we are the honest actor
			// so make sure we use an honest claim value.
			value, err := honestTrace.Get(gs.t.Ctx(), gameState, startClaim.asChallengerClaim(), nextPos)
			gs.require.NoError(err, "Failed to get trace value at position %v", nextPos)
			claimValue = value
		}
		nextMove := Move(parentIdx, claimValue, shouldAttack)

		moves = append(moves, nextMove)
		claims = append(claims, challengerTypes.Claim{
			ClaimData: challengerTypes.ClaimData{
				Value:    nextMove.Claim,
				Bond:     big.NewInt(0),
				Position: nextPos,
			},
			Claimant:            eoa.Address(),
			Clock:               challengerTypes.Clock{},
			ContractIndex:       int(parentIdx + 1),
			ParentContractIndex: int(parentIdx),
		})
		gameState = challengerTypes.NewGameState(claims, maxDepth)
		currentPos = nextPos
		parentIdx++
	}
	addedClaims := gs.PerformMoves(eoa, game, moves)
	return addedClaims[len(addedClaims)-1], gameState
}

func allChallengerClaims(game *FaultDisputeGame) []challengerTypes.Claim {
	claims := game.allClaims()
	challengerClaims := make([]challengerTypes.Claim, len(claims))
	for i, claim := range claims {
		challengerClaims[i] = game.newClaim(uint64(i), claim).asChallengerClaim()
	}
	return challengerClaims
}

func (gs *GameHelper) PerformMoves(eoa *dsl.EOA, game *FaultDisputeGame, moves []GameHelperMove) []*Claim {
	gs.t.Log("Performing moves: \n" + describeMoves(moves))
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

func describeMoves(moves []GameHelperMove) string {
	description := ""
	for _, move := range moves {
		moveType := "Defend"
		if move.Attack {
			moveType = "Attack"
		}
		description += fmt.Sprintf("%s claim %s, value %s\n", moveType, move.ParentIdx, move.Claim)
	}
	return description
}

func (gs *GameHelper) totalMoveBonds(game *FaultDisputeGame, moves []GameHelperMove) eth.ETH {
	claimPositions := map[uint64]challengerTypes.Position{
		0: challengerTypes.RootPosition, // The claim at index 0 is always in the root position
	}
	preExistingClaimCount := game.claimCount()
	totalBond := eth.Ether(0)
	for i, move := range moves {
		parentPos := claimPositions[move.ParentIdx.Uint64()]
		if parentPos == (challengerTypes.Position{}) {
			gs.require.LessOrEqual(move.ParentIdx.Uint64(), preExistingClaimCount, "No parent position found - moves may be out of order")
			// Handle cases were there are existing claims and we're adding moves that reference them
			gs.t.Logf("Loading parent position for existing claim at index %v", move.ParentIdx)
			parentClaim := game.ClaimAtIndex(move.ParentIdx.Uint64())
			parentPos = parentClaim.Position()
			claimPositions[move.ParentIdx.Uint64()] = parentPos
		}
		childPos := parentPos.Defend()
		if move.Attack {
			childPos = parentPos.Attack()
		}
		claimPositions[uint64(i)+preExistingClaimCount] = childPos
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
