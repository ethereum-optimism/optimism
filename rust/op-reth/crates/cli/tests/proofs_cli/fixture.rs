//! A reth datadir seeded with a short OP chain for the proofs subcommands to run against.

use alloy_consensus::{BlockBody, EMPTY_ROOT_HASH, Header};
use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent, eip7685::Requests};
use alloy_primitives::{Address, B256, KECCAK256_EMPTY, U256, keccak256, map::HashMap};
use clap::Parser;
use reth_chainspec::EthChainSpec;
use reth_cli_commands::common::{AccessRights, EnvironmentArgs};
use reth_db::DatabaseEnv;
use reth_node_builder::NodeTypesWithDBAdapter;
use reth_optimism_cli::{Cli, chainspec::OpChainSpecParser, commands::Commands};
use reth_optimism_node::{
    OpNode,
    args::{ProofsHistoryStorageArgs, RollupArgs},
};
use reth_optimism_primitives::{OpBlock, OpReceipt};
use reth_optimism_trie::{
    BlockStateDiff, OpProofsBackfillStore, OpProofsProviderRO, OpProofsProviderRw,
    OpProofsSnapshotProviderRO, OpProofsStorageError, OpProofsStorageResult, OpProofsStore,
    ProofWindowRange, db::MdbxProofsStorage,
};
use reth_primitives_traits::{RecoveredBlock, SealedBlock};
use reth_provider::{
    BlockWriter, ChainSpecProvider, ExecutionOutcome, HashedPostStateProvider,
    LatestStateProviderRef, ProviderFactory, StateRootProvider, TrieWriter,
};
use reth_revm::{db::BundleState, state::AccountInfo};
use reth_tasks::Runtime;
use std::{
    path::PathBuf,
    sync::{Mutex, MutexGuard},
};
use tempfile::TempDir;

/// The account whose balance grows by one wei per block.
const FUNDED_ACCOUNT: Address = Address::repeat_byte(0x11);
/// The account whose storage slot [`STORAGE_SLOT`] takes the block number in every block.
const STORAGE_ACCOUNT: Address = Address::repeat_byte(0x22);
const STORAGE_SLOT: U256 = U256::ZERO;
/// `STOP`, so the storage account is a contract.
const STORAGE_ACCOUNT_CODE: [u8; 1] = [0x00];
const INITIAL_BALANCE: u128 = 1_000_000_000_000_000_000;
const CHAIN_ID: u64 = 901;
const BLOCK_TIME: u64 = 2;
/// What the header of a block whose body reth pruned looks like next to its empty body: a
/// transactions root that is not the empty root.
const PRUNED_BODY_TRANSACTIONS_ROOT: B256 = B256::repeat_byte(0xAB);

type OpProviderFactory = ProviderFactory<NodeTypesWithDBAdapter<OpNode, DatabaseEnv>>;

/// Seeded datadirs exist one at a time: reth and the proofs storage each map an 8 TB MDBX
/// geometry, and a dozen tests opening both in parallel threads of one process run out of
/// address space.
static DATADIR_IN_USE: Mutex<()> = Mutex::new(());

/// A datadir holding genesis and a chain of committed blocks, with the proofs storage at the
/// path the subcommands default to.
pub(crate) struct SeededDatadir {
    /// Hash of every committed block, indexed by block number.
    block_hashes: Vec<B256>,
    datadir: PathBuf,
    genesis_path: PathBuf,
    runtime: Runtime,
    _dir: TempDir,
    _in_use: MutexGuard<'static, ()>,
}

impl SeededDatadir {
    /// Seeds a datadir with `blocks` blocks on top of genesis. Every block changes the funded
    /// account's balance and the storage account's slot, so every block carries account and
    /// storage changes for the backfill and the snapshot to work with.
    pub(crate) fn with_blocks(blocks: u64) -> Self {
        Self::seed(blocks, None)
    }

    /// Like [`Self::with_blocks`], with block `pruned_body_block` stored the way reth stores a
    /// block whose body it pruned.
    pub(crate) fn with_blocks_and_pruned_body(blocks: u64, pruned_body_block: u64) -> Self {
        Self::seed(blocks, Some(pruned_body_block))
    }

