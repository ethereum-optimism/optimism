//! Property tests that pool admission and the payload builder agree.
//!
//! Pool admission and the builder are tested separately elsewhere, and this crate is the only place
//! that can reach both.
//!
//! Each case runs a real validator and [`Pool`] and the real builder against the same head, L1
//! info, base fee, balance and nonce, then builds an otherwise empty block on the real L1-info
//! deposit.
//!
//! Failures reproduce from the printed seed. `PROPTEST_RNG_SEED` and `PROPTEST_CASES` override the
//! fixed seed and the case count, so a scheduled job can explore further than a PR run does.

use super::{OpPayloadBuilderCtx, RethPayloadTransactions, tests::op_pooled_tx_from};
use crate::{
    OpPayloadBuilderAttributes,
    config::{OpBuilderConfig, OpDAConfig, OpGasLimitConfig},
};
use alloy_consensus::{Header, Sealable, Transaction, TxEip1559};
use alloy_eips::{
    eip1559::BaseFeeParams,
    eip2718::{Encodable2718, WithEncoded},
};
use alloy_evm::Evm;
use alloy_primitives::{Address, B64, B256, Bytes, StorageKey, TxKind, U256, b256, map::HashMap};
use op_alloy_consensus::{TxDeposit, encode_jovian_extra_data};
use op_revm::{
    L1BlockInfo, OpSpecId,
    constants::{
        BASE_FEE_SCALAR_OFFSET, BLOB_BASE_FEE_SCALAR_OFFSET, DA_FOOTPRINT_GAS_SCALAR_OFFSET,
        DA_FOOTPRINT_GAS_SCALAR_SLOT, ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT,
        L1_BASE_FEE_SLOT, L1_BLOCK_CONTRACT, OPERATOR_FEE_CONSTANT_OFFSET,
        OPERATOR_FEE_SCALAR_OFFSET,
    },
};
use proptest::{prelude::*, test_runner::RngSeed};
use reth_basic_payload_builder::PayloadConfig;
use reth_chainspec::EthChainSpec;
use reth_evm::execute::BlockBuilder;
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_evm::{L1_BLOCK_JOVIAN_SELECTOR, OpEvmConfig, extract_l1_info_from_tx};
use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
use reth_optimism_txpool::{
    OpPooledTransaction, OpTransactionValidator, estimated_da_size::DataAvailabilitySized,
};
use reth_payload_util::PayloadTransactionsFixed;
use reth_primitives_traits::{Account, SealedHeader};
use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
use reth_revm::{database::StateProviderDatabase, db::State, test_utils::StateProviderTest};
use reth_storage_api::StateProvider;
use reth_transaction_pool::{
    BlockInfo, CoinbaseTipOrdering, Pool, PoolConfig, PoolTransaction, TransactionOrigin,
    TransactionPool, TransactionPoolExt, blobstore::InMemoryBlobStore,
    validate::EthTransactionValidatorBuilder,
};
use revm::context::Block as _;
use std::sync::Arc;

/// Runs of these tests are reproducible unless `PROPTEST_RNG_SEED` says otherwise.
const DEFAULT_SEED: u64 = 0xDA_F00D;

/// The sender of the transaction under test.
const SIGNER: Address = Address::repeat_byte(0x11);
/// The sender's account nonce, which the pool and the executor both see.
const STATE_NONCE: u64 = 7;

/// The gas limit of the L1-info deposit.
const L1_INFO_DEPOSIT_GAS: u64 = 1_000_000;
/// Mirrors the pool's reservation for the L1-info deposit (`L1_INFO_GAS_OVERHEAD`).
const L1_INFO_GAS_OVERHEAD: u64 = 70_000;
/// Every block has to fit the L1-info deposit, or it fails before any pool transaction is tried.
const MIN_BLOCK_GAS_LIMIT: u64 = 2 * L1_INFO_DEPOSIT_GAS;
/// The largest calldata that stays under reth's default 128 KiB pool limit, encoding included.
const MAX_ADMISSIBLE_CALLDATA: usize = 130_000;

