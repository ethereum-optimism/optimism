//! Additional support for estimating the data availability size of transactions.

/// Helper trait that allows attaching an estimated data availability size.
pub trait DataAvailabilitySized {
    /// Get the estimated data availability size of the transaction.
    ///
    /// Note: it is expected that this value will be cached internally.
    fn estimated_da_size(&self) -> u64;
}

#[cfg(test)]
mod tests {
    use super::DataAvailabilitySized;
    use crate::OpPooledTransaction;
    use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, Bytes, Signature, TxKind};
    use proptest::prelude::*;
    use reth_optimism_primitives::OpTransactionSigned;

    /// `FastLZ` only looks for matches this far back, so patterns repeating around it reach both
    /// outcomes of its distance check.
    const FLZ_MAX_DISTANCE: usize = 8192;

    /// Above reth's default 128 KiB transaction input limit.
    const MAX_LEN: usize = 200_000;

    /// `len` bytes where about `zero_pct` percent are zero and the rest are noise.
    fn noise(len: usize, seed: u64, zero_pct: u8) -> Vec<u8> {
        let mut state = seed | 1;
        (0..len)
            .map(|_| {
                state ^= state << 13;
                state ^= state >> 7;
                state ^= state << 17;
                if (state >> 8) % 100 < u64::from(zero_pct) { 0 } else { state as u8 }
            })
            .collect()
    }

    /// Inputs that reach every branch of `FastLZ`, not just the literal path that random bytes
    /// take.
    fn flz_input() -> impl Strategy<Value = Vec<u8>> {
        prop_oneof![
            // Short inputs, around the 13 byte minimum for matching.
            prop::collection::vec(any::<u8>(), 0..64),
            // Incompressible through compressible, up to the largest transactions.
            (0..=MAX_LEN, any::<u64>(), 0..=100u8)
                .prop_map(|(len, seed, zero_pct)| noise(len, seed, zero_pct)),
            // Runs longer than the 262 bytes a single match can encode.
            (any::<u8>(), 0..=MAX_LEN).prop_map(|(byte, len)| vec![byte; len]),
            // A noise block repeated around the maximum match distance.
            (FLZ_MAX_DISTANCE - 16..FLZ_MAX_DISTANCE + 16, any::<u64>(), 2..8usize).prop_map(
                |(period, seed, repeats)| {
                    let block = noise(period, seed, 0);
                    block.iter().copied().cycle().take(period * repeats).collect()
                }
            ),
        ]
    }

    proptest! {
        /// The pool and the executor size a transaction with separate copies of `FastLZ`:
        /// `op-alloy-flz` backs `estimated_da_size`, which pool admission and the builder's limit
        /// check use, and `op-revm` backs the executor's footprint check.
        #[test]
        fn fastlz_copies_agree(input in flz_input()) {
            prop_assert_eq!(
                op_alloy_flz::tx_estimated_size_fjord(&input),
                op_revm::estimate_tx_compressed_size(&input)
            );
        }

        /// Same as [`fastlz_copies_agree`], end to end: the footprint the pool derives for a
        /// transaction equals the one the executor charges for it, so the two also encode the
        /// transaction and apply the scalar the same way.
        #[test]
        fn pool_and_executor_footprints_agree(input in flz_input(), scalar: u16) {
            let tx: OpTransactionSigned = TxEip1559 {
                chain_id: 10,
                gas_limit: 21_000,
                to: TxKind::Call(Address::with_last_byte(0x42)),
                input: Bytes::from(input),
                ..Default::default()
            }
            .into_signed(Signature::test_signature())
            .into();
            let executor_footprint = op_revm::tx_da_footprint(&tx, u64::from(scalar));
            let encoded_len = tx.encode_2718_len();
            let pooled = OpPooledTransaction::<OpTransactionSigned, OpTransactionSigned>::new(
                Recovered::new_unchecked(tx, Address::with_last_byte(1)),
                encoded_len,
            );

            prop_assert_eq!(pooled.estimated_da_size() * u64::from(scalar), executor_footprint);
        }
    }
}
