package crossdomain

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// NonceMask represents a mask used to extract version bytes from the nonce
	NonceMask, _ = new(big.Int).SetString("0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	// relayMessage0ABI represents the v0 relay message encoding
	relayMessage0ABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
	// relayMessage1ABI represents the v1 relay message encoding
	relayMessage1ABI = "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_minGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]"
	// relayMessage0 represents the ABI of relay message v0
	relayMessage0 abi.ABI
	// relayMessage1 represents the ABI of relay message v1
	relayMessage1 abi.ABI
)

// Create the required ABIs
func init() {
	var err error
	relayMessage0, err = abi.JSON(strings.NewReader(relayMessage0ABI))
	if err != nil {
		panic(err)
	}
	relayMessage1, err = abi.JSON(strings.NewReader(relayMessage1ABI))
	if err != nil {
		panic(err)
	}
}

// EncodeCrossDomainMessageV0 will encode the calldata for
// "relayMessage(address,address,bytes,uint256)",
func EncodeCrossDomainMessageV0(
	target common.Address,
	sender common.Address,
	message []byte,
	nonce *big.Int,
) ([]byte, error) {
	return relayMessage0.Pack("relayMessage", target, sender, message, nonce)
}

// EncodeCrossDomainMessageV1 will encode the calldata for
// "relayMessage(uint256,address,address,uint256,uint256,bytes)",
func EncodeCrossDomainMessageV1(
	nonce *big.Int,
	sender common.Address,
	target common.Address,
	value *big.Int,
	gasLimit *big.Int,
	data []byte,
) ([]byte, error) {
	return relayMessage1.Pack("relayMessage", nonce, sender, target, value, gasLimit, data)
}

// DecodeVersionedNonce will decode the version that is encoded in the nonce
func DecodeVersionedNonce(versioned *big.Int) (*big.Int, *big.Int) {
	nonce := new(big.Int).And(versioned, NonceMask)
	version := new(big.Int).Rsh(versioned, 240)
	return nonce, version
}

// EncodeVersionedNonce will encode the version into the nonce
func EncodeVersionedNonce(nonce, version *big.Int) *big.Int {
	shifted := new(big.Int).Lsh(version, 240)
	return new(big.Int).Or(nonce, shifted)
}