/// The block's base fee. The parent header carries it, and the parent's gas usage keeps it flat.
const BASE_FEE: u64 = 1_000;

/// Values that the L1-info deposit sets, and which both sides must see.
#[derive(Debug, Clone, Copy)]
struct L1Params {
    /// L1 base fee, in wei.
    l1_base_fee: u64,
    /// Scalar on the L1 base fee in the L1 data fee.
    base_fee_scalar: u32,
    /// L1 blob base fee, in wei.
    l1_blob_base_fee: u64,
    /// Scalar on the L1 blob base fee in the L1 data fee.
    blob_base_fee_scalar: u32,
    /// Scalar of the operator fee.
    operator_fee_scalar: u32,
    /// Flat part of the operator fee.
    operator_fee_constant: u64,
    /// Gas counted towards the DA footprint per estimated DA byte.
    da_footprint_gas_scalar: u16,
}

impl L1Params {
    /// `setL1BlockValuesJovian` calldata. The validator reads this from the deposit.
    fn calldata(&self) -> Bytes {
        let mut data = L1_BLOCK_JOVIAN_SELECTOR.to_vec();
        data.extend_from_slice(&self.base_fee_scalar.to_be_bytes());
        data.extend_from_slice(&self.blob_base_fee_scalar.to_be_bytes());
        data.extend_from_slice(&[0; 24]); // sequence number, L1 timestamp, L1 number
        data.extend_from_slice(&U256::from(self.l1_base_fee).to_be_bytes::<32>());
        data.extend_from_slice(&U256::from(self.l1_blob_base_fee).to_be_bytes::<32>());
        data.extend_from_slice(&[0; 64]); // L1 block hash, batcher hash
        data.extend_from_slice(&self.operator_fee_scalar.to_be_bytes());
        data.extend_from_slice(&self.operator_fee_constant.to_be_bytes());
        data.extend_from_slice(&self.da_footprint_gas_scalar.to_be_bytes());
        data.into()
    }

    /// `L1Block` storage. The executor reads this from state.
    ///
    /// The test state has no `L1Block` bytecode, so the deposit doesn't write these. Instead they
    /// are in place from the start of the block, as they would be once the deposit has run.
    fn storage(&self) -> HashMap<StorageKey, U256> {
        let mut fee_scalars = [0u8; 32];
        fee_scalars[BASE_FEE_SCALAR_OFFSET..BASE_FEE_SCALAR_OFFSET + 4]
            .copy_from_slice(&self.base_fee_scalar.to_be_bytes());
        fee_scalars[BLOB_BASE_FEE_SCALAR_OFFSET..BLOB_BASE_FEE_SCALAR_OFFSET + 4]
            .copy_from_slice(&self.blob_base_fee_scalar.to_be_bytes());

        let mut operator_and_da = [0u8; 32];
        operator_and_da[DA_FOOTPRINT_GAS_SCALAR_OFFSET..DA_FOOTPRINT_GAS_SCALAR_OFFSET + 2]
            .copy_from_slice(&self.da_footprint_gas_scalar.to_be_bytes());
        operator_and_da[OPERATOR_FEE_SCALAR_OFFSET..OPERATOR_FEE_SCALAR_OFFSET + 4]
            .copy_from_slice(&self.operator_fee_scalar.to_be_bytes());
        operator_and_da[OPERATOR_FEE_CONSTANT_OFFSET..OPERATOR_FEE_CONSTANT_OFFSET + 8]
            .copy_from_slice(&self.operator_fee_constant.to_be_bytes());

        [
            (L1_BASE_FEE_SLOT, U256::from(self.l1_base_fee)),
            (ECOTONE_L1_BLOB_BASE_FEE_SLOT, U256::from(self.l1_blob_base_fee)),
            (ECOTONE_L1_FEE_SCALARS_SLOT, U256::from_be_bytes(fee_scalars)),
            (DA_FOOTPRINT_GAS_SCALAR_SLOT, U256::from_be_bytes(operator_and_da)),
        ]
        .into_iter()
        .map(|(slot, value)| (StorageKey::from(slot), value))
        .collect()
    }

