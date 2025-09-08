use crate::{ExecutionPayloadBaseV1, FlashBlock};
use alloy_eips::eip2718::WithEncoded;
use eyre::{bail, OptionExt};
use reth_primitives_traits::{Recovered, SignedTransaction};
use std::collections::BTreeMap;
use tracing::trace;

/// An ordered B-tree keeping the track of a sequence of [`FlashBlock`]s by their indices.
#[derive(Debug)]
pub(crate) struct FlashBlockPendingSequence<T> {
    /// tracks the individual flashblocks in order
    ///
    /// With a blocktime of 2s and flashblock tick-rate of 200ms plus one extra flashblock per new
    /// pending block, we expect 11 flashblocks per slot.
    inner: BTreeMap<u64, PreparedFlashBlock<T>>,
}

impl<T> FlashBlockPendingSequence<T>
where
    T: SignedTransaction,
{
    pub(crate) const fn new() -> Self {
        Self { inner: BTreeMap::new() }
    }

    /// Inserts a new block into the sequence.
    ///
    /// A [`FlashBlock`] with index 0 resets the set.
    pub(crate) fn insert(&mut self, flashblock: FlashBlock) -> eyre::Result<()> {
        if flashblock.index == 0 {
            trace!(number=%flashblock.block_number(), "Tracking new flashblock sequence");
            // Flash block at index zero resets the whole state
            self.clear();
            self.inner.insert(flashblock.index, PreparedFlashBlock::new(flashblock)?);
            return Ok(())
        }

        // only insert if we previously received the same block, assume we received index 0
        if self.block_number() == Some(flashblock.metadata.block_number) {
            trace!(number=%flashblock.block_number(), index = %flashblock.index, block_count = self.inner.len()  ,"Received followup flashblock");
            self.inner.insert(flashblock.index, PreparedFlashBlock::new(flashblock)?);
        } else {
            trace!(number=%flashblock.block_number(), index = %flashblock.index, current=?self.block_number()  ,"Ignoring untracked flashblock following");
        }

        Ok(())
    }

    /// Iterator over sequence of executable transactions.
    ///
    /// A flashblocks is not ready if there's missing previous flashblocks, i.e. there's a gap in
    /// the sequence
    ///
    /// Note: flashblocks start at `index 0`.
    pub(crate) fn ready_transactions(
        &self,
    ) -> impl Iterator<Item = WithEncoded<Recovered<T>>> + '_ {
        self.inner
            .values()
            .enumerate()
            .take_while(|(idx, block)| {
                // flashblock index 0 is the first flashblock
                block.block().index == *idx as u64
            })
            .flat_map(|(_, block)| block.txs.clone())
    }

    fn clear(&mut self) {
        self.inner.clear();
    }

    /// Returns the first block number
    pub(crate) fn block_number(&self) -> Option<u64> {
        Some(self.inner.values().next()?.block().metadata.block_number)
    }

    /// Returns the payload base of the first tracked flashblock.
    pub(crate) fn payload_base(&self) -> Option<ExecutionPayloadBaseV1> {
        self.inner.values().next()?.block().base.clone()
    }

    /// Returns the number of tracked flashblocks.
    pub(crate) fn count(&self) -> usize {
        self.inner.len()
    }
}

/// A complete sequence of flashblocks, often corresponding to a full block.
/// Ensure invariants of a complete flashblocks sequence.
#[derive(Debug)]
pub struct FlashBlockCompleteSequence(Vec<FlashBlock>);

impl FlashBlockCompleteSequence {
    /// Create a complete sequence from a vector of flashblocks.
    /// Ensure that:
    /// * vector is not empty
    /// * first flashblock have the base payload
    /// * sequence of flashblocks is sound (successive index from 0, same payload id, ...)
    pub fn new(blocks: Vec<FlashBlock>) -> eyre::Result<Self> {
        let first_block = blocks.first().ok_or_eyre("No flashblocks in sequence")?;

        // Ensure that first flashblock have base
        first_block.base.as_ref().ok_or_eyre("Flashblock at index 0 has no base")?;

        // Ensure that index are successive from 0, have same block number and payload id
        if !blocks.iter().enumerate().all(|(idx, block)| {
            idx == block.index as usize &&
                block.payload_id == first_block.payload_id &&
                block.metadata.block_number == first_block.metadata.block_number
        }) {
            bail!("Flashblock inconsistencies detected in sequence");
        }

        Ok(Self(blocks))
    }

