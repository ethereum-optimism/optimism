package utils

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	bindingsnew "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

// ProveWithdrawalParameters calls ProveWithdrawalParametersForBlock with the most recent L2 output after the latest game.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack
func ProveWithdrawalParameters(t devtest.T, l2Chain *dsl.L2Network, l1Client apis.EthClient, l2Client apis.EthClient, l2WithdrawalReceipt *types.Receipt) (withdrawals.ProvenWithdrawalParameters, error) {
	latestGame, err := FindLatestGame(t, l2Chain, l1Client)
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to find latest game: %w", err)
	}

	l2BlockNumber := new(big.Int).SetBytes(latestGame.ExtraData[0:32])
	l2OutputIndex := latestGame.Index
	// Fetch the block header from the L2 node
	l2Header, err := l2Client.InfoByNumber(t.Ctx(), l2BlockNumber.Uint64())
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to get l2Block: %w", err)
	}
	return ProveWithdrawalParametersForBlock(t, l2Client, l2WithdrawalReceipt, l2Header, l2OutputIndex)
}

// ProveWithdrawalParametersForBlock queries L1 & L2 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack, using op-service ethclient
func ProveWithdrawalParametersForBlock(t devtest.T, l2Client apis.EthClient, l2WithdrawalReceipt *types.Receipt, l2Header eth.BlockInfo, l2OutputIndex *big.Int) (withdrawals.ProvenWithdrawalParameters, error) {
	// Transaction receipt
	// Parse the receipt
	ev, err := withdrawals.ParseMessagePassed(l2WithdrawalReceipt)
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, err
	}
	return ProveWithdrawalParametersForEvent(t, l2Client, ev, l2Header, l2OutputIndex)
}

// ProveWithdrawalParametersForEvent queries L1 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack, using op-service ethclient
func ProveWithdrawalParametersForEvent(t devtest.T, l2Client apis.EthClient, ev *bindings.L2ToL1MessagePasserMessagePassed, l2Header eth.BlockInfo, l2OutputIndex *big.Int) (withdrawals.ProvenWithdrawalParameters, error) {
	// Generate then verify the withdrawal proof
	withdrawalHash, err := withdrawals.WithdrawalHash(ev)
	if !bytes.Equal(withdrawalHash[:], ev.WithdrawalHash[:]) {
		return withdrawals.ProvenWithdrawalParameters{}, errors.New("computed withdrawal hash incorrectly")
	}
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, err
	}
	slot := withdrawals.StorageSlotOfWithdrawalHash(withdrawalHash)

	p, err := l2Client.GetProof(t.Ctx(), predeploys.L2ToL1MessagePasserAddr, []common.Hash{slot}, "0x"+fmt.Sprintf("%x", l2Header.NumberU64()))
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, err
	}
	if len(p.StorageProof) != 1 {
		return withdrawals.ProvenWithdrawalParameters{}, errors.New("invalid amount of storage proofs")
	}

	err = VerifyProof(l2Header.Root(), p)
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, err
	}

	// Encode it as expected by the contract
	trieNodes := make([][]byte, len(p.StorageProof[0].Proof))
	for i, s := range p.StorageProof[0].Proof {
		trieNodes[i] = s
	}

	return withdrawals.ProvenWithdrawalParameters{
		Nonce:         ev.Nonce,
		Sender:        ev.Sender,
		Target:        ev.Target,
		Value:         ev.Value,
		GasLimit:      ev.GasLimit,
		L2OutputIndex: l2OutputIndex,
		Data:          ev.Data,
		OutputRootProof: bindings.TypesOutputRootProof{
			Version:                  [32]byte{}, // Empty for version 1
			StateRoot:                eth.Bytes32(l2Header.Root()),
			MessagePasserStorageRoot: *l2Header.WithdrawalsRoot(),
			LatestBlockhash:          l2Header.Hash(),
		},
		WithdrawalProof: trieNodes,
	}, nil
}

// FindLatestGame finds the latest game in the DisputeGameFactory contract.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack, using op-service ethclient
func FindLatestGame(t devtest.T, l2Chain *dsl.L2Network, l1Client apis.EthClient) (bindingsnew.GameSearchResult, error) {
	rollupConfig := l2Chain.Escape().RollupConfig()
	disputeGameFactoryAddr := l2Chain.Escape().Deployment().DisputeGameFactoryProxyAddr()
	optimismPortalAddr := rollupConfig.DepositContractAddress
	portal := bindingsnew.NewBindings[bindingsnew.OptimismPortal2](bindingsnew.WithClient(l1Client), bindingsnew.WithTo(optimismPortalAddr), bindingsnew.WithTest(t))

	disputeGameFactory := bindingsnew.NewBindings[bindingsnew.DisputeGameFactory](bindingsnew.WithClient(l1Client), bindingsnew.WithTo(disputeGameFactoryAddr), bindingsnew.WithTest(t))
	respectedGameType := contract.Read(portal.RespectedGameType())

	gameCount := contract.Read(disputeGameFactory.GameCount())
	if gameCount.Cmp(common.Big0) == 0 {
		return bindingsnew.GameSearchResult{}, errors.New("no games")
	}

	searchStart := new(big.Int).Sub(gameCount, common.Big1)
	latestGames := contract.Read(disputeGameFactory.FindLatestGames(respectedGameType, searchStart, common.Big1))
	if len(latestGames) == 0 {
		return bindingsnew.GameSearchResult{}, errors.New("no latest games")
	}

	latestGame := latestGames[0]
	return latestGame, nil
}
