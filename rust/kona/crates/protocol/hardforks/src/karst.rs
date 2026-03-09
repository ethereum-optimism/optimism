//! Module containing a `TxDeposit` builder for the Karst network upgrade transactions.
//!
//! Karst network upgrade transactions are defined in the [OP Stack Specs][specs].
//!
//! [specs]: https://github.com/ethereum-optimism/specs/tree/main/specs/protocol/karst

use alloy_primitives::Bytes;

use crate::Hardfork;

/// The Karst network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Karst;

impl Hardfork for Karst {
    /// Constructs the network upgrade transactions.
    /// Karst has no upgrade transactions (empty NUT bundle).
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_ {
        core::iter::empty()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec::Vec;

    #[test]
    fn test_karst_no_upgrade_txs() {
        let karst = Karst;
        let txs: Vec<_> = karst.txs().collect();
        assert!(txs.is_empty());
    }
}