    /// The L1-info deposit that opens the block, carrying [`Self::calldata`].
    fn deposit(&self) -> OpTransactionSigned {
        OpTransactionSigned::Deposit(
            TxDeposit {
                source_hash: b256!(
                    "0x1111111111111111111111111111111111111111111111111111111111111111"
                ),
                from: Address::repeat_byte(0xde),
                to: TxKind::Call(L1_BLOCK_CONTRACT),
                gas_limit: L1_INFO_DEPOSIT_GAS,
                input: self.calldata(),
                ..Default::default()
            }
            .seal_slow(),
        )
    }
}

/// A limit set relative to the transaction, so that cases land on the boundaries of each check.
///
/// Picking the transaction first and deriving the limit is what makes a footprint exactly at the
/// limit reachable: random limits would almost never hit it.
#[derive(Debug, Clone, Copy)]
enum Limit {
    /// The footprint plus an offset.
    Footprint(i64),
    /// The declared gas plus the L1-info reservation, plus an offset.
    Gas(i64),
    /// Above both of the others by this much.
    Above(u64),
    /// A value unrelated to the transaction.
    Fixed(u64),
}

impl Limit {
    /// The limit in gas, given the transaction's footprint and declared gas.
    fn resolve(self, footprint: u64, gas: u64) -> u64 {
        let gas_bound = gas + L1_INFO_GAS_OVERHEAD;
        match self {
            Self::Footprint(offset) => footprint.saturating_add_signed(offset),
            Self::Gas(offset) => gas_bound.saturating_add_signed(offset),
            Self::Above(margin) => footprint.max(gas_bound).saturating_add(margin),
            Self::Fixed(limit) => limit,
        }
    }
}

/// The sender's balance.
#[derive(Debug, Clone, Copy)]
enum Balance {
    /// The full cost of the transaction, L1 and operator fees included, plus an offset.
    Cost(i64),
    /// More than the transaction could cost.
    Plenty,
}

/// Knobs that shrink independently, which keeps counterexamples readable.
#[derive(Debug, Clone)]
struct Case {
    /// The fork active from genesis.
    fork: Fork,
    /// Calldata length in bytes.
    calldata_len: usize,
    /// Seed of the calldata's noise.
    calldata_seed: u64,
    /// Roughly what percentage of the calldata is zero bytes, and so how well it compresses.
    calldata_zero_pct: u8,
    /// Declared gas on top of what the calldata requires.
    extra_gas: u64,
    /// Max fee relative to the block's base fee.
    fee_offset: i64,
    /// Whether to leave a gap below this nonce, which parks the transaction in `Queued`.
    nonce_gap: bool,
    /// The sender's balance.
    balance: Balance,
    /// What the L1-info deposit sets.
    l1: L1Params,
    /// The header gas limit.
    block_gas_limit: Limit,
    /// `miner_setGasLimit`.
    operator_gas_limit: Option<Limit>,
    /// `miner_setMaxDASize`, per transaction.
    da_tx_limit: Option<u64>,
    /// `miner_setMaxDASize`, per block.
    da_block_limit: Option<u64>,
    /// `--rollup.max-uncompressed-block-size`.
    max_uncompressed_block_size: Option<u64>,
}

impl Case {
    /// Generates the calldata from its length, seed and share of zero bytes.
    fn calldata(&self) -> Bytes {
        let mut state = self.calldata_seed | 1;
        (0..self.calldata_len)
            .map(|_| {
                state ^= state << 13;
                state ^= state >> 7;
                state ^= state << 17;
                if (state >> 8) % 100 < u64::from(self.calldata_zero_pct) { 0 } else { state as u8 }
            })
            .collect()
    }

