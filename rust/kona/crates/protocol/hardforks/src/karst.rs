//! Module containing a `TxDeposit` builder for the Karst network upgrade transactions.
//!
//! Karst network upgrade transactions are defined in the [OP Stack Specs][specs].
//! The transactions are loaded from a JSON NUT bundle at compile time via `build.rs`.
//!
//! [specs]: https://github.com/ethereum-optimism/specs/tree/main/specs/protocol/karst

// Include the build-script-generated NUT bundle constructor.
include!(concat!(env!("OUT_DIR"), "/karst_nut_bundle.rs"));

/// The Karst network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Karst;

impl_hardfork_from_bundle!(Karst, karst_nut_bundle);

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Hardfork;
    use alloc::vec::Vec;
    use alloy_primitives::Bytes;

    #[test]
    fn test_karst_upgrade_txs() {
        let karst = Karst;
        let txs: Vec<Bytes> = karst.txs().collect();
        assert_eq!(txs.len(), 31);

        // All encoded deposit txs start with the deposit type byte (0x7e).
        for tx in &txs {
            assert_eq!(tx[0], 0x7e);
        }
    }

    #[test]
    fn test_karst_upgrade_gas() {
        let karst = Karst;
        assert_eq!(karst.upgrade_gas(), 55_370_657);
    }

    #[test]
    fn test_karst_bundle_valid() {
        let bundle = karst_nut_bundle();
        assert_eq!(bundle.fork_name, "Karst");
        assert_eq!(bundle.transactions.len(), 31);

        // Verify all transactions have non-empty intents.
        for tx in &bundle.transactions {
            assert!(!tx.intent.is_empty());
        }
    }
}
