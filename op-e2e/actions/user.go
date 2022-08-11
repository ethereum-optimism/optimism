package actions

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type UserEnvironment struct {
	l1 *ethclient.Client
	l2 *ethclient.Client

	l1ChainID *big.Int
	l2ChainID *big.Int

	l1Signer types.Signer
	l2Signer types.Signer

	// contract bindings
	bindingPortal *bindings.OptimismPortal

	// TODO add bindings/actions for interacting with the other contracts

	addressCorpora []common.Address
}

type UserSpawner struct {
	log log.Logger
	rng *rand.Rand
}

// SpawnUser creates a new user with its own RNG
func (s *UserSpawner) SpawnUser() *User {
	rng := rand.New(rand.NewSource(s.rng.Int63()))
	priv, err := ecdsa.GenerateKey(crypto.S256(), rng)
	if err != nil {
		panic(fmt.Errorf("failed to generate priv key: %v", err))
	}
	return NewUser(s.log, priv, rng)
}

type User struct {
	log log.Logger
	rng *rand.Rand
	env *UserEnvironment

	account *ecdsa.PrivateKey
	address common.Address

	// selectedToAddr is the address used as recipient in txs: addressCorpora[selectedToAddr % uint64(len(s.addressCorpora)]
	selectedToAddr uint64
}

var _ ActorUser = (*User)(nil)

func NewUser(log log.Logger, priv *ecdsa.PrivateKey, rng *rand.Rand) *User {
	return &User{
		log:            log,
		rng:            rng,
		account:        priv,
		address:        crypto.PubkeyToAddress(priv.PublicKey),
		selectedToAddr: 0,
	}
}

func (s *User) SetEnv(env *UserEnvironment) {
	s.env = env
}

// add rollup deposit to L1 tx queue
func (s *User) actL1Deposit(t Testing) {
	nonce, err := s.env.l1.PendingNonceAt(t.Ctx(), s.address)
	// create a regular random tx on L1, append to L1 tx queue
	require.NoError(t, err, "failed to get L1 nonce for account %s", s.address)

	// L2 recipient address
	toIndex := s.selectedToAddr % uint64(len(s.env.addressCorpora))
	toAddr := s.env.addressCorpora[toIndex]
	isCreation := toIndex == 0

	// TODO randomize deposit contents
	value := big.NewInt(1_000_000_000)
	gasLimit := uint64(50_000)
	data := []byte{0x42}

	txOpts, err := bind.NewKeyedTransactorWithChainID(s.account, s.env.l1ChainID)
	require.NoError(t, err, "failed to create NewKeyedTransactorWithChainID for L1 deposit")

	txOpts.Nonce = new(big.Int).SetUint64(nonce)
	txOpts.NoSend = true
	// TODO: maybe change the txOpts L1 fee parameters

	tx, err := s.env.bindingPortal.DepositTransaction(txOpts, toAddr, value, gasLimit, isCreation, data)
	require.NoError(t, err, "failed to create deposit tx")

	err = s.env.l1.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send tx")
}

// add regular tx to L1 tx queue
func (s *User) actL1AddTx(t Testing) {
	// create a regular random tx on L1, append to L1 tx queue
	nonce, err := s.env.l1.PendingNonceAt(t.Ctx(), s.address)
	require.NoError(t, err, "failed to get L1 nonce for account %s", s.address)

	toIndex := s.selectedToAddr % uint64(len(s.env.addressCorpora))
	var to *common.Address
	if toIndex > 0 {
		to = &s.env.addressCorpora[toIndex]
	}
	// TODO: randomize tx contents

	gasTipCap := big.NewInt(2 * params.GWei)
	pendingHeader, err := s.env.l1.HeaderByNumber(t.Ctx(), big.NewInt(-1))
	require.NoError(t, err, "need l1 pending header for gas price estimation")
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(pendingHeader.BaseFee, big.NewInt(2)))

	gas, err := s.env.l1.EstimateGas(t.Ctx(), ethereum.CallMsg{
		From:      s.address,
		To:        to,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     big.NewInt(1_000_000_000),
	})
	require.NoError(t, err, "gas estimation should pass")
	tx := types.MustSignNewTx(s.account, s.env.l1Signer, &types.DynamicFeeTx{
		To:        to,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     big.NewInt(1_000_000_000),
		ChainID:   s.env.l1ChainID,
		Nonce:     nonce,
		Gas:       gas,
	})
	err = s.env.l1.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send tx")
}

// add regular tx to L2 tx queue
func (s *User) actL2AddTx(t Testing) {
	// create a regular random tx on L1, append to L1 tx queue
	nonce, err := s.env.l2.PendingNonceAt(t.Ctx(), s.address)
	require.NoError(t, err, "failed to get L2 nonce for account %s", s.address)

	toIndex := s.selectedToAddr % uint64(len(s.env.addressCorpora))
	var to *common.Address
	if toIndex > 0 {
		to = &s.env.addressCorpora[toIndex]
	}
	// TODO: randomize tx contents

	gasTipCap := big.NewInt(2 * params.GWei)
	pendingHeader, err := s.env.l2.HeaderByNumber(t.Ctx(), big.NewInt(-1))
	require.NoError(t, err, "need l1 pending header for gas price estimation")
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(pendingHeader.BaseFee, big.NewInt(2)))

	gas, err := s.env.l2.EstimateGas(t.Ctx(), ethereum.CallMsg{
		From:      s.address,
		To:        to,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     big.NewInt(1_000_000_000),
	})
	require.NoError(t, err, "gas estimation should pass")
	tx := types.MustSignNewTx(s.account, s.env.l2Signer, &types.DynamicFeeTx{
		To:        to,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     big.NewInt(1_000_000_000),
		ChainID:   s.env.l2ChainID,
		Nonce:     nonce,
		Gas:       gas,
	})
	err = s.env.l2.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "must send tx")
}