    /// The transaction under test, test-signed and recovered to [`SIGNER`].
    fn tx(&self) -> OpPooledTransaction {
        let input = self.calldata();
        let tx = TxEip1559 {
            chain_id: self.fork.chain_spec().chain().id(),
            nonce: STATE_NONCE + u64::from(self.nonce_gap),
            gas_limit: min_gas(&input) + self.extra_gas,
            max_fee_per_gas: u128::from(BASE_FEE.saturating_add_signed(self.fee_offset)),
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(Address::repeat_byte(0x42)),
            input,
            ..Default::default()
        };
        op_pooled_tx_from(SIGNER, tx)
    }

    /// The DA footprint of `tx` under this case's scalar.
    fn footprint(&self, tx: &OpPooledTransaction) -> u64 {
        tx.estimated_da_size().saturating_mul(u64::from(self.l1.da_footprint_gas_scalar))
    }

    /// The sender's balance, relative to what `tx` costs including L1 and operator fees.
    fn balance(&self, tx: &OpPooledTransaction) -> U256 {
        let offset = match self.balance {
            Balance::Cost(offset) => offset,
            Balance::Plenty => return U256::from(u128::MAX),
        };
        let mut l1_info =
            extract_l1_info_from_tx(&self.l1.deposit()).expect("valid Jovian L1-info calldata");
        let op_fees =
            l1_info.tx_cost(tx.encoded_2718(), U256::from(tx.gas_limit()), self.fork.spec_id());
        let cost = tx.cost().saturating_add(op_fees);
        if offset < 0 {
            cost.saturating_sub(U256::from(offset.unsigned_abs()))
        } else {
            cost.saturating_add(U256::from(offset))
        }
    }

    /// Resolves this case's limits and balance against `tx` into concrete values.
    fn env(&self, tx: &OpPooledTransaction) -> Env {
        let footprint = self.footprint(tx);
        let gas = tx.gas_limit();
        let block_gas_limit = self.block_gas_limit.resolve(footprint, gas).max(MIN_BLOCK_GAS_LIMIT);
        // Zero means "unset" to `OpGasLimitConfig`.
        let operator_gas_limit = self.operator_gas_limit.map(|l| l.resolve(footprint, gas).max(1));
        let builder_config = OpBuilderConfig {
            da_config: OpDAConfig::new(
                self.da_tx_limit.unwrap_or(0),
                self.da_block_limit.unwrap_or(0),
            ),
            gas_limit_config: OpGasLimitConfig::new(operator_gas_limit.unwrap_or(0)),
            max_uncompressed_block_size: self.max_uncompressed_block_size,
            ..Default::default()
        };
        Env::new(self.fork, self.l1, self.balance(tx), block_gas_limit, builder_config)
    }

    /// Whether a limit that admission deliberately doesn't enforce rules out `tx`.
    fn throttled(&self, tx: &OpPooledTransaction, deposit: &OpTransactionSigned) -> bool {
        let da_size = tx.estimated_da_size();
        let uncompressed = tx.encoded_2718().len() as u64 + deposit.encode_2718_len() as u64;
        self.da_tx_limit.is_some_and(|limit| da_size > limit) ||
            self.da_block_limit.is_some_and(|limit| da_size > limit) ||
            self.max_uncompressed_block_size.is_some_and(|limit| uncompressed > limit)
    }
}

/// The least gas a call with this calldata can declare: EIP-2028 intrinsic gas or the EIP-7623
/// floor, whichever is higher.
fn min_gas(input: &[u8]) -> u64 {
    let zeros: u64 = input.iter().map(|byte| u64::from(*byte == 0)).sum();
    let non_zeros = input.len() as u64 - zeros;
    let intrinsic = 21_000 + 4 * zeros + 16 * non_zeros;
    let floor = 21_000 + 10 * (zeros + 4 * non_zeros);
    intrinsic.max(floor)
}

/// The fork active from genesis.
#[derive(Debug, Clone, Copy)]
enum Fork {
    /// Introduces the DA footprint limit.
    Jovian,
    /// What OP Mainnet runs now. Adds the EIP-7825 cap on a transaction's gas, through Osaka.
    Karst,
}