    fn seed(blocks: u64, pruned_body_block: Option<u64>) -> Self {
        let in_use = DATADIR_IN_USE.lock().unwrap_or_else(|poisoned| poisoned.into_inner());
        let dir = TempDir::new().expect("temp dir");
        let genesis_path = dir.path().join("genesis.json");
        std::fs::write(&genesis_path, genesis_json()).expect("write genesis");
        let datadir = dir.path().join("data");
        let runtime = Runtime::test();

        let factory = EnvironmentArgs::<OpChainSpecParser>::parse_from([
            "op-reth",
            "--chain",
            genesis_path.to_str().expect("utf-8 path"),
            "--datadir",
            datadir.to_str().expect("utf-8 path"),
        ])
        .init::<OpNode>(AccessRights::RW, runtime.clone())
        .expect("initialize datadir")
        .provider_factory;

        let chain_spec = factory.chain_spec();
        let genesis_timestamp = chain_spec.genesis().timestamp;
        let mut block_hashes = vec![chain_spec.genesis_hash()];
        for number in 1..=blocks {
            let parent_hash = block_hashes[number as usize - 1];
            let timestamp = genesis_timestamp + number * BLOCK_TIME;
            let pruned_body = pruned_body_block == Some(number);
            block_hashes.push(append_block(&factory, number, parent_hash, timestamp, pruned_body));
        }
        drop(factory);

        Self { block_hashes, datadir, genesis_path, runtime, _dir: dir, _in_use: in_use }
    }

    /// Runs `op-reth proofs <args>` against this datadir in-process, the way the binary does.
    pub(crate) fn run(&self, args: &[&str]) -> eyre::Result<()> {
        let argv = ["op-reth", "proofs"].into_iter().chain(args.iter().copied()).chain([
            "--chain",
            self.genesis_path.to_str().expect("utf-8 path"),
            "--datadir",
            self.datadir.to_str().expect("utf-8 path"),
        ]);
        let cli = Cli::<OpChainSpecParser, RollupArgs>::try_parse_from(argv)?;
        let Commands::OpProofs(command) = cli.command else {
            unreachable!("the arguments name the proofs command")
        };
        self.runtime.handle().block_on(command.execute::<OpNode>(self.runtime.clone()))
    }

    /// The path the subcommands keep the proofs storage at when none is given.
    pub(crate) fn storage_path(&self) -> PathBuf {
        ProofsHistoryStorageArgs { storage_path: None, storage_version: None }
            .resolve_storage_path(&self.datadir)
    }

    /// The proof window the storage holds, or `None` while it holds no blocks.
    pub(crate) fn proof_window(&self) -> Option<ProofWindowRange> {
        let store = self.open_store();
        match store.provider_ro().expect("ro provider").get_proof_window() {
            Ok(window) => Some(window),
            Err(OpProofsStorageError::NoBlocksFound) => None,
            Err(err) => panic!("read proof window: {err}"),
        }
    }

    /// The window `[earliest, latest]` with the hashes reth holds for both blocks.
    pub(crate) fn window_of(&self, earliest: u64, latest: u64) -> ProofWindowRange {
        ProofWindowRange {
            earliest: NumHash::new(earliest, self.block_hash(earliest)),
            latest: NumHash::new(latest, self.block_hash(latest)),
        }
    }

    /// The block the storage's snapshot is anchored at, or the error a snapshot that is not
    /// ready yields.
    pub(crate) fn snapshot_anchor(&self) -> OpProofsStorageResult<BlockNumHash> {
        let store = self.open_store();
        store.snapshot_provider_ro()?.snapshot_anchor()
    }

    pub(crate) fn block_hash(&self, number: u64) -> B256 {
        self.block_hashes[number as usize]
    }

    /// Appends a block the datadir does not hold to the proofs storage, which leaves the storage
    /// ahead of reth the way an unclean shutdown does.
    pub(crate) fn append_store_only_block(&self) -> NumHash {
        let store = self.open_store();
        let latest = store.provider_ro().expect("ro provider").get_latest_block().expect("latest");
        let block = NumHash::new(latest.number + 1, keccak256((latest.number + 1).to_be_bytes()));
        let provider_rw = store.provider_rw().expect("rw provider");
        provider_rw
            .store_trie_updates(BlockWithParent::new(latest.hash, block), BlockStateDiff::default())
            .expect("store block");
        provider_rw.commit().expect("commit");
        block
    }

    fn open_store(&self) -> MdbxProofsStorage {
        MdbxProofsStorage::new(&self.storage_path()).expect("open proofs storage")
    }
}

