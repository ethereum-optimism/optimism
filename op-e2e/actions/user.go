package actions

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
)

type L1Bindings struct {
	// contract bindings
	OptimismPortal *bindings.OptimismPortal

	L2OutputOracle *bindings.L2OutputOracle
}

func NewL1Bindings(t Testing, l1Cl *ethclient.Client) *L1Bindings {
	optimismPortal, err := bindings.NewOptimismPortal(config.L1Deployments.OptimismPortalProxy, l1Cl)
	require.NoError(t, err)

	l2OutputOracle, err := bindings.NewL2OutputOracle(config.L1Deployments.L2OutputOracleProxy, l1Cl)
	require.NoError(t, err)

	return &L1Bindings{
		OptimismPortal: optimismPortal,
		L2OutputOracle: l2OutputOracle,
	}
}

type L2Bindings struct {
	L2ToL1MessagePasser *bindings.L2ToL1MessagePasser

	ProofClient withdrawals.ProofClient
}

func NewL2Bindings(t Testing, l2Cl *ethclient.Client, proofCl withdrawals.ProofClient) *L2Bindings {
	l2ToL1MessagePasser, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Cl)
	require.NoError(t, err)

	return &L2Bindings{
		L2ToL1MessagePasser: l2ToL1MessagePasser,
		ProofClient:         proofCl,
	}
}

// BasicUserEnv provides access to the eth RPC, signer, and contract bindings for a single ethereum layer.
// This environment can be shared between different BasicUser instances.
type BasicUserEnv[B any] struct {
	EthCl  *ethclient.Client
	Signer types.Signer

	AddressCorpora []common.Address

	Bindings B
}

// BasicUser is an actor on a single ethereum layer, with one account key.
// The user maintains a set of standard txOpts to build its transactions with,
// along with configurable txToAddr and txCallData.
// The user has an RNG source with actions to randomize its transaction building.
type BasicUser[B any] struct {
	log log.Logger
	rng *rand.Rand
	env *BasicUserEnv[B]

	account *ecdsa.PrivateKey
	address common.Address

	txOpts bind.TransactOpts

	txToAddr   *common.Address
	txCallData []byte

	// lastTxHash persists the last transaction,
	// so we can chain together tx sending and tx checking easily.
	// Sending and checking are detached, since txs may not be instantly confirmed.
	lastTxHash common.Hash
}

func NewBasicUser[B any](log log.Logger, priv *ecdsa.PrivateKey, rng *rand.Rand) *BasicUser[B] {
	return &BasicUser[B]{
		log:     log,
		rng:     rng,
		account: priv,
		address: crypto.PubkeyToAddress(priv.PublicKey),
	}
}

// SetUserEnv changes the user environment.
// This way a user can be initialized before being embedded in a genesis allocation,
// and change between different endpoints that may be initialized after the user.
func (s *BasicUser[B]) SetUserEnv(env *BasicUserEnv[B]) {
	s.env = env
}

func (s *BasicUser[B]) signerFn(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
	if address != s.address {
		return nil, bind.ErrNotAuthorized
	}
	signature, err := crypto.Sign(s.env.Signer.Hash(tx).Bytes(), s.account)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s.env.Signer, signature)
}

// ActResetTxOpts prepares the tx options to default values, based on the current pending block header.
func (s *BasicUser[B]) ActResetTxOpts(t Testing) {
	latestHeader, err := s.env.EthCl.HeaderByNumber(t.Ctx(), nil)
	require.NoError(t, err, "need l2 latest header for accurate basefee info")

	gasTipCap := big.NewInt(2 * params.GWei)
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(latestHeader.BaseFee, big.NewInt(2)))

	s.txOpts = bind.TransactOpts{
		From:      s.address,
		Nonce:     nil, // pick nonce based on pending state
		Signer:    s.signerFn,
		Value:     big.NewInt(0),
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		GasLimit:  0,    // a.k.a. estimate
		NoSend:    true, // actions should be explicit about sending
	}
}