impl Fork {
    /// OP Mainnet's chain spec with this fork active from genesis.
    fn chain_spec(self) -> Arc<OpChainSpec> {
        let builder = OpChainSpecBuilder::optimism_mainnet();
        Arc::new(
            match self {
                Self::Jovian => builder.jovian_activated(),
                Self::Karst => builder.karst_activated(),
            }
            .build(),
        )
    }

    /// The EVM spec of this fork.
    const fn spec_id(self) -> OpSpecId {
        match self {
            Self::Jovian => OpSpecId::JOVIAN,
            Self::Karst => OpSpecId::KARST,
        }
    }
}

/// What the pool and the builder made of a case.
#[derive(Debug)]
struct Report {
    /// Estimated DA size times the scalar.
    footprint: u64,
    /// The gas the transaction declares.
    gas: u64,
    /// The header gas limit, once resolved.
    block_gas_limit: u64,
    /// `miner_setGasLimit`, once resolved.
    operator_gas_limit: Option<u64>,
    /// The transaction reached the pending subpool.
    reached_pending: bool,
    /// Total transactions in all subpools.
    pool_size: usize,
    /// An otherwise empty block included it. Only tried if it reached pending.
    included: bool,
    /// An operator limit the admission check deliberately doesn't enforce rules it out.
    throttled: bool,
}

impl Report {
    /// The block gas budget the builder works with.
    fn effective_gas_limit(&self) -> u64 {
        self.operator_gas_limit.map_or(self.block_gas_limit, |l| l.min(self.block_gas_limit))
    }

    /// The transaction can't fit in any block this builder would make.
    fn exceeds_effective_limit(&self) -> bool {
        let limit = self.effective_gas_limit();
        self.footprint > limit || self.gas > limit.saturating_sub(L1_INFO_GAS_OVERHEAD)
    }
}

/// The header of the block being built on, at `block_gas_limit`.
fn parent_header(block_gas_limit: u64) -> SealedHeader {
    SealedHeader::seal_slow(Header {
        gas_limit: block_gas_limit,
        number: 0,
        timestamp: 0,
        base_fee_per_gas: Some(BASE_FEE),
        // Using exactly the EIP-1559 gas target, the limit over the elasticity of 2 in
        // `extra_data`, keeps the base fee flat. The transaction's fee is set before the
        // builder exists, so `BASE_FEE` has to be known up front.
        gas_used: block_gas_limit / 2,
        extra_data: encode_jovian_extra_data(B64::ZERO, BaseFeeParams::new(250, 2), 0)
            .expect("valid EIP-1559 params"),
        ..Default::default()
    })
}

/// The chain, head and accounts that a block gets built on, independent of any one transaction.
///
/// The pool, the builder and the state are all made from this, which is what keeps them on the
/// same head, L1 info, base fee, balance and nonce.
struct Env {
    /// The chain both sides run.
    chain_spec: Arc<OpChainSpec>,
    /// What the L1-info deposit sets.
    l1: L1Params,
    /// The balance of [`SIGNER`].
    balance: U256,
    /// The operator's limits, shared between the builder and the pool.
    builder_config: OpBuilderConfig,
    /// The header gas limit.
    block_gas_limit: u64,
    /// The block being built on.
    parent: SealedHeader,
    /// The L1-info deposit that opens the block.
    deposit: OpTransactionSigned,
}

impl Env {
    /// Creates an environment with these values, on a fresh chain.
    fn new(
        fork: Fork,
        l1: L1Params,
        balance: U256,
        block_gas_limit: u64,
        builder_config: OpBuilderConfig,
    ) -> Self {
        Self {
            chain_spec: fork.chain_spec(),
            l1,
            balance,
            builder_config,
            block_gas_limit,
            parent: parent_header(block_gas_limit),
            deposit: l1.deposit(),
        }
    }