    /// Returns the block number
    pub fn block_number(&self) -> u64 {
        self.0.first().unwrap().metadata.block_number
    }

    /// Returns the payload base of the first flashblock.
    pub fn payload_base(&self) -> &ExecutionPayloadBaseV1 {
        self.0.first().unwrap().base.as_ref().unwrap()
    }

    /// Returns the number of flashblocks in the sequence.
    pub const fn count(&self) -> usize {
        self.0.len()
    }
}

impl<T> TryFrom<FlashBlockPendingSequence<T>> for FlashBlockCompleteSequence {
    type Error = eyre::Error;
    fn try_from(sequence: FlashBlockPendingSequence<T>) -> Result<Self, Self::Error> {
        Self::new(
            sequence.inner.into_values().map(|block| block.block().clone()).collect::<Vec<_>>(),
        )
    }
}

#[derive(Debug)]
struct PreparedFlashBlock<T> {
    /// The prepared transactions, ready for execution
    txs: Vec<WithEncoded<Recovered<T>>>,
    /// The tracked flashblock
    block: FlashBlock,
}

impl<T> PreparedFlashBlock<T> {
    const fn block(&self) -> &FlashBlock {
        &self.block
    }
}

impl<T> PreparedFlashBlock<T>
where
    T: SignedTransaction,
{
    /// Creates a flashblock that is ready for execution by preparing all transactions
    ///
    /// Returns an error if decoding or signer recovery fails.
    fn new(block: FlashBlock) -> eyre::Result<Self> {
        let mut txs = Vec::with_capacity(block.diff.transactions.len());
        for encoded in block.diff.transactions.iter().cloned() {
            let tx = T::decode_2718_exact(encoded.as_ref())?;
            let signer = tx.try_recover()?;
            let tx = WithEncoded::new(encoded, tx.with_signer(signer));
            txs.push(tx);
        }

        Ok(Self { txs, block })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ExecutionPayloadFlashblockDeltaV1;
    use alloy_consensus::{
        transaction::SignerRecoverable, EthereumTxEnvelope, EthereumTypedTransaction, TxEip1559,
    };
    use alloy_eips::Encodable2718;
    use alloy_primitives::{hex, Signature, TxKind, U256};

    #[test]
    fn test_sequence_stops_before_gap() {
        let mut sequence = FlashBlockPendingSequence::new();
        let tx = EthereumTxEnvelope::new_unhashed(
            EthereumTypedTransaction::<TxEip1559>::Eip1559(TxEip1559 {
                chain_id: 4,
                nonce: 26u64,
                max_priority_fee_per_gas: 1500000000,
                max_fee_per_gas: 1500000013,
                gas_limit: 21_000u64,
                to: TxKind::Call(hex!("61815774383099e24810ab832a5b2a5425c154d5").into()),
                value: U256::from(3000000000000000000u64),
                input: Default::default(),
                access_list: Default::default(),
            }),
            Signature::new(
                U256::from_be_bytes(hex!(
                    "59e6b67f48fb32e7e570dfb11e042b5ad2e55e3ce3ce9cd989c7e06e07feeafd"
                )),
                U256::from_be_bytes(hex!(
                    "016b83f4f980694ed2eee4d10667242b1f40dc406901b34125b008d334d47469"
                )),
                true,
            ),
        );
        let tx = Recovered::new_unchecked(tx.clone(), tx.recover_signer_unchecked().unwrap());

        sequence
            .insert(FlashBlock {
                payload_id: Default::default(),
                index: 0,
                base: None,
                diff: ExecutionPayloadFlashblockDeltaV1 {
                    transactions: vec![tx.encoded_2718().into()],
                    ..Default::default()
                },
                metadata: Default::default(),
            })
            .unwrap();

        sequence
            .insert(FlashBlock {
                payload_id: Default::default(),
                index: 2,
                base: None,
                diff: Default::default(),
                metadata: Default::default(),
            })
            .unwrap();

        let actual_txs: Vec<_> = sequence.ready_transactions().collect();
        let expected_txs = vec![WithEncoded::new(tx.encoded_2718().into(), tx)];

        assert_eq!(actual_txs, expected_txs);
    }
}
