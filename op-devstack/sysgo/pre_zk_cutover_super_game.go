package sysgo

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
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

func seedPreZKCutoverSuperGame(
	t devtest.T,
	keys devkeys.Keys,
	primaryL2 eth.ChainID,
	l1EL L1ELNode,
	supernode *SuperNode,
	factoryAddress common.Address,
	minimumSequence uint64,
) {
	require := t.Require()
	extraData := waitForSafeSuperRootAfter(t, supernode, minimumSequence).Marshal()
	rootClaim := crypto.Keccak256Hash(extraData)

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err, "failed to connect to L1 RPC")
	defer rpcClient.Close()
	ethClient := ethclient.NewClient(rpcClient)
	rpcWrapper := client.NewBaseRPCClient(rpcClient)
	contractClient, err := sources.NewEthClient(rpcWrapper, t.Logger(), nil, sources.DefaultEthClientConfig(10))
	require.NoError(err, "failed to create L1 contract client")

	creatorKey, err := keys.Secret(devkeys.ProposerRole.Key(primaryL2.ToBig()))
	require.NoError(err, "failed to derive L1 proposer role key")
	txOpts := txplan.Combine(
		txplan.WithChainID(ethClient),
		txplan.WithPrivateKey(creatorKey),
		txplan.WithPendingNonce(ethClient),
		txplan.WithAgainstLatestBlockEthClient(ethClient),
		txplan.WithEstimator(ethClient, true),
		txplan.WithRetrySubmission(ethClient, 5, retry.Exponential()),
		txplan.WithRetryInclusion(txplan.FromGethReceipts(ethClient), 5, retry.Exponential()),
	)
	dgf := bindings.NewBindings[bindings.DisputeGameFactory](
		bindings.WithClient(contractClient),
		bindings.WithTo(factoryAddress),
		bindings.WithTest(t),
	)
	initBond, err := contractio.Read(dgf.InitBonds(superCannonKonaGameType), t.Ctx())
	require.NoError(err, "failed to read SuperCannonKona init bond")
	receipt, err := contractio.Write(
		dgf.Create(superCannonKonaGameType, rootClaim, extraData),
		t.Ctx(),
		txOpts,
		txplan.WithValue(eth.WeiBig(initBond)),
		txplan.WithGasRatio(2),
	)
	require.NoError(err, "failed to create pre-ZK-cutover SuperCannonKona game")
	require.EqualValues(types.ReceiptStatusSuccessful, receipt.Status, "pre-ZK-cutover game creation failed")
}

func waitForSafeSuperRootAfter(t devtest.T, supernode *SuperNode, minimumSequence uint64) *eth.SuperV1 {
	client, err := dial.DialSuperNodeClientWithTimeout(t.Ctx(), t.Logger(), supernode.UserRPC())
	t.Require().NoError(err, "failed to connect to supernode RPC")
	defer client.Close()

	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	var superRoot *eth.SuperV1
	err = wait.For(ctx, time.Second, func() (bool, error) {
		status, err := client.SuperRootAtTimestamp(ctx, uint64(time.Now().Unix()))
		if err != nil || status.CurrentSafeTimestamp <= minimumSequence {
			return false, nil
		}
		response, err := client.SuperRootAtTimestamp(ctx, status.CurrentSafeTimestamp)
		if err != nil || response.Data == nil {
			return false, nil
		}
		var ok bool
		superRoot, ok = response.Data.Super.(*eth.SuperV1)
		if !ok {
			return false, fmt.Errorf("unsupported super root type %T", response.Data.Super)
		}
		return true, nil
	})
	t.Require().NoError(err, "safe super root did not advance past sequence %d", minimumSequence)
	return superRoot
}
