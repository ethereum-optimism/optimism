//! The test chain the live targets share: a funded signer, block and transaction specs, the
//! block builder, the store constructors and the store's initialization at a block.

use super::store_view::ProofTarget;
use alloy_consensus::{
    Header, TxEip2930,
    constants::{EMPTY_WITHDRAWALS, ETH_TO_WEI},
};
use alloy_eips::{BlockNumHash, eip4895::Withdrawals};
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, keccak256};
use derive_more::Constructor;
use reth_chainspec::{ChainSpec, ChainSpecBuilder, EthereumHardfork, EthereumHardforks, MAINNET};
use reth_db::Database;
use reth_ethereum_primitives::{Block, BlockBody, Transaction, TransactionSigned};
use reth_node_api::NodeTypesWithDB;
use reth_optimism_trie::{
    MdbxProofsStorage, OpProofsStore, RethTrieStorageLayout, initialize::InitializationJob,
};
use reth_primitives_traits::{Block as _, RecoveredBlock};
use reth_provider::{ProviderFactory, StorageSettingsCache};
use secp256k1::Keypair;
use std::sync::Arc;
use tempfile::TempDir;

pub(super) use reth_chainspec::MIN_TRANSACTION_GAS;

/// Seconds between consecutive blocks.
const BLOCK_TIME: u64 = 2;

/// A proof window no test chain fills, so that the pruner never moves the earliest block.
pub(super) const WHOLE_WINDOW: u64 = 1000;

/// Timestamp of the block at `number`.
pub(super) const fn block_timestamp(number: u64) -> u64 {
    number * BLOCK_TIME
}

/// The engine's persistence and backpressure thresholds.
#[derive(Clone, Copy, Debug)]
pub(super) struct FlushCadence {
    pub(super) backpressure: u64,
    pub(super) persistence: u64,
}

impl FlushCadence {
    pub(super) const EVERY_BLOCK: Self = Self { backpressure: 2, persistence: 1 };
    pub(super) const EVERY_FIFTH_BLOCK: Self = Self { backpressure: 10, persistence: 5 };
}

/// Which store implementation a case runs over.
#[derive(Clone, Copy, Debug)]
pub(super) enum Backend {
    InMemory,
    Mdbx,
}

pub(super) fn create_mdbx_proofs_storage() -> Arc<MdbxProofsStorage> {
    let path = TempDir::new().unwrap().keep();
    Arc::new(MdbxProofsStorage::new(&path).unwrap())
}

/// Converts a secp256k1 public key to an Ethereum address.
pub(super) fn public_key_to_address(pubkey: secp256k1::PublicKey) -> Address {
    let hash = keccak256(&pubkey.serialize_uncompressed()[1..]);
    Address::from_slice(&hash[12..])
}

/// Signs a transaction with the given keypair.
fn sign_tx_with_key_pair(key_pair: Keypair, tx: Transaction) -> TransactionSigned {
    use alloy_consensus::SignableTransaction;
    use reth_primitives_traits::crypto::secp256k1::sign_message;
    let secret = B256::from_slice(&key_pair.secret_bytes());
    let sig = sign_message(secret, tx.signature_hash()).unwrap();
    tx.into_signed(sig).into()
}

/// Specification for a transaction within a block
#[derive(Debug, Clone)]
pub(super) struct TxSpec {
    /// Call recipient or contract creation.
    pub(super) to: TxKind,
    /// Transaction input, including initcode for deployments.
    pub(super) data: Bytes,
    /// Maximum execution gas.
    pub(super) gas_limit: u64,
    /// Value to transfer (in wei)
    pub(super) value: U256,
    /// Nonce for the transaction (will be automatically assigned if None)
    pub(super) nonce: Option<u64>,
}

impl TxSpec {
    /// Create a simple transfer transaction
    pub(super) const fn transfer(to: Address, value: U256) -> Self {
        Self {
            to: TxKind::Call(to),
            data: Bytes::new(),
            gas_limit: MIN_TRANSACTION_GAS,
            value,
            nonce: None,
        }
    }

    /// Create a contract deployment running `initcode`.
    pub(super) const fn create(initcode: Bytes, gas_limit: u64) -> Self {
        Self { to: TxKind::Create, data: initcode, gas_limit, value: U256::ZERO, nonce: None }
    }

    /// Create a call to `contract` with `data` as input.
    pub(super) const fn call(contract: Address, data: Bytes, gas_limit: u64) -> Self {
        Self { to: TxKind::Call(contract), data, gas_limit, value: U256::ZERO, nonce: None }
    }

    /// Send `value` along with the transaction.
    pub(super) fn with_value(self, value: U256) -> Self {
        Self { value, ..self }
    }
}

/// Specification for a block in the test chain
#[derive(Debug, Clone, Constructor)]
pub(super) struct BlockSpec {
    /// Transactions to include in this block
    pub(super) txs: Vec<TxSpec>,
}

