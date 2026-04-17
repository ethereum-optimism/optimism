//! Module containing a `TxDeposit` builder for the Karst network upgrade transactions.
//!
//! Karst network upgrade transactions are defined in the [OP Stack Specs][specs].
//! The transactions are loaded from a JSON NUT bundle at compile time via `build.rs`.
//!
//! [specs]: https://github.com/ethereum-optimism/specs/tree/main/specs/protocol/karst

use alloc::vec::Vec;
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::Bytes;

use crate::Hardfork;

// Include the build-script-generated NUT bundle constructor.
include!(concat!(env!("OUT_DIR"), "/karst_nut_bundle.rs"));

/// The Karst network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Karst;

impl Karst {
    /// Returns the parsed NUT bundle for Karst.
    fn bundle() -> op_alloy_consensus::NutBundle {
        karst_nut_bundle()
    }
}

impl Hardfork for Karst {
    /// Constructs the Karst network upgrade transactions from the NUT bundle.
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_ {
        let bundle = Self::bundle();
        let deposits = bundle
            .to_deposit_transactions()
            .expect("karst NUT bundle is invalid");
        deposits
            .into_iter()
            .map(|tx| {
                let mut encoded = Vec::new();
                tx.encode_2718(&mut encoded);
                Bytes::from(encoded)
            })
    }

    /// Returns the total gas required by Karst upgrade transactions.
    fn upgrade_gas(&self) -> u64 {
        Self::bundle().total_gas()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec::Vec;

    #[test]
    fn test_karst_upgrade_txs() {
        let karst = Karst;
        let txs: Vec<_> = karst.txs().collect();
        assert_eq!(txs.len(), 32);

        // All encoded deposit txs start with the deposit type byte (0x7e).
        for tx in &txs {
            assert_eq!(tx[0], 0x7e);
        }
    }

    #[test]
    fn test_karst_upgrade_gas() {
        let karst = Karst;
        assert_eq!(karst.upgrade_gas(), 51_600_000);
    }

    #[test]
    fn test_karst_bundle_valid() {
        let bundle = Karst::bundle();
        assert_eq!(bundle.fork_name, "Karst");
        assert_eq!(bundle.transactions.len(), 32);

        // Verify all transactions have non-empty intents.
        for tx in &bundle.transactions {
            assert!(!tx.intent.is_empty());
        }
    }
}
