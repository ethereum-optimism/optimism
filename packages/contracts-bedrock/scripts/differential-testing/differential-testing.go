package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ABI types
var (
	// Plain dynamic bytes type
	bytes, _ = abi.NewType("bytes", "bytes", []abi.ArgumentMarshaling{
		{Name: "data", Type: "bytes"},
	})
	bytesArgs = abi.Arguments{
		{Type: bytes},
	}

	// Plain fixed bytes32 type
	fixedBytes, _ = abi.NewType("bytes32", "bytes32", []abi.ArgumentMarshaling{
		{Name: "data", Type: "bytes32"},
	})
	fixedBytesArgs = abi.Arguments{
		{Type: fixedBytes},
	}

	// Decoded nonce tuple (nonce, version)
	decodedNonce, _ = abi.NewType("tuple", "DecodedNonce", []abi.ArgumentMarshaling{
		{Name: "nonce", Type: "uint256"},
		{Name: "version", Type: "uint256"},
	})
	decodedNonceArgs = abi.Arguments{
		{Name: "nonce", Type: decodedNonce},
	}

	// Withdrawal transaction tuple (uint256, address, address, uint256, uint256, bytes)
	withdrawalTransaction, _ = abi.NewType("tuple", "WithdrawalTransaction", []abi.ArgumentMarshaling{
		{Name: "nonce", Type: "uint256"},
		{Name: "sender", Type: "address"},
		{Name: "target", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "gasLimit", Type: "uint256"},
		{Name: "data", Type: "bytes"},
	})
	withdrawalTransactionArgs = abi.Arguments{
		{Name: "withdrawalTx", Type: withdrawalTransaction},
	}

	// Output root proof tuple (bytes32, bytes32, bytes32, bytes32)
	outputRootProof, _ = abi.NewType("tuple", "OutputRootProof", []abi.ArgumentMarshaling{
		{Name: "version", Type: "bytes32"},
		{Name: "stateRoot", Type: "bytes32"},
		{Name: "messagePasserStorageRoot", Type: "bytes32"},
		{Name: "latestBlockHash", Type: "bytes32"},
	})
	outputRootProofArgs = abi.Arguments{
		{Name: "proof", Type: outputRootProof},
	}
)

