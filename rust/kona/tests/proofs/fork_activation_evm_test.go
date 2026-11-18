package proofs

import (
	"bytes"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/invoker"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
)

// forkActivationBlock is the L2 block that activates the fork under test.
const forkActivationBlock = 3

// forkActivationProbeGas is the gas limit of every probe transaction.
const forkActivationProbeGas = uint64(2_000_000)

// statusProbe is a transaction whose receipt status a fork flips.
type statusProbe struct {
	name string
	// to is nil for a contract creation that runs data as its initcode.
	to   *common.Address
	data []byte
	// succeedsFromFork is the status from the fork on. Before the fork the status is the
	// opposite. A failing probe halts, so it spends its whole gas limit.
	succeedsFromFork bool
}

// forkActivationBlocks are the blocks built around a fork's activation.
type forkActivationBlocks struct {
	before, activation, after eth.L2BlockRef
}

// forkActivationCase lists EVM rules that a fork switches on.
type forkActivationCase struct {
	fork         forks.Name
	statusProbes []statusProbe
	// outputs are precompile calls made through the Invoker. Each returns no output before the
	// fork, when its address has no code, and the fixture's Output from the fork on. The Invoker
	// uses PUSH0 and MCOPY, so only a fork after Ecotone can have outputs.
	outputs []PrecompileTestFixture
	// probeActivationBlock also sends the probes into the activation block. This catches an engine
	// that takes its rules from the parent block's timestamp. Only a fork whose activation block
	// may carry user transactions can set it: op-node's sequencer leaves them out of each activation
	// block that carries upgrade transactions (Ecotone, Fjord, Isthmus, and Jovian on), and
	// derivation drops them from Jovian on.
	probeActivationBlock bool
	// l2Alloc replaces accounts of the default L2 genesis.
	l2Alloc types.GenesisAlloc
	// checkBlocks checks header fields and state that the fork changes.
	checkBlocks func(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, blocks forkActivationBlocks)
}

// create2DeployerCodeHash is the code hash of the create2 deployer that Canyon force-deploys
// (CREATE_2_DEPLOYER_CODEHASH in alloy-op-evm/src/block/canyon.rs).
var create2DeployerCodeHash = common.HexToHash("0xb0550b5b431e30d38000efb7107aaa0ade03d48a7198a140edda9d27134468b2")