func (s *BasicUser[B]) ActRandomTxToAddr(t Testing) {
	i := s.rng.Intn(len(s.env.AddressCorpora))
	var to *common.Address
	if i > 0 { // 0 == nil
		to = &s.env.AddressCorpora[i]
	}
	s.txToAddr = to
}

func (s *BasicUser[B]) ActSetTxCalldata(calldata []byte) Action {
	return func(t Testing) {
		require.NotNil(t, calldata)
		s.txCallData = calldata
	}
}

func (s *BasicUser[B]) ActSetTxToAddr(to *common.Address) Action {
	return func(t Testing) {
		s.txToAddr = to
	}
}

func (s *BasicUser[B]) ActRandomTxValue(t Testing) {
	// compute a random portion of balance
	precision := int64(1000)
	bal, err := s.env.EthCl.BalanceAt(t.Ctx(), s.address, nil)
	require.NoError(t, err)
	part := big.NewInt(s.rng.Int63n(precision))
	new(big.Int).Div(new(big.Int).Mul(bal, part), big.NewInt(precision))
	s.txOpts.Value = big.NewInt(s.rng.Int63())
}

func (s *BasicUser[B]) ActSetTxValue(value *big.Int) Action {
	return func(t Testing) {
		s.txOpts.Value = value
	}
}

func (s *BasicUser[B]) ActRandomTxData(t Testing) {
	dataLen := s.rng.Intn(128_000)
	out := make([]byte, dataLen)
	_, err := s.rng.Read(out[:])
	require.NoError(t, err)
	s.txCallData = out
}

func (s *BasicUser[B]) PendingNonce(t Testing) uint64 {
	if s.txOpts.Nonce != nil {
		return s.txOpts.Nonce.Uint64()
	}
	// fetch from pending state
	nonce, err := s.env.EthCl.PendingNonceAt(t.Ctx(), s.address)
	require.NoError(t, err, "failed to get L1 nonce for account %s", s.address)
	return nonce
}

func (s *BasicUser[B]) TxValue() *big.Int {
	if s.txOpts.Value != nil {
		return s.txOpts.Value
	}
	return big.NewInt(0)
}

func (s *BasicUser[B]) LastTxReceipt(t Testing) *types.Receipt {
	require.NotEqual(t, s.lastTxHash, common.Hash{}, "must send tx before getting last receipt")
	receipt, err := s.env.EthCl.TransactionReceipt(t.Ctx(), s.lastTxHash)
	require.NoError(t, err)
	return receipt
}

// ActMakeTx makes a tx with the predetermined contents (see randomization and other actions)
// and sends it to the tx pool
func (s *BasicUser[B]) ActMakeTx(t Testing) {
	gas, err := s.env.EthCl.EstimateGas(t.Ctx(), ethereum.CallMsg{
		From:      s.address,
		To:        s.txToAddr,
		GasFeeCap: s.txOpts.GasFeeCap,
		GasTipCap: s.txOpts.GasTipCap,
		Value:     s.TxValue(),
		Data:      s.txCallData,
	})
	require.NoError(t, err, "gas estimation should pass")
	tx := types.MustSignNewTx(s.account, s.env.Signer, &types.DynamicFeeTx{
		To:        s.txToAddr,
		GasFeeCap: s.txOpts.GasFeeCap,
		GasTipCap: s.txOpts.GasTipCap,
		Value:     s.TxValue(),
		ChainID:   s.env.Signer.ChainID(),
		Nonce:     s.PendingNonce(t),
		Gas:       gas,
		Data:      s.txCallData,
	})
	err = s.env.EthCl.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send tx")
	s.lastTxHash = tx.Hash()
	// reset the calldata
	s.txCallData = []byte{}
}

func (s *BasicUser[B]) ActCheckReceiptStatusOfLastTx(success bool) func(t Testing) {
	return func(t Testing) {
		s.CheckReceipt(t, success, s.lastTxHash)
	}
}

