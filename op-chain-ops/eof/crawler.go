package eof

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Account represents an account in the state.
type Account struct {
	Balance   string         `json:"balance"`
	Nonce     uint64         `json:"nonce"`
	Root      hexutil.Bytes  `json:"root"`
	CodeHash  hexutil.Bytes  `json:"codeHash"`
	Code      hexutil.Bytes  `json:"code,omitempty"`
	Address   common.Address `json:"address,omitempty"`
	SecureKey hexutil.Bytes  `json:"key,omitempty"`
}

// emptyCodeHash is the known hash of an account with no code.
var emptyCodeHash = crypto.Keccak256(nil)

// IndexEOFContracts indexes all the EOF contracts in the state trie of the head block
// for the given db and writes them to a JSON file.
func IndexEOFContracts(dbPath string, out string) error {
	// Open an existing Ethereum database
	db, err := rawdb.NewLevelDBDatabase(dbPath, 16, 16, "", true)
	if err != nil {
		return fmt.Errorf("Failed to open database: %w", err)
	}
	stateDB := state.NewDatabase(db)

	// Retrieve the head block
	hash := rawdb.ReadHeadBlockHash(db)
	number := rawdb.ReadHeaderNumber(db, hash)
	if number == nil {
		return errors.New("Failed to retrieve head block number")
	}
	head := rawdb.ReadBlock(db, hash, *number)
	if head == nil {
		return errors.New("Failed to retrieve head block")
	}

	// Retrieve the state belonging to the head block
	st, err := trie.New(trie.StateTrieID(head.Root()), trie.NewDatabase(db))
	if err != nil {
		return fmt.Errorf("Failed to retrieve state trie: %w", err)
	}
	log.Printf("Indexing state trie at head block #%d [0x%x]", *number, hash)

	// Iterate over the entire account trie to search for EOF-prefixed contracts
	start := time.Now()
	missingPreimages := uint64(0)
	eoas := uint64(0)
	nonEofContracts := uint64(0)
	eofContracts := make([]Account, 0)

	it := trie.NewIterator(st.NodeIterator(nil))
	for it.Next() {
		// Decode the state account
		var data types.StateAccount
		err := rlp.DecodeBytes(it.Value, &data)
		if err != nil {
			return fmt.Errorf("Failed to decode state account: %w", err)
		}

		// Check to see if the account has any code associated with it before performing
		// more reads from the trie & db.
		if bytes.Equal(data.CodeHash, emptyCodeHash) {
			eoas++
			continue
		}

		// Create a serializable `Account` object
		account := Account{
			Balance:   data.Balance.String(),
			Nonce:     data.Nonce,
			Root:      data.Root[:],
			CodeHash:  data.CodeHash,
			SecureKey: it.Key,
		}

		// Attempt to get the address of the account from the trie
		addrBytes, err := st.Get(it.Key)
		if err != nil {
			return fmt.Errorf("load address for account: %w", err)
		}
		if addrBytes == nil {
			// Preimage missing! Cannot continue.
			missingPreimages++
			continue
		}
		addr := common.BytesToAddress(addrBytes)

		// Attempt to get the code of the account from the trie
		code, err := stateDB.ContractCode(crypto.Keccak256Hash(addrBytes), common.BytesToHash(data.CodeHash))
		if err != nil {
			return fmt.Errorf("Could not load code for account %x: %w", addr, err)
		}

		// Check if the contract's runtime bytecode starts with the EOF prefix.
		if len(code) >= 1 && code[0] == 0xEF {
			// Append the account to the list of EOF contracts
			account.Address = addr
			account.Code = code
			eofContracts = append(eofContracts, account)
		} else {
			nonEofContracts++
		}
	}

	// Print finishing status
	log.Printf("Indexing done in %v, found %d EOF contracts", time.Since(start), len(eofContracts))
	log.Printf("Num missing preimages: %d", missingPreimages)
	log.Printf("Non-EOF-prefixed contracts: %d", nonEofContracts)
	log.Printf("Accounts with no code (EOAs): %d", eoas)

	// Write the EOF contracts to a file
	file, err := json.MarshalIndent(eofContracts, "", " ")
	if err != nil {
		return fmt.Errorf("Cannot marshal EOF contracts: %w", err)
	}
	err = os.WriteFile(out, file, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write EOF contracts array to file: %w", err)
	}

	log.Printf("Wrote list of EOF contracts to `%v`", out)
	return nil
}
