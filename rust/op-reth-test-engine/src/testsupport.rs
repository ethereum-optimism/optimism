//! Shared test helpers for driving the engine over its public API: an ephemeral OP chain with
//! Karst active at genesis, plus builders for payload attributes, deposits, and user transactions.

use std::{collections::BTreeMap, sync::Arc};

use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_eips::eip2718::Encodable2718;
use alloy_genesis::{ChainConfig, Genesis, GenesisAccount};
use alloy_network::TxSignerSync;
use alloy_primitives::{Address, B64, B256, Bytes, TxKind, U256, b256};
use alloy_rpc_types_engine::{ForkchoiceState, PayloadAttributes};
use alloy_signer_local::PrivateKeySigner;
use op_alloy_consensus::{TxDeposit, encode_jovian_extra_data};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use op_revm::constants::{
    ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT, L1_BASE_FEE_SLOT, L1_BLOCK_CONTRACT,
};
use reth_chainspec::{Chain, EthereumHardfork, ForkCondition};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_primitives::OpTransactionSigned;

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
const FUNDED_BALANCE: U256 = U256::from_limbs([1_000_000_000_000_000_000, 0, 0, 0]);

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

/// An engine over an ephemeral OP Mainnet chain with Karst active at genesis, the `L1Block`
/// predeploy seeded, and `funded` given a spendable balance.
pub(crate) fn test_engine(funded: Address) -> TestEngine {
    test_engine_with_accounts(funded, [])
}

/// [`test_engine`] with additional genesis accounts (e.g. pre-deployed contract code for
/// `eth_call` tests).
pub(crate) fn test_engine_with_accounts(
    funded: Address,
    extra: impl IntoIterator<Item = (Address, GenesisAccount)>,
) -> TestEngine {
    let mut alloc = BTreeMap::from([
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
    alloc.extend(extra);

    let genesis = Genesis {
        config: ChainConfig { chain_id: CHAIN_ID, ..Default::default() },
        gas_limit: GAS_LIMIT,
        base_fee_per_gas: Some(1_000_000_000),
        excess_blob_gas: Some(0),
        blob_gas_used: Some(0),
        extra_data: encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism_canyon(), 1)
            .expect("encode extra data"),
        alloc,
        ..Default::default()
    };

    // `karst_activated()` activates the OP hardforks but not the Ethereum forks they ride; Isthmus
    // rides Prague, which the builder omits, so add it for a consistent spec.
    // See ethereum-optimism/optimism#21239.
    let spec = OpChainSpecBuilder::optimism_mainnet()
        .chain(Chain::from_id(CHAIN_ID))
        .karst_activated()
        .with_fork(EthereumHardfork::Prague, ForkCondition::Timestamp(0))
        .genesis(genesis)
        .build();
    let chain = EphemeralChain::from_chain_spec(Arc::new(spec)).expect("build ephemeral chain");
    TestEngine::from_chain(chain)
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

/// Payload attributes building on the given timestamp with the given forced deposit transactions.
/// Sets the Holocene/Jovian EIP-1559 params and min base fee so block assembly succeeds under
/// Karst.
pub(crate) fn payload_attrs(
    timestamp: u64,
    deposits: Vec<Bytes>,
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
        transactions: Some(deposits),
        no_tx_pool: Some(no_tx_pool),
        gas_limit: Some(GAS_LIMIT),
        eip_1559_params: Some(B64::ZERO),
        min_base_fee: Some(0),
    }
}

/// A forkchoice state pointing head/safe/finalized at `head`.
pub(crate) fn fcu(head: B256) -> ForkchoiceState {
    ForkchoiceState { head_block_hash: head, safe_block_hash: head, finalized_block_hash: head }
}