func (s *BasicUser[B]) CheckReceipt(t Testing, success bool, txHash common.Hash) *types.Receipt {
	receipt, err := s.env.EthCl.TransactionReceipt(t.Ctx(), txHash)
	if receipt != nil && err == nil {
		expected := types.ReceiptStatusFailed
		if success {
			expected = types.ReceiptStatusSuccessful
		}
		require.Equal(t, expected, receipt.Status, "expected receipt status to match")
		return receipt
	} else if err != nil && !errors.Is(err, ethereum.NotFound) {
		t.Fatalf("receipt for tx %s was not found", txHash)
	} else {
		t.Fatalf("receipt error: %v", err)
	}
	return nil
}

type L1User struct {
	BasicUser[*L1Bindings]
}

type L2User struct {
	BasicUser[*L2Bindings]
}

// CrossLayerUser represents the same user account on L1 and L2,
// and provides actions to make cross-layer transactions.
type CrossLayerUser struct {
	L1 L1User
	L2 L2User

	// track the last deposit, to easily chain together deposit actions
	lastL1DepositTxHash common.Hash

	lastL2WithdrawalTxHash common.Hash
}

func NewCrossLayerUser(log log.Logger, priv *ecdsa.PrivateKey, rng *rand.Rand) *CrossLayerUser {
	addr := crypto.PubkeyToAddress(priv.PublicKey)
	return &CrossLayerUser{
		L1: L1User{
			BasicUser: BasicUser[*L1Bindings]{
				log:     log,
				rng:     rng,
				account: priv,
				address: addr,
			},
		},
		L2: L2User{
			BasicUser: BasicUser[*L2Bindings]{
				log:     log,
				rng:     rng,
				account: priv,
				address: addr,
			},
		},
	}
}

func (s *CrossLayerUser) ActDeposit(t Testing) {
	isCreation := false
	toAddr := common.Address{}
	if s.L2.txToAddr == nil {
		isCreation = true
	} else {
		toAddr = *s.L2.txToAddr
	}
	depositTransferValue := s.L2.TxValue()
	depositGas := s.L2.txOpts.GasLimit
	if s.L2.txOpts.GasLimit == 0 {
		// estimate gas used by deposit
		gas, err := s.L2.env.EthCl.EstimateGas(t.Ctx(), ethereum.CallMsg{
			From:       s.L2.address,
			To:         &toAddr,
			Value:      depositTransferValue, // TODO: estimate gas does not support minting yet
			Data:       s.L2.txCallData,
			AccessList: nil,
		})
		require.NoError(t, err)
		depositGas = gas
	}

	// Finally send TX
	s.L1.txOpts.GasLimit = 0
	tx, err := s.L1.env.Bindings.OptimismPortal.DepositTransaction(&s.L1.txOpts, toAddr, depositTransferValue, depositGas, isCreation, s.L2.txCallData)
	require.Nil(t, err, "with deposit tx")

	// Add 10% padding for the L1 gas limit because the estimation process can be affected by the 1559 style cost scale
	// for buying L2 gas in the portal contracts.
	s.L1.txOpts.GasLimit = tx.Gas() + (tx.Gas() / 10)

	tx, err = s.L1.env.Bindings.OptimismPortal.DepositTransaction(&s.L1.txOpts, toAddr, depositTransferValue, depositGas, isCreation, s.L2.txCallData)
	require.NoError(t, err, "failed to create deposit tx")

	s.L1.txOpts.GasLimit = 0

	fmt.Printf("Gas limit: %v\n", tx.Gas())
	// Send the actual tx (since tx opts don't send by default)
	err = s.L1.env.EthCl.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send tx")
	s.lastL1DepositTxHash = tx.Hash()
}

func (s *CrossLayerUser) ActCheckDepositStatus(l1Success, l2Success bool) Action {
	return func(t Testing) {
		s.CheckDepositTx(t, s.lastL1DepositTxHash, 0, l1Success, l2Success)
	}
}

func (s *CrossLayerUser) CheckDepositTx(t Testing, l1TxHash common.Hash, index int, l1Success, l2Success bool) {
	depositReceipt := s.L1.CheckReceipt(t, l1Success, l1TxHash)
	if depositReceipt == nil {
		require.False(t, l1Success)
		require.False(t, l2Success)
	} else {
		require.Less(t, index, len(depositReceipt.Logs), "must have enough logs in receipt")
		reconstructedDep, err := derive.UnmarshalDepositLogEvent(depositReceipt.Logs[index])
		require.NoError(t, err, "Could not reconstruct L2 Deposit")
		l2Tx := types.NewTx(reconstructedDep)
		s.L2.CheckReceipt(t, l2Success, l2Tx.Hash())
	}
}

