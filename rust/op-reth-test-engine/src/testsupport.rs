//! Shared test helpers for driving the engine over its public API: an ephemeral OP chain with
//! Karst active at genesis, plus builders for payload attributes, deposits, and user transactions.

use std::{collections::BTreeMap, sync::Arc};

use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_eips::eip2718::Encodable2718;
use alloy_genesis::{ChainConfig, Genesis, GenesisAccount};
use alloy_network::TxSignerSync;
use alloy_primitives::{Address, B64, B256, Bytes, TxKind, U256, b256};
use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated, PayloadAttributes};
use alloy_signer_local::PrivateKeySigner;
use op_alloy_consensus::{TxDeposit, encode_holocene_extra_data, encode_jovian_extra_data};
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayload, OpPayloadAttributes};
use op_revm::constants::{
    ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT, L1_BASE_FEE_SLOT, L1_BLOCK_CONTRACT,
};
use reth_chainspec::{Chain, EthereumHardfork, ForkCondition};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder, OpHardforks};
use reth_optimism_evm::OpNextBlockEnvAttributes;
use reth_optimism_primitives::{OpBlock, OpTransactionSigned};
use reth_payload_primitives::EngineApiMessageVersion;

use crate::{EphemeralChain, TestEngine};
use alloy_eips::eip1559::BaseFeeParams;

/// A synthetic OP chain id. Deliberately not OP Mainnet (10): op-reth pins OP Mainnet's genesis
/// hash to the registry value (which `hash_slow` cannot reproduce), so a chain built from a
/// synthetic genesis under id 10 would be indexed under the wrong genesis hash.
pub(crate) const CHAIN_ID: u64 = 901;
pub(crate) const GAS_LIMIT: u64 = 30_000_000;
/// A user-tx gas price the funded sender can always cover.
const MAX_FEE: u128 = 10_000_000_000; // 10 gwei
/// Sender balance — 1 ETH, enough for many user txs, without the overflow risk of `U256::MAX`.
pub(crate) const FUNDED_BALANCE: U256 = U256::from_limbs([1_000_000_000_000_000_000, 0, 0, 0]);

fn slot(slot: U256) -> B256 {
    B256::from(slot.to_be_bytes::<32>())
}

fn value(v: u64) -> B256 {
    B256::from(U256::from(v).to_be_bytes::<32>())
}

/// The `L1Block` predeploy storage that drives the L1-cost function, keyed by the slots op-revm
/// reads (operator-fee and DA-footprint scalars default to zero).
fn l1_block_storage() -> BTreeMap<B256, B256> {
    BTreeMap::from([
        (slot(L1_BASE_FEE_SLOT), value(1_000_000_000)),
        (slot(ECOTONE_L1_BLOB_BASE_FEE_SLOT), value(1)),
        (
            slot(ECOTONE_L1_FEE_SCALARS_SLOT),
            b256!("0x0000000000000000000000000000000000001db0000d27300000000000000005"),
        ),
    ])
}

/// An engine over [`test_chain`].
pub(crate) fn test_engine(funded: Address) -> TestEngine {
    TestEngine::from_chain(test_chain(funded))
}

/// An ephemeral OP Mainnet chain with Karst active at genesis, the `L1Block` predeploy seeded, and
/// `funded` given a spendable balance.
pub(crate) fn test_chain(funded: Address) -> EphemeralChain {
    let extra_data = encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism_canyon(), 1)
        .expect("encode extra data");
    chain_with_forks(funded, OpChainSpecBuilder::optimism_mainnet().karst_activated(), extra_data)
}

/// An engine like [`test_engine`] but with Isthmus as the newest active fork, so Jovian (and its
/// minimum base fee) is not yet in force.
pub(crate) fn isthmus_test_engine(funded: Address) -> TestEngine {
    let extra_data = encode_holocene_extra_data(B64::ZERO, BaseFeeParams::optimism_canyon())
        .expect("encode extra data");
    let forks = OpChainSpecBuilder::optimism_mainnet().isthmus_activated();
    TestEngine::from_chain(chain_with_forks(funded, forks, extra_data))
}

/// A chain under the hardforks `forks` activates, with a genesis carrying `extra_data` (whose
/// encoding the forks active at genesis dictate).
fn chain_with_forks(
    funded: Address,
    forks: OpChainSpecBuilder,
    extra_data: Bytes,
) -> EphemeralChain {
    let alloc = BTreeMap::from([
        (
            L1_BLOCK_CONTRACT,
            GenesisAccount {
                nonce: Some(1),
                storage: Some(l1_block_storage()),
                ..Default::default()
            },
        ),
        (funded, GenesisAccount { balance: FUNDED_BALANCE, ..Default::default() }),
    ]);

    let genesis = Genesis {
        config: ChainConfig { chain_id: CHAIN_ID, ..Default::default() },
        gas_limit: GAS_LIMIT,
        base_fee_per_gas: Some(1_000_000_000),
        excess_blob_gas: Some(0),
        blob_gas_used: Some(0),
        extra_data,
        alloc,
        ..Default::default()
    };

    // The fork helpers activate the OP hardforks but not the Ethereum forks they ride; Isthmus
    // rides Prague, which they omit, so add it for a consistent spec.
    // See ethereum-optimism/optimism#21239.
    let spec = forks
        .chain(Chain::from_id(CHAIN_ID))
        .with_fork(EthereumHardfork::Prague, ForkCondition::Timestamp(0))
        .genesis(genesis)
        .build();
    EphemeralChain::from_chain_spec(Arc::new(spec)).expect("build ephemeral chain")
}