    /// The payload builder's context, with the L1-info deposit in its attributes.
    fn payload_ctx(
        &self,
    ) -> OpPayloadBuilderCtx<
        OpEvmConfig<OpChainSpec, OpPrimitives>,
        OpChainSpec,
        OpPayloadBuilderAttributes<OpTransactionSigned>,
    > {
        let attributes = OpPayloadBuilderAttributes {
            timestamp: 2,
            gas_limit: Some(self.block_gas_limit),
            transactions: vec![WithEncoded::new(
                self.deposit.encoded_2718().into(),
                self.deposit.clone(),
            )],
            eip_1559_params: Some(B64::ZERO),
            min_base_fee: Some(0),
            parent_beacon_block_root: Some(B256::ZERO),
            ..Default::default()
        };
        OpPayloadBuilderCtx {
            evm_config: OpEvmConfig::optimism(self.chain_spec.clone()),
            builder_config: self.builder_config.clone(),
            chain_spec: self.chain_spec.clone(),
            config: PayloadConfig {
                parent_header: Arc::new(self.parent.clone()),
                parent_block_info: None,
                payload_id: attributes.id,
                attributes,
            },
            cancel: Default::default(),
            best_payload: None,
        }
    }

    /// The state the builder executes on: the sender's account and the `L1Block` contract.
    fn state_provider(&self) -> StateProviderTest {
        let mut state_provider = StateProviderTest::default();
        state_provider.insert_account(
            SIGNER,
            Account { balance: self.balance, nonce: STATE_NONCE, ..Default::default() },
            None,
            Default::default(),
        );
        // `L1Block` needs code: the deposit touches it, and EIP-161 would clear an empty account
        // along with its storage. A lone `STOP` is enough, as the values are already in storage.
        state_provider.insert_account(
            L1_BLOCK_CONTRACT,
            Account::default(),
            Some(Bytes::from_static(&[0x00])),
            self.l1.storage(),
        );
        state_provider
    }

    /// A real pool and validator that have seen the parent, with the sender's account in state.
    fn pool(&self) -> impl TransactionPool<Transaction = OpPooledTransaction> {
        let client = MockEthProvider::<OpPrimitives, _>::new()
            .with_chain_spec(self.chain_spec.clone())
            .with_genesis_block();
        client.add_account(SIGNER, ExtendedAccount::new(STATE_NONCE, self.balance));
        let validator = OpTransactionValidator::new(
            EthTransactionValidatorBuilder::new(
                client,
                OpEvmConfig::optimism(self.chain_spec.clone()),
            )
            .set_block_gas_limit(self.block_gas_limit)
            .build(InMemoryBlobStore::default()),
        )
        .with_gas_limit_config(self.builder_config.gas_limit_config.clone());
        validator.update_l1_block_info(self.parent.header(), Some(&self.deposit));
        let pool = Pool::new(
            validator,
            CoinbaseTipOrdering::default(),
            InMemoryBlobStore::default(),
            PoolConfig::default(),
        );
        pool.set_block_info(BlockInfo {
            block_gas_limit: self.block_gas_limit,
            last_seen_block_hash: self.parent.hash(),
            last_seen_block_number: self.parent.number,
            pending_basefee: BASE_FEE,
            pending_blob_fee: None,
        });
        pool
    }
}

