package zk

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func TestDeploymentUsesSuperAggregationVKey(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t, presets.WithZK())
	vkey := expectedSuperAggregationVKey(t)
	factory := sys.DisputeGameFactory()

	factory.VerifyGameImplAbsent(gameTypes.SuperCannonKonaGameType)
	zk := factory.ZKGameImpl()
	t.Require().NotEqual(common.Address{}, zk.Address)
	t.Require().Equal(vkey, zk.Args.AbsolutePrestate)
	t.Require().Equal(uint64(presets.DefaultZKChallengeDuration/time.Second), zk.Args.MaxChallengeDuration)
	t.Require().Equal(uint64(presets.DefaultZKProveDuration/time.Second), zk.Args.MaxProveDuration)
	t.Require().Positive(zk.Args.ChallengerBond.Sign())
	t.Require().NotEqual(common.Address{}, zk.Args.AnchorStateRegistry)
	t.Require().NotEqual(common.Address{}, zk.Args.Weth)
	l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	code, err := sys.L1EL.EthClient().CodeAtHash(t.Ctx(), zk.Args.Verifier, l1Head.Hash)
	t.Require().NoError(err)
	t.Require().NotEmpty(code, "mock verifier must have deployed code")
}

func TestChallengedValidProposalAnchors(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t, presets.WithZK())
	factory := sys.DisputeGameFactory()
	challenger, _ := fundedActors(sys)

	// The honest proposer creates the valid root proposal.
	game := factory.WaitForZKGameAtIndex(0)
	t.Require().Equal(uint32(math.MaxUint32), game.ParentIndex())
	t.Require().Equal(proofs.ZKProposalUnchallenged, game.ProposalStatus())

	// A third party grief-challenges the valid proposal; the honest challenger does not challenge it.
	challengedClaim := game.Challenge(challenger)
	t.Require().Equal(challenger.Address(), challengedClaim.Challenger)

	// The kona-sp1-proposer detects the challenge and defends its own game;
	// the proof commits to the submitting signer.
	game.WaitForProposalStatus(proofs.ZKProposalChallengedAndValidProofProvided)
	t.Require().Equal(zkProposerAddress(t, sys), game.ClaimData().Prover)

	// The proven-valid proposal resolves and anchors. The live proposer keeps
	// chaining and anchoring later games, so assert the anchor reached at
	// least this game's sequence (descendants can only anchor if this game
	// resolved in the defender's favor).
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRootAtLeast(game)
}

// TestProposerDefendsForeignValidGame proves, resolves, and claims prover
// credit on a challenged valid game that the proposer did not create.
func TestProposerDefendsForeignValidGame(gt *testing.T) {
	t := devtest.SerialT(gt)
	// The honest challenger resolves games and claims credit on the
	// proposer's behalf; disable it so every assertion below binds to the
	// proposer.
	sys := presets.NewSimpleInterop(t, presets.WithZK(), presets.WithoutHonestChallenger())
	factory := sys.DisputeGameFactory()
	proposerAddr := zkProposerAddress(t, sys)
	weth := factory.DelayedWETH(factory.ZKGameImpl().Args.Weth)
	creator, challenger := fundedActors(sys)

	// The DSL derives a valid claim from the supernode. Use an off-grid
	// timestamp to avoid a factory UUID collision with the proposer's game,
	// which would have the same claim and extraData.
	proposerGame := factory.WaitForZKGameAtIndex(0)
	foreignTimestamp := proposerGame.L2SequenceNumber() + 1
	factory.WaitForSafeSuperRootAfter(foreignTimestamp)
	game := factory.StartZKGame(creator, proofs.WithL2SequenceNumber(foreignTimestamp))
	challengerBond := game.ChallengerBond()
	challengedClaim := game.Challenge(challenger)
	t.Require().Equal(challenger.Address(), challengedClaim.Challenger)

	// The proposer's defense set is prestate-based, not creator-based: it
	// proves the foreign game and its resolution task resolves it; the test
	// never calls prove or resolve itself.
	game.WaitForProposalStatus(proofs.ZKProposalChallengedAndValidProofProvided)
	t.Require().Equal(proposerAddr, game.ClaimData().Prover)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)

	// The proposer's claim task unlocks its prover credit - the challenger's
	// bond - into DelayedWETH. Asserting the unlocked amount is race-free:
	// the payout cannot run until L1 time advances past the WETH delay.
	var withdrawal proofs.ZKWithdrawal
	t.Require().Eventuallyf(func() bool {
		withdrawal = weth.Withdrawal(game.Address, proposerAddr)
		return withdrawal.Amount.Sign() > 0
	}, 2*time.Minute, time.Second, "proposer did not unlock its prover credit")
	t.Require().Equal(challengerBond, eth.WeiBig(withdrawal.Amount),
		"prover credit must equal the challenger bond")

	advanceL1To(&sys.SingleChainInterop, withdrawal.MaturesAt(weth.Delay()))
	game.WaitForClaimedCredit(proposerAddr)
}