func main() {
	args := os.Args[1:]

	// This command requires arguments
	if len(args) == 0 {
		panic("Error: No arguments provided")
	}

	switch args[0] {
	case "decodeVersionedNonce":
		input, ok := new(big.Int).SetString(args[1], 10)
		checkOk(ok)

		// Decode versioned nonce
		nonce, version := crossdomain.DecodeVersionedNonce(input)

		// ABI encode output
		packArgs := struct {
			Nonce   *big.Int
			Version *big.Int
		}{
			nonce,
			version,
		}
		packed, err := decodedNonceArgs.Pack(&packArgs)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "encodeCrossDomainMessage":
		nonce, ok := new(big.Int).SetString(args[1], 10)
		checkOk(ok)
		sender := common.HexToAddress(args[2])
		target := common.HexToAddress(args[3])
		value, ok := new(big.Int).SetString(args[4], 10)
		checkOk(ok)
		gasLimit, ok := new(big.Int).SetString(args[5], 10)
		checkOk(ok)
		data := common.FromHex(args[6])

		// Encode cross domain message
		encoded, err := encodeCrossDomainMessage(nonce, sender, target, value, gasLimit, data)
		checkErr(err, fmt.Sprintf("Error encoding cross domain message: %s", err))

		// Pack encoded cross domain message
		packed, err := bytesArgs.Pack(&encoded)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "hashCrossDomainMessage":
		// Parse input arguments
		nonce, ok := new(big.Int).SetString(args[1], 10)
		checkOk(ok)
		sender := common.HexToAddress(args[2])
		target := common.HexToAddress(args[3])
		value, ok := new(big.Int).SetString(args[4], 10)
		checkOk(ok)
		gasLimit, ok := new(big.Int).SetString(args[5], 10)
		checkOk(ok)
		data := common.FromHex(args[6])

		// Encode cross domain message
		encoded, err := encodeCrossDomainMessage(nonce, sender, target, value, gasLimit, data)
		checkErr(err, fmt.Sprintf("Error encoding cross domain message: %s", err))

		// Hash encoded cross domain message
		hash := crypto.Keccak256Hash(encoded)

		// Pack hash
		packed, err := fixedBytesArgs.Pack(&hash)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "hashDepositTransaction":
		// Parse input arguments
		l1BlockHash := common.HexToHash(args[1])
		logIndex, ok := new(big.Int).SetString(args[2], 10)
		checkOk(ok)
		from := common.HexToAddress(args[3])
		to := common.HexToAddress(args[4])
		mint, ok := new(big.Int).SetString(args[5], 10)
		checkOk(ok)
		value, ok := new(big.Int).SetString(args[6], 10)
		checkOk(ok)
		gasLimit, ok := new(big.Int).SetString(args[7], 10)
		checkOk(ok)
		data := common.FromHex(args[8])

		// Create deposit transaction
		depositTx := makeDepositTx(from, to, value, mint, gasLimit, false, data, l1BlockHash, logIndex)

		// RLP encode deposit transaction
		encoded, err := types.NewTx(&depositTx).MarshalBinary()
		checkErr(err, fmt.Sprintf("Error encoding deposit transaction: %s", err))

		// Hash encoded deposit transaction
		hash := crypto.Keccak256Hash(encoded)

		// Pack hash
		packed, err := fixedBytesArgs.Pack(&hash)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "encodeDepositTransaction":
		// Parse input arguments
		from := common.HexToAddress(args[1])
		to := common.HexToAddress(args[2])
		value, ok := new(big.Int).SetString(args[3], 10)
		checkOk(ok)
		mint, ok := new(big.Int).SetString(args[4], 10)
		checkOk(ok)
		gasLimit, ok := new(big.Int).SetString(args[5], 10)
		checkOk(ok)
		isCreate := args[6] == "true"
		data := common.FromHex(args[7])
		l1BlockHash := common.HexToHash(args[8])
		logIndex, ok := new(big.Int).SetString(args[9], 10)
		checkOk(ok)

		depositTx := makeDepositTx(from, to, value, mint, gasLimit, isCreate, data, l1BlockHash, logIndex)

		// RLP encode deposit transaction
		encoded, err := types.NewTx(&depositTx).MarshalBinary()
		checkErr(err, fmt.Sprintf("Failed to RLP encode deposit transaction: %s", err))
		// Pack rlp encoded deposit transaction
		packed, err := bytesArgs.Pack(&encoded)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "hashWithdrawal":
		// Parse input arguments
		nonce, ok := new(big.Int).SetString(args[1], 10)
		checkOk(ok)
		sender := common.HexToAddress(args[2])
		target := common.HexToAddress(args[3])
		value, ok := new(big.Int).SetString(args[4], 10)
		checkOk(ok)
		gasLimit, ok := new(big.Int).SetString(args[5], 10)
		checkOk(ok)
		data := common.FromHex(args[6])

		// Pack withdrawal
		wdtx := struct {
			Nonce    *big.Int
			Sender   common.Address
			Target   common.Address
			Value    *big.Int
			GasLimit *big.Int
			Data     []byte
		}{
			Nonce:    nonce,
			Sender:   sender,
			Target:   target,
			Value:    value,
			GasLimit: gasLimit,
			Data:     data,
		}
		packed, err := withdrawalTransactionArgs.Pack(&wdtx)
		// println(hexutil.Encode(packed))
		checkErr(err, fmt.Sprintf("Error packing withdrawal: %s", err))

		// Hash packed withdrawal (we ignore the pointer)
		hash := crypto.Keccak256Hash(packed[32:])

		// Pack hash
		packed, err = fixedBytesArgs.Pack(&hash)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "hashOutputRootProof":
		// Parse input arguments
		version := common.HexToHash(args[1])
		stateRoot := common.HexToHash(args[2])
		messagePasserStorageRoot := common.HexToHash(args[3])
		latestBlockHash := common.HexToHash(args[4])

		// Pack proof
		proof := struct {
			Version                  common.Hash
			StateRoot                common.Hash
			MessagePasserStorageRoot common.Hash
			LatestBlockHash          common.Hash
		}{
			Version:                  version,
			StateRoot:                stateRoot,
			MessagePasserStorageRoot: messagePasserStorageRoot,
			LatestBlockHash:          latestBlockHash,
		}
		packed, err := outputRootProofArgs.Pack(&proof)
		checkErr(err, fmt.Sprintf("Error packing proof: %s", err))

		// Hash packed proof
		hash := crypto.Keccak256Hash(packed)

		// Pack hash
		packed, err = fixedBytesArgs.Pack(&hash)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "getProveWithdrawalTransactionInputs":
		// TODO
		break
	default:
		panic(fmt.Errorf("Unknown command: %s", args[0]))
	}
}
