package genesis

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/rawdbv3"
	"github.com/ledgerwatch/erigon/boba-chain-ops/crossdomain"
	"github.com/ledgerwatch/erigon/boba-chain-ops/ether"
	"github.com/ledgerwatch/erigon/consensus/ethash"
	"github.com/ledgerwatch/erigon/consensus/serenity"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

func MigrateDB(chaindb kv.RwDB, genesis *types.Genesis, config *DeployConfig, blockHeader *types.Header, migrationData *crossdomain.MigrationData, commit, noCheck bool) error {
	// Before we do anything else, we need to ensure that all of the input configuration is correct
	// and nothing is missing. We'll first verify the contract configuration, then we'll verify the
	// witness data for the migration. We operate under the assumption that the witness data is
	// untrusted and must be verified explicitly before we can use it.

	// Generate and verify the configuration for storage variables to be set on L2.
	storage, err := NewL2StorageConfig(config, blockHeader)
	if err != nil {
		return fmt.Errorf("cannot create storage config: %w", err)
	}

	// Generate and verify the configuration for immutable variables to be set on L2.
	immutable, err := NewL2ImmutableConfig(config, blockHeader)
	if err != nil {
		return fmt.Errorf("cannot create immutable config: %w", err)
	}
	log.Debug("Created L2 configuration", "storage", storage, "immutable", immutable)

	// Convert all input messages into legacy messages. Note that this list is not yet filtered and
	// may be missing some messages or have some extra messages.
	unfilteredWithdrawals, invalidMessages, err := migrationData.ToWithdrawals()
	if err != nil {
		return fmt.Errorf("cannot serialize withdrawals: %w", err)
	}

	log.Info("Read withdrawals from witness data", "unfiltered", len(unfilteredWithdrawals), "invalid", len(invalidMessages))

	// We now need to check that we have all of the withdrawals that we expect to have. An error
	// will be thrown if there are any missing messages, and any extra messages will be removed.
	var filteredWithdrawals crossdomain.SafeFilteredWithdrawals
	if !noCheck {
		log.Info("Checking withdrawals...")
		filteredWithdrawals, err = crossdomain.PreCheckWithdrawals(genesis, unfilteredWithdrawals, invalidMessages)
		if err != nil {
			return fmt.Errorf("withdrawals mismatch: %w", err)
		}
	} else {
		log.Info("Skipping checking withdrawals")
		filteredWithdrawals = crossdomain.SafeFilteredWithdrawals(unfilteredWithdrawals)
	}

	log.Info("Filtered withdrawals", "filtered", len(filteredWithdrawals))

	// At this point we've fully verified the witness data for the migration, so we can begin the
	// actual migration process.

	// We need to wipe the storage of every predeployed contract EXCEPT for the GovernanceToken,
	// WETH9, the DeployerWhitelist, the LegacyMessagePasser, and LegacyERC20ETH. We have verified
	// that none of the legacy storage (other than the aforementioned contracts) is accessible and
	// therefore can be safely removed from the database. Storage must be wiped before anything
	// else or the ERC-1967 proxy storage slots will be removed.
	if err := WipePredeployStorage(genesis); err != nil {
		return fmt.Errorf("cannot wipe storage: %w", err)
	}

	// Next order of business is to convert all predeployed smart contracts into proxies so they
	// can be easily upgraded later on. In the legacy system, all upgrades to predeployed contracts
	// required hard forks which was a huge pain. Note that we do NOT put the GovernanceToken or
	// WETH9 contracts behind proxies because we do not want to make these easily upgradable.
	log.Info("Converting predeployed contracts to proxies")
	if err := SetL2Proxies(genesis); err != nil {
		return fmt.Errorf("cannot set L2Proxies: %w", err)
	}

	// Here we update the storage of each predeploy with the new storage variables that we want to
	// set on L2 and update the implementations for all predeployed contracts that are behind
	// proxies (NOT the GovernanceToken or WETH9).
	log.Info("Updating implementations for predeployed contracts")
	if err := SetImplementations(genesis, storage, immutable); err != nil {
		return fmt.Errorf("cannot set implementations: %w", err)
	}

	// We need to update the code for LegacyERC20ETH. This is NOT a standard predeploy because it's
	// deployed at the 0xdeaddeaddead... address and therefore won't be updated by the previous
	// function call to SetImplementations.
	log.Info("Updating code for LegacyERC20ETH")
	if err := SetLegacyETH(genesis, storage, immutable); err != nil {
		return fmt.Errorf("cannot set legacy ETH: %w", err)
	}

	// Now we migrate legacy withdrawals from the LegacyMessagePasser contract to their new format
	// in the Bedrock L2ToL1MessagePasser contract. Note that we do NOT delete the withdrawals from
	// the LegacyMessagePasser contract. Here we operate on the list of withdrawals that we
	// previously filtered and verified.
	log.Info("Starting to migrate withdrawals", "no-check", noCheck)
	err = crossdomain.MigrateWithdrawals(filteredWithdrawals, genesis, &config.L1CrossDomainMessengerProxy, noCheck)
	if err != nil {
		return fmt.Errorf("cannot migrate withdrawals: %w", err)
	}

	// Finally we migrate the balances held inside the LegacyERC20ETH contract into the state trie.
	// We also delete the balances from the LegacyERC20ETH contract. Unlike the steps above, this step
	// combines the check and mutation steps into one in order to reduce migration time.
	log.Info("Starting to migrate ERC20 ETH")
	err = ether.MigrateBalances(genesis, migrationData.Addresses(), migrationData.OvmAllowances, noCheck)
	if err != nil {
		return fmt.Errorf("failed to migrate OVM_ETH: %w", err)
	}

	if !commit {
		log.Info("Dry run complete!")
		return nil
	}

	if err = WriteGenesis(chaindb, genesis); err != nil {
		return err
	}

	return nil
}

