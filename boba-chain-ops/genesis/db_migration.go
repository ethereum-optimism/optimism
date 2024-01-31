package genesis

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/crossdomain"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/ether"
	"github.com/ledgerwatch/erigon-lib/chain"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/length"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/rawdbv3"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/turbo/trie"
	"github.com/ledgerwatch/log/v3"
)

func MigrateDB(chaindb kv.RwDB, genesis *types.Genesis, config *DeployConfig, blockHeader *types.Header, migrationData *crossdomain.MigrationData, commit, noCheck bool) (*types.Block, error) {
	// Before we do anything else, we need to ensure that all of the input configuration is correct
	// and nothing is missing. We'll first verify the contract configuration, then we'll verify the
	// witness data for the migration. We operate under the assumption that the witness data is
	// untrusted and must be verified explicitly before we can use it.

	// Generate and verify the configuration for storage variables to be set on L2.
	storage, err := NewL2StorageConfig(config, blockHeader)
	if err != nil {
		return nil, fmt.Errorf("cannot create storage config: %w", err)
	}

	// Generate and verify the configuration for immutable variables to be set on L2.
	immutable, err := NewL2ImmutableConfig(config, blockHeader)
	if err != nil {
		return nil, fmt.Errorf("cannot create immutable config: %w", err)
	}
	log.Debug("Created L2 configuration", "storage", storage, "immutable", immutable)

	// Convert all input messages into legacy messages. Note that this list is not yet filtered and
	// may be missing some messages or have some extra messages.
	unfilteredWithdrawals, invalidMessages, err := migrationData.ToWithdrawals()
	if err != nil {
		return nil, fmt.Errorf("cannot serialize withdrawals: %w", err)
	}

	log.Info("Read withdrawals from witness data", "unfiltered", len(unfilteredWithdrawals), "invalid", len(invalidMessages))

	// We now need to check that we have all of the withdrawals that we expect to have. An error
	// will be thrown if there are any missing messages, and any extra messages will be removed.
	var filteredWithdrawals crossdomain.SafeFilteredWithdrawals
	if !noCheck {
		log.Info("Checking withdrawals...")
		filteredWithdrawals, err = crossdomain.PreCheckWithdrawals(genesis, unfilteredWithdrawals, invalidMessages)
		if err != nil {
			return nil, fmt.Errorf("withdrawals mismatch: %w", err)
		}
	} else {
		log.Info("Skipping checking withdrawals")
		filteredWithdrawals = crossdomain.SafeFilteredWithdrawals(unfilteredWithdrawals)
	}

	log.Info("Filtered withdrawals", "filtered", len(filteredWithdrawals))

	// At this point, we have verified that the witness data is correct and retrieved the legacy
	// credit from the genesis. We can now start to mutate the genesis to prepare it for the

	// We need to wipe the storage of every predeployed contract EXCEPT for the GovernanceToken,
	// WETH9, the DeployerWhitelist, the LegacyMessagePasser, and LegacyERC20ETH. We have verified
	// that none of the legacy storage (other than the aforementioned contracts) is accessible and
	// therefore can be safely removed from the database. Storage must be wiped before anything
	// else or the ERC-1967 proxy storage slots will be removed.
	if err := WipePredeployStorage(genesis); err != nil {
		return nil, fmt.Errorf("cannot wipe storage: %w", err)
	}

	// Next order of business is to convert all predeployed smart contracts into proxies so they
	// can be easily upgraded later on. In the legacy system, all upgrades to predeployed contracts
	// required hard forks which was a huge pain. Note that we do NOT put the GovernanceToken or
	// WETH9 contracts behind proxies because we do not want to make these easily upgradable.
	log.Info("Converting predeployed contracts to proxies")
	if err := SetL2Proxies(genesis); err != nil {
		return nil, fmt.Errorf("cannot set L2Proxies: %w", err)
	}

	// Here we update the storage of each predeploy with the new storage variables that we want to
	// set on L2 and update the implementations for all predeployed contracts that are behind
	// proxies (NOT the GovernanceToken or WETH9).
	log.Info("Updating implementations for predeployed contracts")
	if err := SetImplementations(genesis, storage, immutable); err != nil {
		return nil, fmt.Errorf("cannot set implementations: %w", err)
	}

	// We need to update the code for LegacyERC20ETH. This is NOT a standard predeploy because it's
	// deployed at the 0xdeaddeaddead... address and therefore won't be updated by the previous
	// function call to SetImplementations.
	log.Info("Updating code for LegacyERC20ETH")
	if err := SetLegacyETH(genesis, storage, immutable); err != nil {
		return nil, fmt.Errorf("cannot set legacy ETH: %w", err)
	}

	// Now we migrate legacy withdrawals from the LegacyMessagePasser contract to their new format
	// in the Bedrock L2ToL1MessagePasser contract. Note that we do NOT delete the withdrawals from
	// the LegacyMessagePasser contract. Here we operate on the list of withdrawals that we
	// previously filtered and verified.
	log.Info("Starting to migrate withdrawals", "no-check", noCheck)
	err = crossdomain.MigrateWithdrawals(filteredWithdrawals, genesis, &config.L1CrossDomainMessengerProxy, noCheck)
	if err != nil {
		return nil, fmt.Errorf("cannot migrate withdrawals: %w", err)
	}

	// We migrate the balances held inside the LegacyERC20ETH contract into the state trie.
	// We also delete the balances from the LegacyERC20ETH contract. Unlike the steps above, this step
	// combines the check and mutation steps into one in order to reduce migration time.
	log.Info("Starting to migrate ERC20 ETH")
	err = ether.MigrateBalances(genesis, migrationData.Addresses(), migrationData.OvmAllowances, noCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate OVM_ETH: %w", err)
	}

	if !commit {
		log.Info("Dry run complete!")
		return nil, nil
	}

	block, err := WriteGenesis(chaindb, genesis, config)
	if err != nil {
		return nil, fmt.Errorf("cannot write genesis: %w", err)
	}

	return block, nil
}