func (s *CrossLayerUser) ActStartWithdrawal(t Testing) {
	targetAddr := common.Address{}
	if s.L1.txToAddr != nil {
		targetAddr = *s.L2.txToAddr
	}
	tx, err := s.L2.env.Bindings.L2ToL1MessagePasser.InitiateWithdrawal(&s.L2.txOpts, targetAddr, new(big.Int).SetUint64(s.L1.txOpts.GasLimit), s.L1.txCallData)
	require.NoError(t, err, "create initiate withdraw tx")
	err = s.L2.env.EthCl.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send tx")
	s.lastL2WithdrawalTxHash = tx.Hash()
}

// ActCheckStartWithdrawal checks that a previous witdrawal tx was either successful or failed.
func (s *CrossLayerUser) ActCheckStartWithdrawal(success bool) Action {
	return func(t Testing) {
		s.L2.CheckReceipt(t, success, s.lastL2WithdrawalTxHash)
	}
}

func (s *CrossLayerUser) Address() common.Address {
	return s.L1.address
}

// ActCompleteWithdrawal creates a L1 proveWithdrawal tx for latest withdrawal.
// The tx hash is remembered as the last L1 tx, to check as L1 actor.
func (s *CrossLayerUser) ActProveWithdrawal(t Testing) {
	s.L1.lastTxHash = s.ProveWithdrawal(t, s.lastL2WithdrawalTxHash)
}

// ProveWithdrawal creates a L1 proveWithdrawal tx for the given L2 withdrawal tx, returning the tx hash.
func (s *CrossLayerUser) ProveWithdrawal(t Testing, l2TxHash common.Hash) common.Hash {
	// Figure out when our withdrawal was included
	receipt := s.L2.CheckReceipt(t, true, l2TxHash)
	l2WithdrawalBlock, err := s.L2.env.EthCl.BlockByNumber(t.Ctx(), receipt.BlockNumber)
	require.NoError(t, err)

	// Figure out what the Output oracle on L1 has seen so far
	l2OutputBlockNr, err := s.L1.env.Bindings.L2OutputOracle.LatestBlockNumber(&bind.CallOpts{})
	require.NoError(t, err)
	l2OutputBlock, err := s.L2.env.EthCl.BlockByNumber(t.Ctx(), l2OutputBlockNr)
	require.NoError(t, err)
	l2OutputIndex, err := s.L1.env.Bindings.L2OutputOracle.GetL2OutputIndexAfter(&bind.CallOpts{}, l2OutputBlockNr)
	require.NoError(t, err)

	// Check if the L2 output is even old enough to include the withdrawal
	if l2OutputBlock.NumberU64() < l2WithdrawalBlock.NumberU64() {
		t.InvalidAction("the latest L2 output is %d and is not past L2 block %d that includes the withdrawal yet, no withdrawal can be proved yet", l2OutputBlock.NumberU64(), l2WithdrawalBlock.NumberU64())
		return common.Hash{}
	}

	// We generate a proof for the latest L2 output, which shouldn't require archive-node data if it's recent enough.
	header, err := s.L2.env.EthCl.HeaderByNumber(t.Ctx(), l2OutputBlockNr)
	require.NoError(t, err)
	params, err := withdrawals.ProveWithdrawalParameters(t.Ctx(), s.L2.env.Bindings.ProofClient, s.L2.env.EthCl, s.lastL2WithdrawalTxHash, header, &s.L1.env.Bindings.L2OutputOracle.L2OutputOracleCaller)
	require.NoError(t, err)

	// Create the prove tx
	tx, err := s.L1.env.Bindings.OptimismPortal.ProveWithdrawalTransaction(
		&s.L1.txOpts,
		bindings.TypesWithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		},
		l2OutputIndex,
		params.OutputRootProof,
		params.WithdrawalProof,
	)
	require.NoError(t, err)

	// Send the actual tx (since tx opts don't send by default)
	err = s.L1.env.EthCl.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send prove tx")
	return tx.Hash()
}

