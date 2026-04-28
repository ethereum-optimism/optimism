package sysgo

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const preGenesisRollupStartBlockDelay = uint64(6)

var preGenesisStartingAnchorRoot = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000042")

func preparePreGenesisSuperGame(
	t devtest.T,
	keys devkeys.Keys,
	wb *worldBuilder,
	l1Net *L1Network,
	l1EL *L1Geth,
	migration *interopMigrationState,
	gameCfg *PreGenesisSuperGameConfig,
	l2Nets ...*L2Network,
) {
	require := t.Require()
	require.NotNil(gameCfg, "pre-genesis super game config is required")
	require.NotEmpty(l2Nets, "at least one L2 network is required")
	require.Len(gameCfg.ClaimedOutputs, len(l2Nets), "claimed outputs must match the number of L2 networks")

	blockTime := l2Nets[0].RollupConfig().BlockTime
	for _, l2Net := range l2Nets[1:] {
		require.Equal(blockTime, l2Net.RollupConfig().BlockTime,
			"pre-genesis super game only supports identical L2 block times")
	}

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err, "failed to connect to L1 RPC")
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)
	rpcWrapper := opclient.NewBaseRPCClient(rpcClient)
	ethClient, err := sources.NewEthClient(rpcWrapper, t.Logger(), nil, sources.DefaultEthClientConfig(10))
	require.NoError(err, "failed to create L1 eth client")

	latestL1Header, err := client.HeaderByNumber(t.Ctx(), nil)
	require.NoError(err, "failed to fetch latest L1 header")
	require.NotNil(latestL1Header, "latest L1 header must be available")
	require.NotNil(latestL1Header.Number, "latest L1 header must include a block number")

	plannedRollupStartBlockNumber := bigs.Uint64Strict(latestL1Header.Number) + preGenesisRollupStartBlockDelay
	plannedGenesisTime := latestL1Header.Time + preGenesisRollupStartBlockDelay*l1Net.blockTime
	claimedTimestamp := plannedGenesisTime + blockTime

	t.Logger().Info("preparing pre-genesis super game",
		"current_l1_head", bigs.Uint64Strict(latestL1Header.Number),
		"planned_rollup_start_block", plannedRollupStartBlockNumber,
		"planned_genesis_time", plannedGenesisTime,
		"claimed_timestamp", claimedTimestamp,
	)

	startingProposal := Proposal{
		Root:             preGenesisStartingAnchorRoot,
		L2SequenceNumber: new(big.Int).SetUint64(plannedGenesisTime),
	}
	sharedDGF := migrateSuperRootsWithProposal(
		t,
		keys,
		migration,
		l1Net.ChainID(),
		l1EL,
		startingProposal,
		l2Nets[0].ChainID(),
	)
	gameType := superCannonKonaGameType

	claimedChains := make([]eth.ChainIDAndOutput, 0, len(l2Nets))
	for i, l2Net := range l2Nets {
		claimedChains = append(claimedChains, eth.ChainIDAndOutput{
			ChainID: l2Net.ChainID(),
			Output:  gameCfg.ClaimedOutputs[i],
		})
	}
	extraData := eth.NewSuperV1(claimedTimestamp, claimedChains...).Marshal()
	rootClaim := crypto.Keccak256Hash(extraData)

	gameCreatorKey, err := keys.Secret(devkeys.UserKey(funderMnemonicIndex))
	require.NoError(err, "failed to derive pre-genesis game creator key")

	txOpts := txplan.Combine(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(gameCreatorKey),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithEstimator(client, true),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(client, 5, retry.Exponential()),
	)

	dgf := bindings.NewBindings[bindings.DisputeGameFactory](
		bindings.WithClient(ethClient),
		bindings.WithTo(sharedDGF),
		bindings.WithTest(t),
	)
	initBond, err := contractio.Read(dgf.InitBonds(uint32(gameType)), t.Ctx())
	require.NoError(err, "failed to read dispute game init bond")
	receipt, err := contractio.Write(
		dgf.Create(uint32(gameType), rootClaim, extraData),
		t.Ctx(),
		txOpts,
		txplan.WithValue(eth.WeiBig(initBond)),
		txplan.WithGasRatio(2),
	)
	require.NoError(err, "failed to create pre-genesis dispute game")
	require.EqualValues(types.ReceiptStatusSuccessful, receipt.Status, "pre-genesis dispute game creation failed")
	require.NotNil(receipt.BlockNumber, "pre-genesis dispute game receipt must include a block number")
	require.Lessf(bigs.Uint64Strict(receipt.BlockNumber), plannedRollupStartBlockNumber,
		"pre-genesis dispute game must be created before the chosen rollup start block")

	startBlockHeader := waitForL1Header(t, client, plannedRollupStartBlockNumber)
	require.EqualValues(plannedRollupStartBlockNumber, bigs.Uint64Strict(startBlockHeader.Number), "unexpected planned rollup start block")
	require.Greaterf(bigs.Uint64Strict(startBlockHeader.Number), bigs.Uint64Strict(receipt.BlockNumber),
		"rollup start block must be strictly after the pre-genesis dispute game creation block")
	require.EqualValues(plannedGenesisTime, startBlockHeader.Time, "unexpected planned rollup genesis time")

	for _, chainState := range wb.output.Chains {
		chainState.StartBlock = state.BlockRefJsonFromHeader(startBlockHeader)
	}
	wb.buildL2Genesis()
	wb.buildFullConfigSet()
	for _, l2Net := range l2Nets {
		l2Net.genesis = wb.outL2Genesis[l2Net.ChainID()]
		l2Net.rollupCfg = wb.outL2RollupCfg[l2Net.ChainID()]
	}
}

func waitForL1Header(t devtest.T, client *ethclient.Client, blockNum uint64) *types.Header {
	t.Helper()

	var header *types.Header
	t.Require().Eventuallyf(func() bool {
		var err error
		header, err = client.HeaderByNumber(t.Ctx(), new(big.Int).SetUint64(blockNum))
		if err != nil {
			return false
		}
		return header != nil
	}, contextTimeout(t), 250*time.Millisecond, "waiting for L1 block %d", blockNum)
	return header
}

func contextTimeout(t devtest.T) time.Duration {
	if deadline, ok := t.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			return remaining
		}
	}
	return 2 * time.Minute
}