/// Commits block `number` with its state changes, hashed state and trie updates, the way reth's
/// persistence does, and returns its hash.
fn append_block(
    factory: &OpProviderFactory,
    number: u64,
    parent_hash: B256,
    timestamp: u64,
    pruned_body: bool,
) -> B256 {
    let bundle = block_state_changes(number);
    let provider = factory.provider().expect("ro provider");
    let latest_state = LatestStateProviderRef::new(&provider);
    let hashed_state = latest_state.hashed_post_state(&bundle).expect("hashed post state");
    let (state_root, trie_updates) =
        latest_state.state_root_with_updates(hashed_state.clone()).expect("state root");
    drop(provider);

    let header = Header {
        base_fee_per_gas: Some(1_000_000_000),
        gas_limit: 30_000_000,
        number,
        parent_hash,
        state_root,
        timestamp,
        transactions_root: if pruned_body {
            PRUNED_BODY_TRANSACTIONS_ROOT
        } else {
            EMPTY_ROOT_HASH
        },
        ..Default::default()
    };
    let block = RecoveredBlock::new_sealed(
        SealedBlock::seal_slow(OpBlock { header, body: BlockBody::default() }),
        vec![],
    );
    let hash = block.hash();
    let outcome =
        ExecutionOutcome::<OpReceipt>::new(bundle, vec![vec![]], number, vec![Requests::default()]);

    let provider_rw = factory.provider_rw().expect("rw provider");
    provider_rw
        .append_blocks_with_state(vec![block], &outcome, hashed_state.into_sorted())
        .expect("append block");
    provider_rw.write_trie_updates(trie_updates).expect("write trie updates");
    provider_rw.commit().expect("commit block");
    hash
}

/// The state changes of block `number`: the funded account's balance goes from `number - 1` to
/// `number` wei above its initial balance and the storage account's slot from `number - 1` to
/// `number`.
fn block_state_changes(number: u64) -> BundleState {
    let funded_before = funded_account(number - 1);
    let funded_after = funded_account(number);
    let storage_account = storage_account();
    let (slot_before, slot_after) = (U256::from(number - 1), U256::from(number));
    BundleState::new(
        [
            (FUNDED_ACCOUNT, Some(funded_before.clone()), Some(funded_after), HashMap::default()),
            (
                STORAGE_ACCOUNT,
                Some(storage_account.clone()),
                Some(storage_account),
                HashMap::from_iter([(STORAGE_SLOT, (slot_before, slot_after))]),
            ),
        ],
        [[
            (FUNDED_ACCOUNT, Some(Some(funded_before)), Vec::<(U256, U256)>::new()),
            (STORAGE_ACCOUNT, None, vec![(STORAGE_SLOT, slot_before)]),
        ]],
        std::iter::empty(),
    )
}

fn funded_account(blocks_in: u64) -> AccountInfo {
    AccountInfo {
        balance: U256::from(INITIAL_BALANCE) + U256::from(blocks_in),
        code_hash: KECCAK256_EMPTY,
        nonce: 0,
        ..Default::default()
    }
}

fn storage_account() -> AccountInfo {
    AccountInfo {
        balance: U256::ZERO,
        code_hash: keccak256(STORAGE_ACCOUNT_CODE),
        nonce: 0,
        ..Default::default()
    }
}

/// A genesis with every OP fork active, the funded account and the storage account.
fn genesis_json() -> String {
    format!(
        r#"{{
  "config": {{
    "chainId": {CHAIN_ID},
    "homesteadBlock": 0, "eip150Block": 0, "eip155Block": 0, "eip158Block": 0,
    "byzantiumBlock": 0, "constantinopleBlock": 0, "petersburgBlock": 0, "istanbulBlock": 0,
    "muirGlacierBlock": 0, "berlinBlock": 0, "londonBlock": 0, "arrowGlacierBlock": 0,
    "grayGlacierBlock": 0, "mergeNetsplitBlock": 0,
    "terminalTotalDifficulty": 0, "terminalTotalDifficultyPassed": true,
    "bedrockBlock": 0, "regolithTime": 0, "canyonTime": 0, "ecotoneTime": 0, "fjordTime": 0,
    "graniteTime": 0, "holoceneTime": 0, "isthmusTime": 0,
    "optimism": {{ "eip1559Elasticity": 6, "eip1559Denominator": 50, "eip1559DenominatorCanyon": 250 }}
  }},
  "nonce": "0x0",
  "timestamp": "0x0",
  "extraData": "0x",
  "gasLimit": "0x1c9c380",
  "difficulty": "0x0",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "baseFeePerGas": "0x3b9aca00",
  "alloc": {{
    "{FUNDED_ACCOUNT}": {{ "balance": "{:#x}" }},
    "{STORAGE_ACCOUNT}": {{ "balance": "0x0", "code": "0x00" }}
  }}
}}"#,
        INITIAL_BALANCE
    )
}