// ActCompleteWithdrawal creates a L1 withdrawal finalization tx for latest withdrawal.
// The tx hash is remembered as the last L1 tx, to check as L1 actor.
// The withdrawal functions like CompleteWithdrawal
func (s *CrossLayerUser) ActCompleteWithdrawal(t Testing) {
	s.L1.lastTxHash = s.CompleteWithdrawal(t, s.lastL2WithdrawalTxHash)
}

// CompleteWithdrawal creates a L1 withdrawal finalization tx for the given L2 withdrawal tx, returning the tx hash.
// It's an invalid action to attempt to complete a withdrawal that has not passed the L1 finalization period yet
func (s *CrossLayerUser) CompleteWithdrawal(t Testing, l2TxHash common.Hash) common.Hash {
	finalizationPeriod, err := s.L1.env.Bindings.L2OutputOracle.FINALIZATIONPERIODSECONDS(&bind.CallOpts{})
	require.NoError(t, err)

	// Figure out when our withdrawal was included
	receipt := s.L2.CheckReceipt(t, true, l2TxHash)
	l2WithdrawalBlock, err := s.L2.env.EthCl.BlockByNumber(t.Ctx(), receipt.BlockNumber)
	require.NoError(t, err)

	// Figure out what the Output oracle on L1 has seen so far
	l2OutputBlockNr, err := s.L1.env.Bindings.L2OutputOracle.LatestBlockNumber(&bind.CallOpts{})
	require.NoError(t, err)
	l2OutputBlock, err := s.L2.env.EthCl.BlockByNumber(t.Ctx(), l2OutputBlockNr)
	require.NoError(t, err)

	// Check if the L2 output is even old enough to include the withdrawal
	if l2OutputBlock.NumberU64() < l2WithdrawalBlock.NumberU64() {
		t.InvalidAction("the latest L2 output is %d and is not past L2 block %d that includes the withdrawal yet, no withdrawal can be completed yet", l2OutputBlock.NumberU64(), l2WithdrawalBlock.NumberU64())
		return common.Hash{}
	}

	l1Head, err := s.L1.env.EthCl.HeaderByNumber(t.Ctx(), nil)
	require.NoError(t, err)

	// Check if the withdrawal may be completed yet
	if l2OutputBlock.Time()+finalizationPeriod.Uint64() >= l1Head.Time {
		t.InvalidAction("withdrawal tx %s was included in L2 block %d (time %d) but L1 only knows of L2 proposal %d (time %d) at head %d (time %d) which has not reached output confirmation yet (period is %d)",
			l2TxHash, l2WithdrawalBlock.NumberU64(), l2WithdrawalBlock.Time(), l2OutputBlock.NumberU64(), l2OutputBlock.Time(), l1Head.Number.Uint64(), l1Head.Time, finalizationPeriod.Uint64())
		return common.Hash{}
	}

	// We generate a proof for the latest L2 output, which shouldn't require archive-node data if it's recent enough.
	// Note that for the `FinalizeWithdrawalTransaction` function, this proof isn't needed. We simply use some of the
	// params for the `WithdrawalTransaction` type generated in the bindings.
	header, err := s.L2.env.EthCl.HeaderByNumber(t.Ctx(), l2OutputBlockNr)
	require.NoError(t, err)
	params, err := withdrawals.ProveWithdrawalParameters(t.Ctx(), s.L2.env.Bindings.ProofClient, s.L2.env.EthCl, s.lastL2WithdrawalTxHash, header, &s.L1.env.Bindings.L2OutputOracle.L2OutputOracleCaller)
	require.NoError(t, err)

	// Create the withdrawal tx
	tx, err := s.L1.env.Bindings.OptimismPortal.FinalizeWithdrawalTransaction(
		&s.L1.txOpts,
		bindings.TypesWithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		},
	)
	require.NoError(t, err)

	// Send the actual tx (since tx opts don't send by default)
	err = s.L1.env.EthCl.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send finalize tx")
	return tx.Hash()
}
