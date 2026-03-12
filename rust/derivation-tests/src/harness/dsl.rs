//! Scenario-oriented DSL types for derivation tests.
//!
//! These types provide a high-level API for constructing test scenarios.
//! See [`DerivationTest`](super::DerivationTest) for the DSL methods.

use alloy_primitives::{Address, U256};
use op_alloy_consensus::OpTxEnvelope;

use crate::l2::L2BlockRef;

use super::DerivationTest;

/// How the batch is encoded.
#[derive(Debug, Clone, Copy, Default)]
pub enum BatchEncoding {
    /// One `SingleBatch` per L2 block.
    Singular,
    /// Multiple L2 blocks in a single `SpanBatch`.
    #[default]
    SpanBatch,
}

/// How the batch is submitted to L1.
#[derive(Debug, Clone, Copy, Default)]
pub enum BatchSubmissionType {
    /// Batch data in calldata (EIP-1559 transaction).
    Calldata,
    /// Batch data in a blob (EIP-4844 transaction).
    #[default]
    Blobs,
}

/// Configuration for batch submission. Defaults to span batch via blobs.
#[derive(Debug, Clone, Copy, Default)]
pub struct BatchConfig {
    /// How the batch is encoded.
    pub encoding: BatchEncoding,
    /// How the batch is submitted to L1.
    pub submission: BatchSubmissionType,
}

impl BatchConfig {
    /// Singular batch via calldata (the simplest format).
    pub const fn singular_calldata() -> Self {
        Self { encoding: BatchEncoding::Singular, submission: BatchSubmissionType::Calldata }
    }

    /// Span batch via calldata.
    pub const fn span_calldata() -> Self {
        Self { encoding: BatchEncoding::SpanBatch, submission: BatchSubmissionType::Calldata }
    }
}

/// Builder for L2 blocks with user transactions.
///
/// Created by [`DerivationTest::derive_l2_block`]. Collects transactions and
/// builds the block on [`build`](Self::build).
#[allow(missing_debug_implementations)]
pub struct BlockBuilder<'a> {
    test: &'a mut DerivationTest,
    user_txs: Vec<OpTxEnvelope>,
}

impl<'a> BlockBuilder<'a> {
    pub(crate) const fn new(test: &'a mut DerivationTest) -> Self {
        Self { test, user_txs: Vec::new() }
    }

    /// Add a pre-signed transaction.
    pub fn with_tx(mut self, tx: OpTxEnvelope) -> Self {
        self.user_txs.push(tx);
        self
    }

    /// Add a simple ETH transfer from the prefunded test account.
    ///
    /// Signs an EIP-1559 transaction with `gas_limit=21000`, `max_fee=1`, `priority_fee=0`.
    /// Nonce is auto-tracked across calls within the same [`DerivationTest`].
    pub fn with_funded_transfer(mut self, to: Address, value: U256) -> Self {
        use alloy_consensus::{SignableTransaction, TxEip1559};
        use alloy_signer::SignerSync;
        use alloy_signer_local::PrivateKeySigner;

        let signer = PrivateKeySigner::from_bytes(&crate::config::PREFUNDED_ACCOUNT_KEY)
            .expect("valid prefunded key");

        let nonce = self.test.prefunded_nonce;
        self.test.prefunded_nonce += 1;

        let tx = TxEip1559 {
            chain_id: self.test.config.l2_chain_id,
            nonce,
            gas_limit: 21_000,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 0,
            to: alloy_primitives::TxKind::Call(to),
            value,
            ..Default::default()
        };

        let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing works");
        let signed = tx.into_signed(sig);
        let eth_envelope = alloy_consensus::TxEnvelope::Eip1559(signed);
        let op_tx =
            OpTxEnvelope::try_from_eth_envelope(eth_envelope).expect("convert to OP envelope");

        self.user_txs.push(op_tx);
        self
    }

    /// Build the L2 block with collected transactions.
    ///
    /// Consumes the builder, releasing the mutable borrow on [`DerivationTest`].
    pub fn build(self) -> L2BlockRef {
        let block_ref = self.test.l2.build_block(self.user_txs).expect("failed to build L2 block");
        self.test.pending_l2_blocks.push(block_ref);
        block_ref
    }
}