// Write genesis to chaindb
func WriteGenesis(chaindb kv.RwDB, genesis *types.Genesis, config *DeployConfig) (*types.Block, error) {
	tx, err := chaindb.BeginRw(context.Background())
	if err != nil {
		log.Error("failed to begin write genesis block", "err", err)
		return nil, err
	}
	defer tx.Rollback()

	parentHeader := rawdb.ReadCurrentHeader(tx)
	if config.L2OutputOracleStartingBlockNumber == parentHeader.Number.Uint64() {
		return nil, fmt.Errorf("cannot write genesis: genesis block already exists")
	}
	if config.L2OutputOracleStartingBlockNumber != parentHeader.Number.Uint64()+1 {
		return nil, fmt.Errorf("L2OutputOracleStartingBlockNumber must be %d", parentHeader.Number.Uint64()+1)
	}
	transitionBlockNumber := config.L2OutputOracleStartingBlockNumber

	var root libcommon.Hash
	header, err := CreateHeader(genesis, parentHeader, config)
	if err != nil {
		log.Error("failed to create header from genesis config", "err", err)
		return nil, err
	}

	statedb, root, err := AllocToGenesis(genesis, header)
	if err != nil {
		log.Error("failed to create genesis state", "err", err)
		return nil, err
	}

	header.Root = root
	block := types.NewBlock(header, nil, nil, nil, nil)

	var stateWriter state.StateWriter
	for addr, account := range genesis.Alloc {
		if len(account.Code) > 0 || len(account.Storage) > 0 {
			// Special case for weird tests - inaccessible storage
			var b [8]byte
			binary.BigEndian.PutUint64(b[:], state.FirstContractIncarnation)
			if err := tx.Put(kv.IncarnationMap, addr[:], b[:]); err != nil {
				return nil, err
			}
		}
	}

	stateWriter = state.NewPlainStateWriter(tx, tx, transitionBlockNumber)

	if err := statedb.CommitBlock(&chain.Rules{}, stateWriter); err != nil {
		return nil, fmt.Errorf("cannot commit genesis block: %w", err)
	}
	if csw, ok := stateWriter.(state.WriterWithChangeSets); ok {
		if err := csw.WriteChangeSets(); err != nil {
			return nil, fmt.Errorf("cannot write changesets: %w", err)
		}
		if err := csw.WriteHistory(); err != nil {
			return nil, fmt.Errorf("cannot write history: %w", err)
		}
	}

	if err := CommitGenesisBlock(tx, genesis, "", block, statedb); err != nil {
		log.Error("failed to write genesis block", "err", err)
		return nil, err
	}

	if err := CommitHashedState(tx); err != nil {
		log.Error("failed to write hashed state", "err", err)
		return nil, err
	}

	// verify state root
	root, err = trie.CalcRoot("transition", tx)
	if err != nil {
		return nil, err
	}
	if root != header.Root {
		return nil, fmt.Errorf("state root mismatch: %x != %x", root, header.Root)
	}

	// save StageProgress
	if err := stages.SaveStageProgress(tx, stages.IntermediateHashes, transitionBlockNumber); err != nil {
		return nil, err
	}
	if err := stages.SaveStageProgress(tx, stages.Execution, transitionBlockNumber); err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Error("failed to commit genesis block", "err", err)
		return nil, err
	}
	log.Info("Successfully wrote genesis state", "hash", block.Hash())

	return block, nil
}

