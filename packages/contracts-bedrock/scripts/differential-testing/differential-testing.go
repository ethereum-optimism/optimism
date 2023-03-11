package main

import (
	"bytes"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
)

// ABI types
var (
	// Plain dynamic bytesAbi type
	bytesAbi, _ = abi.NewType("bytes", "bytes", []abi.ArgumentMarshaling{
		{Name: "data", Type: "bytes"},
	})
	bytesArgs = abi.Arguments{
		{Type: bytesAbi},
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

	// WithdrawalHash slot tuple (bytes32, bytes32)
	withdrawalSlot, _ = abi.NewType("tuple", "ArraySlotHash", []abi.ArgumentMarshaling{
		{Name: "withdrawalHash", Type: "bytes32"},
		{Name: "zeroPadding", Type: "bytes32"},
	})
	withdrawalSlotArgs = abi.Arguments{
		{Name: "slotHash", Type: withdrawalSlot},
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

	// Prove withdrawal inputs tuple (bytes32, bytes32, bytes32, bytes32, bytes[])
	proveWithdrawalInputs, _ = abi.NewType("tuple", "ProveWithdrawalInputs", []abi.ArgumentMarshaling{
		{Name: "worldRoot", Type: "bytes32"},
		{Name: "storageRoot", Type: "bytes32"},
		{Name: "outputRoot", Type: "bytes32"},
		{Name: "withdrawalHash", Type: "bytes32"},
		{Name: "proof", Type: "bytes[]"},
	})
	proveWithdrawalInputsArgs = abi.Arguments{
		{Name: "inputs", Type: proveWithdrawalInputs},
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
		// Parse input arguments
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

		// Hash withdrawal
		hash, err := hashWithdrawal(nonce, sender, target, value, gasLimit, data)
		checkErr(err, fmt.Sprintf("Error hashing withdrawal: %s", err))

		// Pack hash
		packed, err := fixedBytesArgs.Pack(&hash)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "hashOutputRootProof":
		// Parse input arguments
		version := common.HexToHash(args[1])
		stateRoot := common.HexToHash(args[2])
		messagePasserStorageRoot := common.HexToHash(args[3])
		latestBlockHash := common.HexToHash(args[4])

		// Hash the output root proof
		hash, err := hashOutputRootProof(version, stateRoot, messagePasserStorageRoot, latestBlockHash)

		// Pack hash
		packed, err := fixedBytesArgs.Pack(&hash)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		fmt.Print(hexutil.Encode(packed))
		break
	case "getProveWithdrawalTransactionInputs":
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

		wdHash, err := hashWithdrawal(nonce, sender, target, value, gasLimit, data)
		checkErr(err, fmt.Sprintf("Error hashing withdrawal: %s", err))

		// Compute the storage slot the withdrawalHash will be stored in
		slot := struct {
			WithdrawalHash common.Hash
			ZeroPadding    common.Hash
		}{
			WithdrawalHash: wdHash,
			ZeroPadding:    common.Hash{},
		}
		packed, err := withdrawalSlotArgs.Pack(&slot)
		checkErr(err, fmt.Sprintf("Error packing withdrawal slot: %s", err))

		// Compute the storage slot the withdrawalHash will be stored in
		hash := crypto.Keccak256Hash(packed)

		// Create a secure trie for storage
		storage, err := trie.NewStateTrie(
			trie.TrieID(common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")),
			trie.NewDatabase(rawdb.NewMemoryDatabase()),
		)
		checkErr(err, fmt.Sprintf("Error creating secure trie: %s", err))

		// Put a "true" bool in the storage slot
		storage.Update(hash.Bytes(), []byte{0x01})

		// Create a secure trie for the world state
		world, err := trie.NewStateTrie(
			trie.TrieID(common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")),
			trie.NewDatabase(rawdb.NewMemoryDatabase()),
		)
		checkErr(err, fmt.Sprintf("Error creating secure trie: %s", err))

		// Put the storage root into the L2ToL1MessagePasser storage
		address := common.HexToAddress("0x4200000000000000000000000000000000000016")
		account := types.StateAccount{
			Nonce:   0,
			Balance: big.NewInt(0),
			Root:    storage.Hash(),
		}

		writer := new(bytes.Buffer)
		checkErr(account.EncodeRLP(writer), fmt.Sprintf("Error encoding account: %s", err))
		world.Update(address.Bytes(), writer.Bytes())

		// Get the proof
		var proof proofList
		checkErr(storage.Prove(address.Bytes(), 0, &proof), fmt.Sprintf("Error getting proof: %s", err))

		// Get the output root
		outputRoot, err := hashOutputRootProof(common.Hash{}, world.Hash(), storage.Hash(), common.Hash{})

		// Pack the output
		output := struct {
			WorldRoot      common.Hash
			StorageRoot    common.Hash
			OutputRoot     common.Hash
			WithdrawalHash common.Hash
			Proof          [][]byte
		}{
			WorldRoot:      world.Hash(),
			StorageRoot:    storage.Hash(),
			OutputRoot:     outputRoot,
			WithdrawalHash: wdHash,
			Proof:          proof,
		}
		packed, err = proveWithdrawalInputsArgs.Pack(&output)
		checkErr(err, fmt.Sprintf("Error encoding output: %s", err))

		// Print the output
		fmt.Print(hexutil.Encode(packed[32:]))

		break
	default:
		panic(fmt.Errorf("Unknown command: %s", args[0]))
	}
}

// Custom type to write the generated proof to
type proofList [][]byte

func (n *proofList) Put(key []byte, value []byte) error {
	*n = append(*n, value)
	return nil
}

func (n *proofList) Delete(key []byte) error {
	panic("not supported")
}