// blsMalformedInputs are the EIP-2537 failure vectors: well-formed calls that each BLS12-381
// precompile must reject from Isthmus on.
var blsMalformedInputs = []statusProbe{
	// A G1 point not on the curve.
	{name: "blsG1AddOffCurve", to: ptr(common.BytesToAddress([]byte{0x0b})), data: common.FromHex("0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb00000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a2100000000000000000000000000000000112b98340eee2777cc3c14163dea3ec97977ac3dc5c70da32e6e87578f44912e902ccef9efe28d4a78b8999dfbca942600000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a21")},
	// A non-canonical field element.
	{name: "blsG2AddNonCanonical", to: ptr(common.BytesToAddress([]byte{0x0d})), data: common.FromHex("000000000000000000000000000000001c4bb49d2a0ef12b7123acdd7110bd292b5bc659edc54dc21b81de057194c79b2a5803255959bbef8e7f56c8c12168630000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be00000000000000000000000000000000103121a2ceaae586d240843a398967325f8eb5a93e8fea99b62b9f88d8556c80dd726a4b30e84a36eeabaf3592937f2700000000000000000000000000000000086b990f3da2aeac0a36143b7d7c824428215140db1bb859338764cb58458f081d92664f9053b50b3fbd2e4723121b68000000000000000000000000000000000f9e7ba9a86a8f7624aa2b42dcc8772e1af4ae115685e60abc2c9b90242167acef3d0be4050bf935eed7c3b6fc7ba77e000000000000000000000000000000000d22c3652d0dc6f0fc9316e14268477c2049ef772e852108d269d9c38dba1d4802e8dae479818184c08f9a569d878451")},
	// A G1 point not on the curve.
	{name: "blsG1MsmOffCurve", to: ptr(common.BytesToAddress([]byte{0x0c})), data: common.FromHex("0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb00000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a21000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000112b98340eee2777cc3c14163dea3ec97977ac3dc5c70da32e6e87578f44912e902ccef9efe28d4a78b8999dfbca942600000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a210000000000000000000000000000000000000000000000000000000000000002")},
	// A non-canonical field element.
	{name: "blsG2MsmNonCanonical", to: ptr(common.BytesToAddress([]byte{0x0e})), data: common.FromHex("000000000000000000000000000000001c4bb49d2a0ef12b7123acdd7110bd292b5bc659edc54dc21b81de057194c79b2a5803255959bbef8e7f56c8c12168630000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000103121a2ceaae586d240843a398967325f8eb5a93e8fea99b62b9f88d8556c80dd726a4b30e84a36eeabaf3592937f2700000000000000000000000000000000086b990f3da2aeac0a36143b7d7c824428215140db1bb859338764cb58458f081d92664f9053b50b3fbd2e4723121b68000000000000000000000000000000000f9e7ba9a86a8f7624aa2b42dcc8772e1af4ae115685e60abc2c9b90242167acef3d0be4050bf935eed7c3b6fc7ba77e000000000000000000000000000000000d22c3652d0dc6f0fc9316e14268477c2049ef772e852108d269d9c38dba1d4802e8dae479818184c08f9a569d8784510000000000000000000000000000000000000000000000000000000000000002")},
	// A non-canonical field element.
	{name: "blsHashToG1NonCanonical", to: ptr(common.BytesToAddress([]byte{0x10})), data: common.FromHex("000000000000000000000000000000002f6d9c5465982c0421b61e74579709b3b5b91e57bdd4f6015742b4ff301abb7ef895b9cce00c33c7d48f8e5fa4ac09ae")},
	// A non-canonical field element.
	{name: "blsHashToG2NonCanonical", to: ptr(common.BytesToAddress([]byte{0x11})), data: common.FromHex("0000000000000000000000000000000021366f100476ce8d3be6cfc90d59fe13349e388ed12b6dd6dc31ccd267ff000e2c993a063ca66beced06f804d4b8e5af0000000000000000000000000000000002829ce3c021339ccb5caf3e187f6370e1e2a311dec9b75363117063ab2015603ff52c3d3b98f19c2f65575e99e8b78c")},
	// A G1 point outside the subgroup.
	{name: "blsPairingCheckOutsideSubgroup", to: ptr(common.BytesToAddress([]byte{0x0f})), data: common.FromHex("000000000000000000000000000000000123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00000000000000000000000000000000193fb7cedb32b2c3adc06ec11a96bc0d661869316f5e4a577a9f7c179593987beb4fb2ee424dbb2f5dd891e228b46c4a00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb80000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e100000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb80000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000d1b3cc2c7027888be51d9ef691d77bcb679afda66c73f17f9ee3837a55024f78c71363275a75d75d86bab79f74782aa0000000000000000000000000000000013fa4d4a0ad8b1ce186ed5061789213d993923066dddaf1040bc3ff59f825c78df74f2d75467e25e0f55f8a00fa030ed")},
}