// Write genesis to chaindb
func WriteGenesis(chaindb kv.RwDB, genesis *types.Genesis) error {
	tx, err := chaindb.BeginRw(context.Background())
	if err != nil {
		log.Error("failed to begin write genesis block", "err", err)
		return err
	}
	defer tx.Rollback()

	hash, err := rawdb.ReadCanonicalHash(tx, 0)
	if err != nil {
		log.Error("failed to read canonical hash of block #0", "err", err)
		return err
	}

	if (hash != common.Hash{}) {
		log.Error("genesis block already exists")
		return errors.New("genesis block already exists")
	}

	header, err := CreateHeader(genesis)
	if err != nil {
		log.Error("failed to create header from genesis config", "err", err)
		return err
	}

	statedb, err := AllocToGenesis(genesis, header)
	if err != nil {
		log.Error("failed to create genesis state", "err", err)
		return err
	}

	block := types.NewBlock(header, nil, nil, nil, []*types.Withdrawal{})

	var stateWriter state.StateWriter
	for addr, account := range genesis.Alloc {
		if len(account.Code) > 0 || len(account.Storage) > 0 {
			// Special case for weird tests - inaccessible storage
			var b [8]byte
			binary.BigEndian.PutUint64(b[:], state.FirstContractIncarnation)
			if err := tx.Put(kv.IncarnationMap, addr[:], b[:]); err != nil {
				return err
			}
		}
	}

	stateWriter = state.NewPlainStateWriter(tx, tx, 0)

	if block.Number().Sign() != 0 {
		return fmt.Errorf("genesis block number is not 0")
	}

	if err := statedb.CommitBlock(&chain.Rules{}, stateWriter); err != nil {
		return fmt.Errorf("cannot commit genesis block: %w", err)
	}
	if csw, ok := stateWriter.(state.WriterWithChangeSets); ok {
		if err := csw.WriteChangeSets(); err != nil {
			return fmt.Errorf("cannot write changesets: %w", err)
		}
		if err := csw.WriteHistory(); err != nil {
			return fmt.Errorf("cannot write history: %w", err)
		}
	}

	if err := CommitGenesisBlock(tx, genesis, "", block, statedb); err != nil {
		log.Error("failed to write genesis block", "err", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error("failed to commit genesis block", "err", err)
		return err
	}
	log.Info("Successfully wrote genesis state", "hash", block.Hash())

	return nil
}

// Write writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func CommitGenesisBlock(tx kv.RwTx, g *types.Genesis, tmpDir string, block *types.Block, statedb *state.IntraBlockState) error {
	config := g.Config
	if config == nil {
		config = params.AllProtocolChanges
	}
	if err := config.CheckConfigForkOrder(); err != nil {
		return err
	}
	if err := rawdb.WriteTd(tx, block.Hash(), block.NumberU64(), g.Difficulty); err != nil {
		return err
	}
	if err := rawdb.WriteBlock(tx, block); err != nil {
		return err
	}
	if err := rawdbv3.TxNums.WriteForGenesis(tx, 1); err != nil {
		return err
	}
	if err := rawdb.WriteReceipts(tx, block.NumberU64(), nil); err != nil {
		return err
	}

	if err := rawdb.WriteCanonicalHash(tx, block.Hash(), block.NumberU64()); err != nil {
		return err
	}

	rawdb.WriteHeadBlockHash(tx, block.Hash())
	if err := rawdb.WriteHeadHeaderHash(tx, block.Hash()); err != nil {
		return err
	}
	if err := rawdb.WriteChainConfig(tx, block.Hash(), config); err != nil {
		return err
	}
	// We support ethash/serenity for issuance (for now)
	if g.Config.Consensus != chain.EtHashConsensus {
		return nil
	}
	// Issuance is the sum of allocs
	genesisIssuance := big.NewInt(0)
	for _, account := range g.Alloc {
		genesisIssuance.Add(genesisIssuance, account.Balance)
	}

	// BlockReward can be present at genesis
	if block.Header().Difficulty.Cmp(serenity.SerenityDifficulty) == 0 {
		// Proof-of-stake is 0.3 ether per block (TODO: revisit)
		genesisIssuance.Add(genesisIssuance, serenity.RewardSerenity)
	} else {
		blockReward, _ := ethash.AccumulateRewards(g.Config, block.Header(), nil)
		// Set BlockReward
		genesisIssuance.Add(genesisIssuance, blockReward.ToBig())
	}
	if err := rawdb.WriteTotalIssued(tx, 0, genesisIssuance); err != nil {
		return err
	}
	if err := rawdb.WriteTotalBurnt(tx, 0, common.Big0); err != nil {
		return err
	}

	log.Info("genesis block is written to database")

	return nil
}