/// Runs a case through the pool and, if it reaches pending, the builder.
fn run(case: &Case) -> Report {
    let tx = case.tx();
    let env = case.env(&tx);

    // Open the block, up to and including the L1-info deposit.
    let ctx = env.payload_ctx();
    let state_provider = env.state_provider();
    let mut db = State::builder()
        .with_database(StateProviderDatabase::new((&state_provider).into_evm_state_provider()))
        .with_bundle_update()
        .build();
    db.load_cache_account(L1_BLOCK_CONTRACT).expect("L1Block loads");
    let mut builder = ctx.block_builder(&mut db).expect("block builder can be created");
    builder.apply_pre_execution_changes().expect("pre-execution changes apply");
    let mut info =
        ctx.execute_sequencer_transactions(&mut builder, None).expect("L1-info deposit executes");
    assert_eq!(
        builder.evm_mut().block().basefee(),
        BASE_FEE,
        "the pool must see the builder's base fee"
    );
    assert_eq!(
        L1BlockInfo::fetch_da_footprint_gas_scalar(builder.evm_mut().db_mut()).unwrap(),
        case.l1.da_footprint_gas_scalar,
        "the executor must see the same scalar as the validator"
    );

    // Offer the transaction to the pool. Rejection is one of the outcomes under test, and shows up
    // as an empty pool.
    let pool = env.pool();
    let hash = *tx.hash();
    let _ = tokio::runtime::Builder::new_current_thread()
        .build()
        .expect("runtime builds")
        .block_on(pool.add_transaction(TransactionOrigin::External, tx.clone()));
    let reached_pending = pool.pending_transactions().iter().any(|pending| *pending.hash() == hash);

    // Build, if the pool says it's ready.
    let mut committed = Vec::new();
    if reached_pending {
        let cancelled = ctx
            .execute_best_transactions(
                &mut info,
                &mut builder,
                RethPayloadTransactions(PayloadTransactionsFixed::new(vec![tx.clone()])),
                None,
                Some(&mut committed),
            )
            .expect("the block builds");
        assert!(cancelled.is_none(), "the build isn't cancelled");
    }

    Report {
        footprint: case.footprint(&tx),
        gas: tx.gas_limit(),
        block_gas_limit: env.block_gas_limit,
        operator_gas_limit: env.builder_config.gas_limit_config.gas_limit(),
        reached_pending,
        pool_size: pool.pool_size().total,
        included: !committed.is_empty(),
        throttled: case.throttled(&tx, &env.deposit),
    }
}

/// A fixed seed by default, so that a PR only fails because of the PR.
fn config() -> ProptestConfig {
    let mut config = ProptestConfig::default();
    if matches!(config.rng_seed, RngSeed::Random) {
        config.rng_seed = RngSeed::Fixed(DEFAULT_SEED);
    }
    config
}

/// L1 info, mostly with the default scalar, where the footprint limit is reachable.
fn l1_params() -> impl Strategy<Value = L1Params> {
    (
        (0..=1_000_000_000_000u64, 0..=1_000_000u32, 0..=1_000_000_000u64, 0..=1_000_000u32),
        (0..=100_000u32, 0..=1_000_000u64),
        prop_oneof![
            8 => Just(400u16),
            1 => Just(0u16),
            1 => any::<u16>(),
        ],
    )
        .prop_map(
            |(
                (l1_base_fee, base_fee_scalar, l1_blob_base_fee, blob_base_fee_scalar),
                (operator_fee_scalar, operator_fee_constant),
                da_footprint_gas_scalar,
            )| L1Params {
                l1_base_fee,
                base_fee_scalar,
                l1_blob_base_fee,
                blob_base_fee_scalar,
                operator_fee_scalar,
                operator_fee_constant,
                da_footprint_gas_scalar,
            },
        )
}

/// Either fork, as chains run both.
fn fork() -> impl Strategy<Value = Fork> {
    prop_oneof![Just(Fork::Jovian), Just(Fork::Karst)]
}

/// A limit on or near a boundary of the transaction, or unrelated to it.
fn limit() -> impl Strategy<Value = Limit> {
    prop_oneof![
        3 => (-2i64..=2).prop_map(Limit::Footprint),
        3 => (-2i64..=2).prop_map(Limit::Gas),
        2 => (0u64..=1_000_000).prop_map(Limit::Above),
        2 => (MIN_BLOCK_GAS_LIMIT..=60_000_000).prop_map(Limit::Fixed),
    ]
}

/// A limit the transaction fits under, often with nothing to spare.
fn above_limit() -> impl Strategy<Value = Limit> {
    prop_oneof![Just(Limit::Above(0)), (0u64..=1_000_000).prop_map(Limit::Above),]
}