// TestProposerIgnoresInvalidChallengedGame pins the defense boundary: the
// proposer defends only games whose claims re-derive against its supernode.
// A foreign proposal with a corrupted output root is challenged, and the
// proposer must never submit a proof for it - even though the game carries
// the prestate it can prove - so the challenge wins at the deadline by
// forfeit.
func TestProposerIgnoresInvalidChallengedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// The honest challenger would race the test's own challenge; disable it
	// so the scenario stays deterministic.
	sys := presets.NewSimpleInterop(t, presets.WithZK(), presets.WithoutHonestChallenger())
	factory := sys.DisputeGameFactory()
	creator, challenger := fundedActors(sys)

	// The honest proposer is live before the invalid game exists.
	factory.WaitForZKGameAtIndex(0)

	// A foreign EOA proposes a corrupted super root at an already-safe
	// timestamp (the honest-challenger invalid-proposal recipe), and a
	// second EOA challenges it.
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	game := factory.StartZKGame(creator,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	game.Challenge(challenger)

	holdCtx, cancelHold := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancelHold()
	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()
	holding := true
	for holding {
		select {
		case <-holdCtx.Done():
			t.Require().NoError(t.Ctx().Err(), "test context ended while checking invalid game")
			holding = false
		case <-pollTicker.C:
			claim, err := game.WaitForClaimData(holdCtx)
			if holdCtx.Err() != nil {
				t.Require().NoError(t.Ctx().Err(), "test context ended while checking invalid game")
				holding = false
				continue
			}
			t.Require().NoError(err, "read claim data while checking invalid game")
			t.Require().Equal(proofs.ZKProposalChallenged, proofs.ZKProposalStatus(claim.Status),
				"proposer must not defend an invalid game")
			t.Require().Equal(common.Address{}, claim.Prover,
				"proposer must not prove an invalid game")
		}
	}

	// With no proof by the deadline, the challenger wins by forfeit.
	// Resolution is permissionless; the test resolves since the honest
	// challenger is disabled and the proposer does not own the game.
	advanceL1To(&sys.SingleChainInterop, game.ClaimData().Deadline+1)
	game.Resolve(challenger)
	t.Require().Equal(gameTypes.GameStatusChallengerWon, game.GameStatus())
	t.Require().Equal(common.Address{}, game.ClaimData().Prover,
		"an invalid game must never be proven by the proposer")
}

// TestProposerDefendsMultipleChallengedGamesConcurrently proves two foreign
// valid games in parallel. The spawned-task metric must reach two before
// either proof lands; transaction landing times are serialized by the signer
// lock and do not measure proving concurrency.
func TestProposerDefendsMultipleChallengedGamesConcurrently(gt *testing.T) {
	t := devtest.ParallelT(gt)
	metricsPort := freeTCPPort(t)
	// The honest challenger resolves games and claims credit on the
	// proposer's behalf; disable it so proof submission is attributable to
	// the proposer alone.
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithZKProposerOption(sysgo.WithZKMetricsPort(metricsPort)),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
	)
	factory := sys.DisputeGameFactory()
	proposerAddr := zkProposerAddress(t, sys)
	creator, challenger := fundedActors(sys)

	// Create two foreign valid root games at distinct safe timestamps.
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	timestampA, _ := factory.WaitForSafeSuperRootAfter(anchorSequence)
	timestampB, _ := factory.WaitForSafeSuperRootAfter(timestampA)
	gameA := factory.StartZKGame(creator, proofs.WithL2SequenceNumber(timestampA))
	gameB := factory.StartZKGame(creator, proofs.WithL2SequenceNumber(timestampB))

	// Start the proposer after both challenges are on-chain so its first
	// defense scan sees both candidates.
	gameA.Challenge(challenger)
	gameB.Challenge(challenger)
	sys.StartZKProposer()

	metricsURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", metricsPort)
	const spawnedMetric = "kona_sp1_proposer_games_defense_spawned"
	const provingErrorMetric = "kona_sp1_proposer_game_proving_error"
	sawConcurrentDefense := false
	var proverA, proverB common.Address
	pollDeadline := time.Now().Add(10 * time.Minute)
	for (proverA == common.Address{} || proverB == common.Address{}) && time.Now().Before(pollDeadline) {
		if proverA == (common.Address{}) {
			claim, err := gameA.WaitForClaimData(t.Ctx())
			t.Require().NoError(err, "read game A claim data")
			proverA = claim.Prover
		}
		if proverB == (common.Address{}) {
			claim, err := gameB.WaitForClaimData(t.Ctx())
			t.Require().NoError(err, "read game B claim data")
			proverB = claim.Prover
		}
		// Two spawned tasks before either proof lands witnesses concurrency.
		// Zero failures excludes one failed and respawned task.
		if (proverA == common.Address{}) && (proverB == common.Address{}) &&
			scrapeMetric(metricsURL, spawnedMetric) == 2 &&
			scrapeMetric(metricsURL, provingErrorMetric) == 0 {
			sawConcurrentDefense = true
		}
		time.Sleep(time.Second)
	}
	t.Require().Equal(proposerAddr, proverA, "game A was not proven by the proposer in time")
	t.Require().Equal(proposerAddr, proverB, "game B was not proven by the proposer in time")
	t.Require().True(sawConcurrentDefense,
		"both defense tasks must be live before either proof lands (parallel proving)")
}

func freeTCPPort(t devtest.T) uint16 {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	t.Require().NoError(err, "reserve an ephemeral port")
	defer listener.Close()
	return uint16(listener.Addr().(*net.TCPAddr).Port)
}

// scrapeMetric returns the first matching Prometheus sample, or 0 while the
// endpoint or sample is unavailable.
func scrapeMetric(url, name string) float64 {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, name) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		return value
	}
	return 0
}

func fundedActors(sys *presets.SimpleInterop) (*dsl.EOA, *dsl.EOA) {
	actors := sys.FunderL1.NewFundedEOAs(2, eth.OneEther)
	return actors[0], actors[1]
}

func advanceL1To(sys *presets.SingleChainInterop, timestamp uint64) {
	current := sys.L1EL.BlockRefByLabel(eth.Unsafe).Time
	if current < timestamp {
		sys.AdvanceTime(time.Duration(timestamp-current) * time.Second)
	}
	sys.L1EL.WaitForTime(timestamp)
}