// Write the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func CommitGenesisBlock(tx kv.RwTx, g *types.Genesis, tmpDir string, block *types.Block, statedb *state.IntraBlockState) error {
	config := g.Config
	if config == nil {
		config = params.AllProtocolChanges
	}
	if err := config.CheckConfigForkOrder(); err != nil {
		return err
	}
	if err := rawdb.WriteTd(tx, block.Hash(), block.NumberU64(), big.NewInt(3)); err != nil {
		return err
	}
	if err := rawdb.WriteBlock(tx, block); err != nil {
		return err
	}
	if err := rawdbv3.TxNums.Append(tx, block.NumberU64(), 0); err != nil {
		return err
	}
	if err := rawdb.WriteReceipts(tx, block.NumberU64(), []*types.Receipt{}); err != nil {
		return err
	}

	if err := rawdb.WriteCanonicalHash(tx, block.Hash(), block.NumberU64()); err != nil {
		return err
	}

	rawdb.WriteHeadBlockHash(tx, block.Hash())
	if err := rawdb.WriteHeadHeaderHash(tx, block.Hash()); err != nil {
		return err
	}
	if err := rawdb.WriteHeaderNumber(tx, block.Hash(), block.NumberU64()); err != nil {
		fmt.Println("Failed to write WriteHeaderNumber")
		panic(err)
	}

	rawdb.WriteForkchoiceHead(tx, block.Hash())
	rawdb.WriteForkchoiceFinalized(tx, block.Hash())
	rawdb.WriteForkchoiceSafe(tx, block.Hash())

	// override chain config in the genesis block, so we can avoid changes in
	// the erigon
	hash, err := rawdb.ReadCanonicalHash(tx, 0)
	if err != nil {
		return err
	}
	if err := rawdb.WriteChainConfig(tx, hash, config); err != nil {
		return err
	}

	log.Info("genesis block is written to database")

	return nil
}

// Write hashedStorage and hashedAccounts to database
func CommitHashedState(tx kv.RwTx) error {
	cursor, err := tx.RwCursor(kv.PlainState)
	if err != nil {
		return err
	}
	defer cursor.Close()

	h := libcommon.NewHasher()
	defer libcommon.ReturnHasherToPool(h)
	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return fmt.Errorf("interate over plain state: %w", err)
		}
		var newK []byte
		if len(k) == length.Addr {
			newK = make([]byte, length.Hash)
		} else {
			newK = make([]byte, length.Hash*2+length.Incarnation)
		}
		h.Sha.Reset()
		//nolint:errcheck
		h.Sha.Write(k[:length.Addr])
		//nolint:errcheck
		h.Sha.Read(newK[:length.Hash])
		if len(k) > length.Addr {
			copy(newK[length.Hash:], k[length.Addr:length.Addr+length.Incarnation])
			h.Sha.Reset()
			//nolint:errcheck
			h.Sha.Write(k[length.Addr+length.Incarnation:])
			//nolint:errcheck
			h.Sha.Read(newK[length.Hash+length.Incarnation:])
			if err = tx.Put(kv.HashedStorage, newK, libcommon.CopyBytes(v)); err != nil {
				return fmt.Errorf("insert hashed key: %w", err)
			}
		} else {
			if err = tx.Put(kv.HashedAccounts, newK, libcommon.CopyBytes(v)); err != nil {
				return fmt.Errorf("insert hashed key: %w", err)
			}
		}
	}

	return nil
}
