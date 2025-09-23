package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type ProvenWithdrawalsResult struct {
	DisputeGameProxy common.Address
	Timestamp        uint64
}

type WithdrawalTransaction struct {
	Nonce    *big.Int
	Sender   common.Address
	Target   common.Address
	Value    *big.Int
	GasLimit *big.Int
	Data     []byte
}

type OutputRootProof struct {
	Version                  [32]byte
	StateRoot                [32]byte
	MessagePasserStorageRoot [32]byte
	LatestBlockhash          [32]byte
}

type SuperRootProof struct {
	Version     [1]byte
	Timestamp   uint64
	OutputRoots []OutputRootWithChainID
}

type OutputRootWithChainID struct {
	ChainID *big.Int
	Root    [32]byte
}

type OptimismPortal2 struct {
	// Read-only functions
	CheckWithdrawal                 func(withdrawalHash [32]byte, proofSubmitter common.Address) TypedCall[any] `sol:"checkWithdrawal"`
	DisputeGameBlacklist            func(disputeGame common.Address) TypedCall[bool]                            `sol:"disputeGameBlacklist"`
	DisputeGameFactoryAddr          func() TypedCall[common.Address]                                            `sol:"disputeGameFactory"`
	DisputeGameFinalityDelaySeconds func() TypedCall[*big.Int]                                                  `sol:"disputeGameFinalityDelaySeconds"`
	FinalizedWithdrawals            func(withdrawalHash [32]byte) TypedCall[bool]                               `sol:"finalizedWithdrawals"`
	Guardian                        func() TypedCall[common.Address]                                            `sol:"guardian"`
	L2Sender                        func() TypedCall[common.Address]                                            `sol:"l2Sender"`
	MinimumGasLimit                 func(byteCount uint64) TypedCall[uint64]                                    `sol:"minimumGasLimit"`
	NumProofSubmitters              func(withdrawalHash [32]byte) TypedCall[*big.Int]                           `sol:"numProofSubmitters"`
	Params                          func() TypedCall[struct {
		PrevBaseFee   *big.Int
		PrevBoughtGas uint64
		PrevBlockNum  uint64
	}] `sol:"params"`
	Paused                     func() TypedCall[bool]                                                                     `sol:"paused"`
	SuperRootsActive           func() TypedCall[bool]                                                                     `sol:"superRootsActive"`
	ProofMaturityDelaySeconds  func() TypedCall[*big.Int]                                                                 `sol:"proofMaturityDelaySeconds"`
	ProofSubmitters            func(withdrawalHash [32]byte, index *big.Int) TypedCall[common.Address]                    `sol:"proofSubmitters"`
	ProvenWithdrawals          func(withdrawalHash [32]byte, submitter common.Address) TypedCall[ProvenWithdrawalsResult] `sol:"provenWithdrawals"`
	RespectedGameType          func() TypedCall[uint32]                                                                   `sol:"respectedGameType"`
	RespectedGameTypeUpdatedAt func() TypedCall[uint64]                                                                   `sol:"respectedGameTypeUpdatedAt"`
	SuperchainConfig           func() TypedCall[common.Address]                                                           `sol:"superchainConfig"`
	SystemConfig               func() TypedCall[common.Address]                                                           `sol:"systemConfig"`
	Version                    func() TypedCall[string]                                                                   `sol:"version"`

	// Write functions
	DepositTransaction            func(to common.Address, value eth.ETH, gaslimit uint64, isCreation bool, data []byte) TypedCall[any] `sol:"depositTransaction"`
	BlacklistDisputeGame          func(disputeGame common.Address) TypedCall[any]                                                      `sol:"blacklistDisputeGame"`
	DonateETH                     func() TypedCall[any]                                                                                `sol:"donateETH"`
	FinalizeWithdrawalTransaction func(tx struct {
		Nonce    *big.Int
		Sender   common.Address
		Target   common.Address
		Value    *big.Int
		GasLimit *big.Int
		Data     []byte
	}) TypedCall[any] `sol:"finalizeWithdrawalTransaction"`
	FinalizeWithdrawalTransactionExternalProof func(tx struct {
		Nonce    *big.Int
		Sender   common.Address
		Target   common.Address
		Value    *big.Int
		GasLimit *big.Int
		Data     []byte
	}, proofSubmitter common.Address) TypedCall[any] `sol:"finalizeWithdrawalTransactionExternalProof"`
	Initialize                          func(disputeGameFactory common.Address, systemConfig common.Address, superchainConfig common.Address) TypedCall[any]                                                                          `sol:"initialize"`
	ProveWithdrawalTransactionSuperRoot func(tx WithdrawalTransaction, disputeGame common.Address, outputRootIndex *big.Int, superRootProof SuperRootProof, outputRootProof OutputRootProof, withdrawalProof [][]byte) TypedCall[any] `sol:"proveWithdrawalTransaction"`
	ProveWithdrawalTransaction          func(tx WithdrawalTransaction, disputeGameIndex *big.Int, outputRootProof OutputRootProof, withdrawalProof [][]byte) TypedCall[any]                                                           `sol:"proveWithdrawalTransaction"`
	SetRespectedGameType                func(gameType uint32) TypedCall[any]                                                                                                                                                          `sol:"setRespectedGameType"`
	Receive                             func() TypedCall[any]                                                                                                                                                                         `sol:"receive"`
}