/// The environment of a block at `timestamp` on top of a parent with `parent_extra_data`, carrying
/// the parent's EIP-1559 parameters forward.
pub(crate) fn next_env(timestamp: u64, parent_extra_data: Bytes) -> OpNextBlockEnvAttributes {
    OpNextBlockEnvAttributes {
        timestamp,
        suggested_fee_recipient: Address::ZERO,
        prev_randao: B256::ZERO,
        gas_limit: GAS_LIMIT,
        parent_beacon_block_root: Some(B256::ZERO),
        extra_data: parent_extra_data,
    }
}

/// The execution data `engine_newPayload` carries for `block`, with the block hash recomputed from
/// the (possibly tampered) header.
pub(crate) fn payload_of(block: &OpBlock) -> OpExecutionData {
    let (payload, sidecar) = OpExecutionPayload::from_block_slow(block);
    OpExecutionData::new(payload, sidecar)
}

/// A fixed secp256k1 key so user transactions across nonces share one (fundable) sender. A real
/// signature is required — a constant signature would recover a different sender per transaction.
const SIGNER_KEY: B256 =
    b256!("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef");

fn signer() -> PrivateKeySigner {
    PrivateKeySigner::from_bytes(&SIGNER_KEY).expect("valid signer key")
}

/// An EIP-1559 user transaction with an explicit gas limit, signed by [`signer`].
pub(crate) fn user_tx_with_gas(nonce: u64, gas_limit: u64) -> OpTransactionSigned {
    let mut tx = TxEip1559 {
        chain_id: CHAIN_ID,
        nonce,
        gas_limit,
        max_fee_per_gas: MAX_FEE,
        max_priority_fee_per_gas: 0,
        to: TxKind::Call(Address::ZERO),
        ..Default::default()
    };
    let signature = signer().sign_transaction_sync(&mut tx).expect("sign tx");
    tx.into_signed(signature).into()
}

/// An EIP-1559 value transfer (21000 gas) at `nonce`, signed by [`signer`].
pub(crate) fn user_tx(nonce: u64) -> OpTransactionSigned {
    user_tx_with_gas(nonce, 21_000)
}

/// The sender address of [`user_tx`]/[`user_tx_with_gas`] — fund this in genesis.
pub(crate) fn user_sender() -> Address {
    signer().address()
}

/// A distinct depositor so a deposit's nonce bump doesn't collide with the user tx.
pub(crate) fn depositor() -> Address {
    Address::with_last_byte(0xde)
}

/// A bare deposit (type 0x7E) from `from`.
pub(crate) fn deposit_tx(from: Address) -> OpTransactionSigned {
    TxDeposit { from, to: TxKind::Call(from), gas_limit: 21_000, ..Default::default() }.into()
}

/// The EIP-2718 encoding of a transaction (the form carried in payload attributes / accepted by
/// `include_tx`).
pub(crate) fn encode(tx: &OpTransactionSigned) -> Bytes {
    Bytes::from(tx.encoded_2718())
}

/// Payload attributes building on the given timestamp with the given forced transactions (deposits,
/// then on a derived block the batch's transactions). Sets the Holocene/Jovian EIP-1559 params and
/// min base fee so block assembly succeeds under Karst.
pub(crate) fn payload_attrs(
    timestamp: u64,
    forced_txs: Vec<Bytes>,
    no_tx_pool: bool,
) -> OpPayloadAttributes {
    OpPayloadAttributes {
        payload_attributes: PayloadAttributes {
            timestamp,
            prev_randao: B256::ZERO,
            suggested_fee_recipient: Address::ZERO,
            withdrawals: Some(vec![]),
            parent_beacon_block_root: Some(B256::ZERO),
            slot_number: None,
            target_gas_limit: None,
        },
        transactions: Some(forced_txs),
        no_tx_pool: Some(no_tx_pool),
        gas_limit: Some(GAS_LIMIT),
        eip_1559_params: Some(B64::ZERO),
        min_base_fee: Some(0),
    }
}

/// The `engine_forkchoiceUpdated` message version a consensus client calls with attributes at
/// `timestamp` on `chain_spec`: V3 from Ecotone, V2 from Canyon, V1 before. Mirrors op-node's
/// `ForkchoiceUpdatedVersion` in `op-node/rollup/types.go`.
pub(crate) fn fcu_version(chain_spec: &OpChainSpec, timestamp: u64) -> EngineApiMessageVersion {
    if chain_spec.is_ecotone_active_at_timestamp(timestamp) {
        EngineApiMessageVersion::V3
    } else if chain_spec.is_canyon_active_at_timestamp(timestamp) {
        EngineApiMessageVersion::V2
    } else {
        EngineApiMessageVersion::V1
    }
}

impl TestEngine {
    /// [`TestEngine::forkchoice_updated`] through the method version op-node calls: the
    /// [`fcu_version`] for the attributes' timestamp, and V3 without attributes.
    pub(crate) fn forkchoice_updated_as_op_node(
        &mut self,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> crate::Result<ForkchoiceUpdated> {
        let version = attributes.as_ref().map_or(EngineApiMessageVersion::V3, |attributes| {
            fcu_version(&self.chain.chain_spec(), attributes.payload_attributes.timestamp)
        });
        self.forkchoice_updated(version, state, attributes)
    }
}

/// A forkchoice state pointing head/safe/finalized at `head`.
pub(crate) fn fcu(head: B256) -> ForkchoiceState {
    ForkchoiceState { head_block_hash: head, safe_block_hash: head, finalized_block_hash: head }
}
