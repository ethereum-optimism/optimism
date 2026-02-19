package utils

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const TokenVaultArtifact = "../contracts/artifacts/TokenVault.sol/TokenVault.json"
const BalanceSlotIndex = 0
const AllowanceSlotIndex = 1
const DepositorSlotIndex = 2

type TokenVault struct {
	*Contract
	t devtest.T
}

func (c *TokenVault) Deposit(user *dsl.EOA, amount eth.ETH) *types.Receipt {
	depositCalldata, err := c.Contract.parsedABI.Pack("deposit")
	if err != nil {
		require.NoError(c.t, err, "failed to pack deposit calldata")
	}
	depTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&c.Contract.address), txplan.WithData(depositCalldata), txplan.WithValue(amount))
	depRes, err := depTx.Included.Eval(c.t.Ctx())
	if err != nil {
		require.NoError(c.t, err, "deposit tx failed")
	}

	if depRes.Status != types.ReceiptStatusSuccessful {
		require.NoError(c.t, err, "deposit transaction failed")
	}

	return depRes
}

func (c *TokenVault) Approve(user *dsl.EOA, spender common.Address, amount *big.Int) *types.Receipt {
	approveCalldata, err := c.Contract.parsedABI.Pack("approve", spender, amount)
	if err != nil {
		require.NoError(c.t, err, "failed to pack approve calldata")
	}

	approveTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&c.Contract.address), txplan.WithData(approveCalldata))
	approveRes, err := approveTx.Included.Eval(c.t.Ctx())
	if err != nil {
		require.NoError(c.t, err, "approve tx failed")
	}

	if approveRes.Status != types.ReceiptStatusSuccessful {
		require.NoError(c.t, err, "approve transaction failed")
	}
	return approveRes
}

func (c *TokenVault) DeactivateAllowance(user *dsl.EOA, spender common.Address) *types.Receipt {
	deactCalldata, err := c.Contract.parsedABI.Pack("deactivateAllowance", spender)
	if err != nil {
		require.NoError(c.t, err, "failed to pack deactivateAllowance calldata")
	}
	deactTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&c.Contract.address), txplan.WithData(deactCalldata))
	deactRes, err := deactTx.Included.Eval(c.t.Ctx())
	if err != nil {
		require.NoError(c.t, err, "deactivateAllowance tx failed")
	}

	if deactRes.Status != types.ReceiptStatusSuccessful {
		require.NoError(c.t, err, "deactivateAllowance transaction failed")
	}
	return deactRes
}

func (c *TokenVault) GetBalanceSlot(user common.Address) common.Hash {
	keyBytes := common.LeftPadBytes(user.Bytes(), 32)
	slotBytes := common.LeftPadBytes(new(big.Int).SetUint64(BalanceSlotIndex).Bytes(), 32)
	return crypto.Keccak256Hash(append(keyBytes, slotBytes...))
}

func (c *TokenVault) GetAllowanceSlot(owner, spender common.Address) common.Hash {
	ownerBytes := common.LeftPadBytes(owner.Bytes(), 32)
	slotBytes := common.LeftPadBytes(new(big.Int).SetUint64(AllowanceSlotIndex).Bytes(), 32)
	inner := crypto.Keccak256(ownerBytes, slotBytes)
	spenderBytes := common.LeftPadBytes(spender.Bytes(), 32)
	return crypto.Keccak256Hash(append(spenderBytes, inner...))
}

func (c *TokenVault) GetDepositorSlot(index uint64) common.Hash {
	slotBytes := common.LeftPadBytes(new(big.Int).SetUint64(DepositorSlotIndex).Bytes(), 32)
	base := crypto.Keccak256(slotBytes)
	baseInt := new(big.Int).SetBytes(base)
	elem := new(big.Int).Add(baseInt, new(big.Int).SetUint64(index))
	return common.BigToHash(elem)
}

func DeployTokenVault(t devtest.T, user *dsl.EOA) (*TokenVault, *types.Receipt) {
	parsedABI, bin := LoadArtifact(t, TokenVaultArtifact)
	contractAddress, receipt := DeployContract(t, user, bin)
	contract := NewContract(contractAddress, parsedABI)
	return &TokenVault{contract, t}, receipt
}