impl BlockSpec {
    /// A block without transactions.
    pub(super) const fn empty() -> Self {
        Self { txs: Vec::new() }
    }
}

/// Recipients the transfer chain pays in turn.
pub(super) const RECIPIENTS: u64 = 4;

pub(super) const fn recipient(serial: u64) -> Address {
    Address::repeat_byte(0x40 + (serial % RECIPIENTS) as u8)
}

/// The block with `serial`: `serial` wei to its recipient, so that no two blocks are alike.
pub(super) fn serial_transfer_block(serial: u64) -> BlockSpec {
    BlockSpec::new(vec![TxSpec::transfer(recipient(serial), U256::from(serial))])
}

/// Proof targets for the transfer chain: the sender, the zero address and every recipient.
pub(super) fn proof_targets_for_transfers(sender: Address) -> Vec<ProofTarget> {
    ProofTarget::accounts([sender, Address::ZERO].into_iter().chain((0..RECIPIENTS).map(recipient)))
}

/// A mainnet-shaped chain spec whose genesis allocation is `alloc`, with the hardforks still to
/// be chosen.
pub(super) fn chain_spec_builder(
    alloc: impl IntoIterator<Item = (Address, GenesisAccount)>,
) -> ChainSpecBuilder {
    ChainSpecBuilder::default()
        .chain(MAINNET.chain)
        .genesis(Genesis { alloc: alloc.into_iter().collect(), ..MAINNET.genesis.clone() })
}

/// A genesis allocation funding `address`.
pub(super) fn funded(address: Address) -> (Address, GenesisAccount) {
    (address, GenesisAccount { balance: U256::from(10 * ETH_TO_WEI), ..Default::default() })
}

/// Helper to create a chain spec with a genesis account funded
pub(super) fn chain_spec_with_address(address: Address) -> Arc<ChainSpec> {
    Arc::new(chain_spec_builder([funded(address)]).paris_activated().build())
}

/// Creates a block from a spec, executing transactions with the given keypair
pub(super) fn create_block_from_spec(
    spec: &BlockSpec,
    block_number: u64,
    parent_hash: B256,
    chain_spec: &Arc<ChainSpec>,
    key_pair: Keypair,
    nonce_counter: &mut u64,
) -> RecoveredBlock<Block> {
    let transactions: Vec<TransactionSigned> = spec
        .txs
        .iter()
        .map(|tx_spec| {
            let nonce = tx_spec.nonce.unwrap_or_else(|| {
                let current = *nonce_counter;
                *nonce_counter += 1;
                current
            });

            sign_tx_with_key_pair(
                key_pair,
                TxEip2930 {
                    chain_id: chain_spec.chain.id(),
                    nonce,
                    gas_limit: tx_spec.gas_limit,
                    input: tx_spec.data.clone(),
                    gas_price: 1_500_000_000,
                    to: tx_spec.to,
                    value: tx_spec.value,
                    ..Default::default()
                }
                .into(),
            )
        })
        .collect();

    let gas_total = spec.txs.iter().map(|tx| tx.gas_limit).sum::<u64>();
    let timestamp = block_timestamp(block_number);
    let shanghai = chain_spec.is_shanghai_active_at_timestamp(timestamp);
    let cancun = chain_spec.is_cancun_active_at_timestamp(timestamp);

    Block {
        header: Header {
            parent_hash,
            receipts_root: alloy_primitives::b256!(
                "0xd3a6acf9a244d78b33831df95d472c4128ea85bf079a1d41e32ed0b7d2244c9e"
            ),
            difficulty: chain_spec.fork(EthereumHardfork::Paris).ttd().expect("Paris TTD"),
            number: block_number,
            gas_limit: gas_total.max(MIN_TRANSACTION_GAS),
            gas_used: gas_total,
            timestamp,
            state_root: B256::ZERO, // Will be calculated by executor
            withdrawals_root: shanghai.then_some(EMPTY_WITHDRAWALS),
            blob_gas_used: cancun.then_some(0),
            excess_blob_gas: cancun.then_some(0),
            parent_beacon_block_root: cancun.then(|| keccak256(block_number.to_be_bytes())),
            ..Default::default()
        },
        body: BlockBody {
            transactions,
            withdrawals: shanghai.then(Withdrawals::default),
            ..Default::default()
        },
    }
    .try_into_recovered()
    .unwrap()
}

/// Seeds `storage` with reth's state and trie at `block`, the proofs store's starting point.
pub(super) fn initialize_storage<N, S>(
    storage: S,
    provider_factory: &ProviderFactory<N>,
    block: BlockNumHash,
) -> eyre::Result<()>
where
    N: NodeTypesWithDB,
    S: OpProofsStore + Send + Sync + Clone + std::fmt::Debug + 'static,
{
    let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
        RethTrieStorageLayout::Packed
    } else {
        RethTrieStorageLayout::Legacy
    };
    let tx = provider_factory.db_ref().tx()?;
    InitializationJob::new(storage, tx, trie_layout).run(block.number, block.hash)?;
    Ok(())
}
