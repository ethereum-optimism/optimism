package genesis

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	abiTrue  = common.Hash{31: 0x01}
	abiFalse = common.Hash{}
	// BedrockTransitionBlockExtraData represents the extradata
	// set in the very first bedrock block. This value must be
	// less than 32 bytes long or it will create an invalid block.
	// BedrockTransitionBlockExtraData = []byte("BEDROCK")
)

type MigrationResult struct {
	TransitionHeight    uint64
	TransitionTimestamp uint64
	TransitionBlockHash common.Hash
}

var EIP1559Denominator = uint64(1)
var EIP1559Elasticity = uint64(2)

// MigrateDB will migrate an l2geth legacy Optimism database to a Bedrock database.
func MigrateDB(ldb ethdb.Database, commit, noCheck bool) (*MigrationResult, error) {
	log.Info("Migrating DB")
	// Grab the hash of the tip of the legacy chain.
	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)

	// Grab the header number.
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return nil, fmt.Errorf("cannot find header number for %s", hash)
	}
	log.Info("Reading chain tip num from database", "number", num)

	// Grab the full header.
	// header := rawdb.ReadHeader(ldb, hash, *num)
	// log.Info("Read header from database", "number", *num)

	dbFactory := func() (*state.StateDB, error) {
		// Set up the backing store.
		underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
			Preimages: true,
			Cache:     1024,
		})

		// Open up the state database.
		db, err := state.New(hash, underlyingDB, nil)
		// db, err := state.New(header.Root, underlyingDB, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot open StateDB: %w", err)
		}

		return db, nil
	}

	db, err := dbFactory()
	if err != nil {
		return nil, fmt.Errorf("cannot create StateDB: %w", err)
	}

	fmt.Println("db", db)

	// Remove old blocks, so that we start with a fresh genesis block
	// currentHash := header.ParentHash
	// for {
	// 	// There are no uncles in Celo
	// 	num = rawdb.ReadHeaderNumber(ldb, currentHash)
	// 	hash = rawdb.ReadCanonicalHash(ldb, *num)
	// 	// header = rawdb.ReadHeader(ldb, currentHash, *num)
	// 	// if header == nil {
	// 	// 	return nil, fmt.Errorf("couldn't find header")
	// 	// }

	// 	log.Info("Deleting block", "hash", currentHash, "c", currentHash, "number", *num)
	// 	// rawdb.DeleteBlock(ldb, currentHash, *num)
	// 	if *num == 0 {
	// 		break
	// 	}

	// 	currentHash = header.ParentHash
	// }

	for i := *num; i >= 0; i-- {
		hash = rawdb.ReadCanonicalHash(ldb, i)
		log.Info("Deleting block", "hash", hash, "number", *num)

		// rawdb.DeleteBlock(ldb, currentHash, *num)
	}

	// Return the result and have a nice day.
	return nil, nil
}