// TestForkActivationEVMRules activates a fork after genesis, as live chains activated it, and
// checks the EVM rules that the fork switches on. It runs the same probes in a block before the
// activation block and in a block after it, and in the activation block too when that block may
// carry user transactions. For Canyon and Ecotone it also checks header fields and state that the
// fork changes. It then proves the three blocks with the fault-proof program. TestPrecompiles
// activates each precompile's fork at genesis, so it cannot catch an engine that picks its opcodes
// or precompiles from the genesis fork instead of from each block's timestamp.
func TestForkActivationEVMRules(gt *testing.T) {
	kzg := precompileFixture(gt, "kzgPointEvaluation")
	// Changing the claimed evaluation y, at bytes 64..96, makes the proof fail to verify.
	kzgWrongEvaluation := bytes.Clone(kzg.Input)
	kzgWrongEvaluation[95] ^= 1

	p256 := precompileFixture(gt, "p256Verify")
	// A signature that does not verify does not make P256VERIFY fail: it returns no output.
	// Changing the message hash, at bytes 0..32, makes the signature fail to verify.
	p256WrongHash := bytes.Clone(p256.Input)
	p256WrongHash[31] ^= 1
	p256Unverified := PrecompileTestFixture{Name: "p256VerifyWrongHash", Address: p256.Address, Input: p256WrongHash, Output: []byte{}}

	var blsOutputs []PrecompileTestFixture
	for _, fixture := range PrecompileTestFixtures {
		if fixture.ActivationFork == forks.Isthmus {
			blsOutputs = append(blsOutputs, fixture)
		}
	}
	require.Len(gt, blsOutputs, len(blsMalformedInputs), "one valid-output fixture per EIP-2537 precompile")

	matrix := helpers.NewMatrix[forkActivationCase]()
	for _, c := range []forkActivationCase{
		{
			fork: forks.Canyon,
			statusProbes: []statusProbe{
				{name: "PUSH0", data: []byte{byte(vm.PUSH0)}, succeedsFromFork: true},
			},
			probeActivationBlock: true,
			// The default L2 genesis preinstalls the create2 deployer, so Canyon's force-deployment
			// would change nothing. An empty account removes it.
			l2Alloc:     types.GenesisAlloc{predeploys.Create2DeployerAddr: {Balance: new(big.Int)}},
			checkBlocks: checkCanyonBlocks,
		},
		{
			fork: forks.Ecotone,
			statusProbes: []statusProbe{
				{name: "TSTORE", data: []byte{byte(vm.PUSH1), 1, byte(vm.PUSH1), 1, byte(vm.TSTORE)}, succeedsFromFork: true},
				{name: "kzgPointEvaluationWrongEvaluation", to: &kzg.Address, data: kzgWrongEvaluation},
			},
			checkBlocks: checkEcotoneBlocks,
		},
		{fork: forks.Fjord, outputs: []PrecompileTestFixture{p256, p256Unverified}},
		{fork: forks.Isthmus, statusProbes: blsMalformedInputs, outputs: blsOutputs},
	} {
		// The fork before the one under test is active at genesis. It only names the subtest:
		// runForkActivationTest schedules every fork itself.
		genesisFork := lookupHardforkHelper(forks.Prev(c.fork))
		require.NotNil(gt, genesisFork, "no hardfork helper for the fork before %s", c.fork)
		matrix.AddTestCase(string(c.fork), c, helpers.NewForkMatrix(genesisFork), runForkActivationTest, helpers.ExpectNoError())
	}
	matrix.Run(gt)
}