/// Anything goes: the transaction may break any number of rules, or none.
fn any_case() -> impl Strategy<Value = Case> {
    (
        fork(),
        (0..=MAX_ADMISSIBLE_CALLDATA + 8_000, any::<u64>(), 0..=100u8),
        (
            prop_oneof![
                1 => Just(0u64),
                8 => 0..=2_000_000u64,
                // Around the EIP-7825 cap of 16.78M.
                1 => 12_000_000..=18_000_000u64,
            ],
            -2i64..=1_000,
            any::<bool>(),
        ),
        prop_oneof![
            3 => (-2i64..=2).prop_map(Balance::Cost),
            2 => Just(Balance::Plenty),
        ],
        l1_params(),
        limit(),
        prop::option::weighted(0.5, limit()),
        (
            prop::option::weighted(0.1, 1..=200_000u64),
            prop::option::weighted(0.1, 1..=200_000u64),
            prop::option::weighted(0.1, 1..=200_000u64),
        ),
    )
        .prop_map(
            |(
                fork,
                (calldata_len, calldata_seed, calldata_zero_pct),
                (extra_gas, fee_offset, nonce_gap),
                balance,
                l1,
                block_gas_limit,
                operator_gas_limit,
                (da_tx_limit, da_block_limit, max_uncompressed_block_size),
            )| Case {
                fork,
                calldata_len,
                calldata_seed,
                calldata_zero_pct,
                extra_gas,
                fee_offset,
                nonce_gap,
                balance,
                l1,
                block_gas_limit,
                operator_gas_limit,
                da_tx_limit,
                da_block_limit,
                max_uncompressed_block_size,
            },
        )
}

/// A transaction that breaks no rule, placed on as many boundaries as that allows.
fn valid_case() -> impl Strategy<Value = Case> {
    (
        fork(),
        (0..=MAX_ADMISSIBLE_CALLDATA - 2_000, any::<u64>(), 0..=100u8),
        (prop_oneof![Just(0u64), 0..=2_000_000u64], 0i64..=1_000),
        prop_oneof![
            3 => (0i64..=2).prop_map(Balance::Cost),
            1 => Just(Balance::Plenty),
        ],
        l1_params(),
        above_limit(),
        prop::option::weighted(0.5, above_limit()),
    )
        .prop_map(
            |(
                fork,
                (calldata_len, calldata_seed, calldata_zero_pct),
                (extra_gas, fee_offset),
                balance,
                l1,
                block_gas_limit,
                operator_gas_limit,
            )| Case {
                fork,
                calldata_len,
                calldata_seed,
                calldata_zero_pct,
                extra_gas,
                fee_offset,
                nonce_gap: false,
                balance,
                l1,
                block_gas_limit,
                operator_gas_limit,
                da_tx_limit: None,
                da_block_limit: None,
                max_uncompressed_block_size: None,
            },
        )
}

proptest! {
    #![proptest_config(config())]

    /// The validator isn't too strict: a transaction that breaks no rule is admitted, and built.
    ///
    /// This also keeps [`pending_transactions_are_included`] honest, as that property would pass
    /// if the pool simply rejected everything.
    #[test]
    fn valid_transactions_are_admitted_and_included(case in valid_case()) {
        let report = run(&case);
        prop_assert!(report.reached_pending, "did not reach pending: {report:?}");
        prop_assert!(report.included, "not included: {report:?}");
    }

    /// The core property: if the pool says a transaction is pending, an otherwise empty block
    /// includes it.
    ///
    /// `throttled` is the written list of known exceptions: batcher throttling and the uncompressed
    /// size limit aren't enforced at admission.
    #[test]
    fn pending_transactions_are_included(case in any_case()) {
        let report = run(&case);
        if report.reached_pending {
            prop_assert_eq!(report.included, !report.throttled, "{:?}", report);
        }
    }

    /// A transaction above the block limits isn't admitted to any subpool.
    #[test]
    fn transactions_that_cannot_fit_enter_no_subpool(case in any_case()) {
        let report = run(&case);
        if report.exceeds_effective_limit() {
            prop_assert_eq!(report.pool_size, 0, "{:?}", report);
        }
    }
}
