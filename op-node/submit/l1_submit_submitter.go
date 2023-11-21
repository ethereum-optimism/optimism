package submit

import (
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

// L1SubmitTxData creates the transaction data for the L1Submit function
func L1SubmitTxData(address common.Address, index uint64, commitment, sign hexutil.Bytes) ([]byte, error) {
	parsed, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return l1SubmitTxData(parsed, address, index, commitment, sign)
}

func l1SubmitTxData(abi *abi.ABI, address common.Address, index uint64, commitment, sign []byte) ([]byte, error) {
	return abi.Pack(
		"SubmitCommitment",
		address,
		new(big.Int).SetUint64(index),
		commitment)
}