func runForkActivationTest(gt *testing.T, testCfg *helpers.TestCfg[forkActivationCase]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	c := testCfg.Custom
	if c.l2Alloc != nil {
		allocs := *actionsHelpers.DefaultAlloc
		allocs.L2Alloc = c.l2Alloc
		testCfg.Allocs = &allocs
	}
	batcherCfg := helpers.NewBatcherCfg(func(b *actionsHelpers.BatcherCfg) {
		// Calldata is the only batch transport before Ecotone.
		b.DataAvailabilityType = batcherFlags.CalldataType
	})
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), batcherCfg, func(dc *genesis.DeployConfig) {
		dc.ActivateForkAtOffset(c.fork, forkActivationBlock*dc.L2BlockTime)
	})
	// The L1 genesis has a zero parent beacon block root. An L1 block timed for the first L2 block
	// becomes the L1 origin of every block below, so checkEcotoneBlocks sees a real root.
	env.Miner.ActL1StartBlock(env.Sd.RollupCfg.BlockTime)(t)
	env.Miner.ActL1EndBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	l2 := env.Engine.EthClient()
	alice := env.Alice.Address()
	chainID := env.Sd.RollupCfg.L2ChainID
	signer := types.LatestSignerForChainID(chainID)
	invokerABI, err := invoker.InvokerMetaData.GetAbi()
	require.NoError(t, err)

	// Probes are signed here rather than with Alice's L2 user: it estimates gas, which fails for
	// a transaction that fails.
	nonce, err := l2.NonceAt(t.Ctx(), alice, nil)
	require.NoError(t, err)
	send := func(to *common.Address, data []byte) *types.Transaction {
		tip := big.NewInt(params.GWei)
		feeCap := new(big.Int).Add(new(big.Int).Mul(env.Engine.LatestHeader(t).BaseFee, big.NewInt(2)), tip)
		tx := types.MustSignNewTx(env.Dp.Secrets.Alice, signer, &types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: tip,
			GasFeeCap: feeCap,
			Gas:       forkActivationProbeGas,
			To:        to,
			Data:      data,
		})
		nonce++
		require.NoError(t, l2.SendTransaction(t.Ctx(), tx))
		return tx
	}
	buildBlock := func(txs []*types.Transaction) eth.L2BlockRef {
		env.Sequencer.ActL2StartBlock(t)
		for range txs {
			env.Engine.ActL2IncludeTx(alice)(t)
		}
		return env.Sequencer.ActL2EndBlock(t)
	}
	receipt := func(tx *types.Transaction, block eth.L2BlockRef) *optypes.Receipt {
		r, err := env.Engine.SourceClient().TransactionReceipt(t.Ctx(), tx.Hash())
		require.NoError(t, err)
		require.Equal(t, block.Hash, r.BlockHash, "transaction must be in block %d", block.Number)
		return r
	}

	var deployTxs []*types.Transaction
	if len(c.outputs) > 0 {
		deployTxs = append(deployTxs, send(nil, common.FromHex(invoker.InvokerMetaData.Bin)))
	}
	deployBlock := buildBlock(deployTxs)
	var invokerAddr common.Address
	if len(deployTxs) > 0 {
		r := receipt(deployTxs[0], deployBlock)
		require.Equal(t, types.ReceiptStatusSuccessful, r.Status, "Invoker deployment")
		invokerAddr = r.ContractAddress
	}

	// sendProbes sends every probe, the status probes first.
	sendProbes := func() []*types.Transaction {
		var txs []*types.Transaction
		for _, probe := range c.statusProbes {
			txs = append(txs, send(probe.to, probe.data))
		}
		for _, fixture := range c.outputs {
			calldata, err := invokerABI.Pack("invokePrecompile", fixture.Address, fixture.Input)
			require.NoError(t, err)
			txs = append(txs, send(&invokerAddr, calldata))
		}
		return txs
	}
	checkProbes := func(block eth.L2BlockRef, txs []*types.Transaction, forkActive bool) {
		for i, probe := range c.statusProbes {
			r := receipt(txs[i], block)
			if probe.succeedsFromFork == forkActive {
				require.Equal(t, types.ReceiptStatusSuccessful, r.Status, "%s with %s active: %v", probe.name, c.fork, forkActive)
			} else {
				require.Equal(t, types.ReceiptStatusFailed, r.Status, "%s with %s active: %v", probe.name, c.fork, forkActive)
				require.Equal(t, txs[i].Gas(), r.GasUsed, "%s must halt, spending its whole gas limit", probe.name)
			}
		}
		for i, fixture := range c.outputs {
			r := receipt(txs[len(c.statusProbes)+i], block)
			require.Equal(t, types.ReceiptStatusSuccessful, r.Status, "Invoker call to %s", fixture.Name)
			require.Len(t, r.Logs, 1)
			var out struct {
				Result             []byte
				DelegateCallResult []byte
			}
			require.NoError(t, invokerABI.UnpackIntoInterface(&out, "PrecompileInvoked", r.Logs[0].Data))
			want := []byte{}
			if forkActive {
				want = fixture.Output
			}
			require.Equal(t, want, out.Result, "%s with %s active: %v", fixture.Name, c.fork, forkActive)
			require.Equal(t, want, out.DelegateCallResult, "%s with %s active: %v", fixture.Name, c.fork, forkActive)
		}
	}

	beforeTxs := sendProbes()
	before := buildBlock(beforeTxs)
	var activationTxs []*types.Transaction
	if c.probeActivationBlock {
		activationTxs = sendProbes()
	}
	activation := buildBlock(activationTxs)
	require.Equal(t, uint64(forkActivationBlock), activation.Number)
	require.True(t, env.Sd.RollupCfg.IsActivationBlockForFork(activation.Time, c.fork), "block %d must activate %s", activation.Number, c.fork)
	afterTxs := sendProbes()
	after := buildBlock(afterTxs)

	checkProbes(before, beforeTxs, false)
	if c.probeActivationBlock {
		checkProbes(activation, activationTxs, true)
	}
	checkProbes(after, afterTxs, true)
	if c.checkBlocks != nil {
		c.checkBlocks(t, env, forkActivationBlocks{before: before, activation: activation, after: after})
	}

	require.Equal(t, after, env.BatchMineAndSync(t), "derivation must reproduce the engine-built blocks")
	for _, block := range []eth.L2BlockRef{before, activation, after} {
		env.RunFaultProofProgram(t, block.Number, testCfg.CheckResult, testCfg.InputParams...)
	}
}

