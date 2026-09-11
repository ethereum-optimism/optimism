package superfaultproofs

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/atomic"
)

type sponsoredAccount struct {
	account, paymaster                                     common.Address
	accountArtifact, entryPointArtifact, paymasterArtifact *foundry.Artifact
	owner                                                  *dsl.EOA
}

func prepareSponsoredAccount(t devtest.T, owner *dsl.EOA, el *dsl.L2ELNode, router common.Address, read func(string, string) *foundry.Artifact, deploy func(*dsl.EOA, *foundry.Artifact) common.Address) *sponsoredAccount {
	factory := read("AtomicAccountFactory", "AtomicAccountFactory")
	paymaster := read("AtomicPaymaster", "AtomicPaymaster")
	config := &sponsoredAccount{owner: owner, accountArtifact: read("SimpleAccount", "SimpleAccount"), entryPointArtifact: read("IEntryPoint", "IEntryPoint"), paymasterArtifact: paymaster}
	factoryAddr := deploy(owner, factory)
	data, err := factory.ABI.Pack("getAddress", owner.Address(), new(big.Int))
	t.Require().NoError(err)
	var predicted hexutil.Bytes
	t.Require().NoError(el.EthClient().RPC().CallContext(t.Ctx(), &predicted, "eth_call", map[string]any{"to": factoryAddr, "data": hexutil.Bytes(data)}, "latest"))
	config.account = common.BytesToAddress(predicted)
	data, err = factory.ABI.Pack("createAccount", owner.Address(), new(big.Int))
	t.Require().NoError(err)
	owner.Transact(owner.Plan(), txplan.WithTo(&factoryAddr), txplan.WithData(data))
	args, err := paymaster.ABI.Constructor.Inputs.Pack(owner.Address(), router, big.NewInt(5e16))
	t.Require().NoError(err)
	deployment := *paymaster
	deployment.Bytecode.Object = append(append([]byte(nil), paymaster.Bytecode.Object...), args...)
	config.paymaster = deploy(owner, &deployment)
	data, err = paymaster.ABI.Pack("setAccountAllowed", config.account, true)
	t.Require().NoError(err)
	owner.Transact(owner.Plan(), txplan.WithTo(&config.paymaster), txplan.WithData(data))
	data, err = paymaster.ABI.Pack("deposit")
	t.Require().NoError(err)
	owner.Transact(owner.Plan(), txplan.WithTo(&config.paymaster), txplan.WithData(data), txplan.WithValue(eth.OneEther.Div(10)))
	return config
}

func (s *sponsoredAccount) executor(backend *atomic.RPCExecutor, bundler *dsl.EOA) *atomic.UserOperationExecutor {
	return &atomic.UserOperationExecutor{Backend: backend, EntryPointABI: s.entryPointArtifact.ABI, AccountABI: s.accountArtifact.ABI, Account: s.account, Bundler: bundler.Address(), Paymaster: s.paymaster, ChainID: bundler.ChainID(), Nonce: new(big.Int), VerificationGas: 500_000, PaymasterVerificationGas: 200_000, PreVerificationGas: 300_000, OuterGas: 12_000_000, MaxFeePerGas: big.NewInt(1e9), MaxPriorityFeePerGas: big.NewInt(1), Sign: func(_ context.Context, hash common.Hash) ([]byte, error) {
		signature, err := crypto.Sign(accounts.TextHash(hash[:]), s.owner.Key().Priv())
		if err == nil {
			signature[64] += 27
		}
		return signature, err
	}}
}

func (s *sponsoredAccount) check(t devtest.T, el *dsl.L2ELNode, tx *txplan.PlannedTx, block uint64, succeeded bool) {
	receipt, err := el.EthClient().TransactionReceipt(t.Ctx(), tx.Signed.Value().Hash())
	t.Require().NoError(err)
	t.Require().Equal(uint64(types.ReceiptStatusSuccessful), receipt.Status, "EntryPoint catches operation reverts")
	event := s.entryPointArtifact.ABI.Events["UserOperationEvent"]
	count := 0
	cost := new(big.Int)
	for _, log := range receipt.Logs {
		if log.Address == predeploys.EntryPoint_v070Addr && len(log.Topics) == 4 && log.Topics[0] == event.ID {
			t.Require().Equal(s.account, common.BytesToAddress(log.Topics[2].Bytes()))
			t.Require().Equal(s.paymaster, common.BytesToAddress(log.Topics[3].Bytes()))
			values, err := event.Inputs.NonIndexed().Unpack(log.Data)
			t.Require().NoError(err)
			t.Require().Equal(succeeded, values[1].(bool), "application outcome is the UserOperation outcome")
			cost = values[2].(*big.Int)
			t.Require().Positive(cost.Sign(), "paymaster must pay for success and failure")
			count++
		}
	}
	t.Require().Equal(1, count)
	data, err := s.paymasterArtifact.ABI.Pack("getDeposit")
	t.Require().NoError(err)
	var remaining hexutil.Bytes
	t.Require().NoError(el.EthClient().RPC().CallContext(t.Ctx(), &remaining, "eth_call", map[string]any{"to": s.paymaster, "data": hexutil.Bytes(data)}, hexutil.EncodeUint64(block)))
	t.Require().Equal(new(big.Int).Sub(big.NewInt(1e17), cost), new(big.Int).SetBytes(remaining), "sponsor deposit settles the actual UserOperation charge")
	var balance hexutil.Big
	t.Require().NoError(el.EthClient().RPC().CallContext(t.Ctx(), &balance, "eth_getBalance", s.account, hexutil.EncodeUint64(block)))
	t.Require().Zero((*big.Int)(&balance).Sign(), "smart account needs no native ETH")
}