// checkCanyonBlocks checks that blocks carry withdrawals from Canyon on, and that the activation
// block force-deploys the create2 deployer.
func checkCanyonBlocks(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, blocks forkActivationBlocks) {
	require.Nil(t, env.Engine.HeaderByHash(t, blocks.before.Hash).WithdrawalsHash, "blocks have no withdrawals before Canyon")
	for _, block := range []eth.L2BlockRef{blocks.activation, blocks.after} {
		withdrawalsHash := env.Engine.HeaderByHash(t, block.Hash).WithdrawalsHash
		require.NotNil(t, withdrawalsHash, "block %d must have withdrawals", block.Number)
		require.Equal(t, types.EmptyWithdrawalsHash, *withdrawalsHash, "block %d must have empty withdrawals", block.Number)
	}

	code := func(block eth.L2BlockRef) []byte {
		code, err := env.Engine.EthClient().CodeAt(t.Ctx(), predeploys.Create2DeployerAddr, new(big.Int).SetUint64(block.Number))
		require.NoError(t, err)
		return code
	}
	require.Empty(t, code(blocks.before), "the create2 deployer must not exist before Canyon")
	require.Equal(t, create2DeployerCodeHash, crypto.Keccak256Hash(code(blocks.activation)), "Canyon must force-deploy the create2 deployer")
}

// checkEcotoneBlocks checks that from Ecotone on each block carries the parent beacon block root of
// its L1 origin.
func checkEcotoneBlocks(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, blocks forkActivationBlocks) {
	require.Nil(t, env.Engine.HeaderByHash(t, blocks.before.Hash).ParentBeaconRoot, "blocks have no parent beacon block root before Ecotone")
	for _, block := range []eth.L2BlockRef{blocks.activation, blocks.after} {
		origin, err := env.Miner.EthClient().HeaderByHash(t.Ctx(), block.L1Origin.Hash)
		require.NoError(t, err)
		// op-node falls back to the zero hash when the L1 origin has no root, which would make the
		// check below pass for any engine that defaults to zero.
		require.NotNil(t, origin.ParentBeaconRoot, "the L1 origin of block %d must have a parent beacon block root", block.Number)
		require.NotEqual(t, common.Hash{}, *origin.ParentBeaconRoot, "the L1 origin of block %d must have a nonzero parent beacon block root", block.Number)
		root := env.Engine.HeaderByHash(t, block.Hash).ParentBeaconRoot
		require.NotNil(t, root, "block %d must have a parent beacon block root", block.Number)
		require.Equal(t, *origin.ParentBeaconRoot, *root, "block %d must carry the parent beacon block root of its L1 origin", block.Number)
	}
}

// precompileFixture returns the fixture of PrecompileTestFixtures called name.
func precompileFixture(t *testing.T, name string) PrecompileTestFixture {
	i := slices.IndexFunc(PrecompileTestFixtures, func(f PrecompileTestFixture) bool { return f.Name == name })
	require.NotEqual(t, -1, i, "no precompile fixture %s", name)
	return PrecompileTestFixtures[i]
}
