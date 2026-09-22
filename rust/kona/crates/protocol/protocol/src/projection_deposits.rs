//! User deposits are inert on the public projection. System deposits retain execution.

use crate::L1BlockInfoTx;
use alloc::{boxed::Box, vec::Vec};
use alloy_eips::Decodable2718;
use alloy_primitives::{TxKind, address};
use kona_hardforks::{Hardfork, Hardforks, Lagoon};
use once_cell::race::OnceBox;
use op_alloy_consensus::{L1InfoDepositSource, OpTxEnvelope, TxDeposit};
const L1_BLOCK_CONTRACT: alloy_primitives::Address =
    address!("4200000000000000000000000000000000000015");

/// Classify by the authenticated deposit source, not the destination or the legacy
/// `is_system_transaction` gas flag. A user can deposit arbitrary system-call calldata.
pub fn is_projection_user_deposit(tx: &TxDeposit, index: usize) -> bool {
    if index == 0 &&
        tx.from == address!("deaddeaddeaddeaddeaddeaddeaddeaddead0001") &&
        tx.to == TxKind::Call(L1_BLOCK_CONTRACT) &&
        tx.mint == 0 &&
        tx.value.is_zero() &&
        let Ok(info) = L1BlockInfoTx::decode_calldata(&tx.input) &&
        tx.source_hash ==
            L1InfoDepositSource::new(info.block_hash(), info.sequence_number()).source_hash()
    {
        return false;
    }
    // Upgrade deposits have a different source-hash domain from portal deposits.
    // Match the complete transaction as well, so an upgrade source cannot authorize
    // arbitrary calldata. Reuse the canonical upgrade definitions instead of a
    // second list of privileged senders or contract addresses.
    !upgrades().iter().any(|system| system.source_hash == tx.source_hash && system == tx)
}

fn upgrades() -> &'static [TxDeposit] {
    static UPGRADES: OnceBox<Vec<TxDeposit>> = OnceBox::new();
    UPGRADES.get_or_init(|| {
        let mut deposits: Vec<_> = Hardforks::ECOTONE
            .txs()
            .chain(Hardforks::FJORD.txs())
            .chain(Hardforks::ISTHMUS.txs())
            .chain(Hardforks::JOVIAN.txs())
            .chain(Hardforks::KARST.txs())
            .map(|bytes| {
                let OpTxEnvelope::Deposit(tx) = OpTxEnvelope::decode_2718(&mut bytes.as_ref())
                    .expect("canonical upgrade transaction must decode")
                else {
                    panic!("canonical upgrade transaction must be a deposit")
                };
                tx.into_inner()
            })
            .collect();
        deposits.extend(Lagoon::deposits(true));
        Box::new(deposits)
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{L1BlockInfoBedrock, L1BlockInfoEcotone, L1BlockInfoIsthmus, L1BlockInfoJovian};
    use alloy_primitives::{B256, U256};
    use op_alloy_consensus::UserDepositSource;
    #[test]
    fn projection_authenticates_system_deposits() {
        for info in [
            L1BlockInfoTx::Bedrock(L1BlockInfoBedrock::default()),
            L1BlockInfoTx::Ecotone(L1BlockInfoEcotone::default()),
            L1BlockInfoTx::Isthmus(L1BlockInfoIsthmus::default()),
            L1BlockInfoTx::Jovian(L1BlockInfoJovian::default()),
        ] {
            let mut tx = TxDeposit {
                from: address!("deaddeaddeaddeaddeaddeaddeaddeaddead0001"),
                to: TxKind::Call(L1_BLOCK_CONTRACT),
                source_hash: L1InfoDepositSource::new(info.block_hash(), info.sequence_number())
                    .source_hash(),
                input: info.encode_calldata(),
                ..Default::default()
            };
            assert!(!is_projection_user_deposit(&tx, 0));
            assert!(is_projection_user_deposit(&tx, 1), "L1 attributes must be first");
            tx.source_hash = UserDepositSource::new(info.block_hash(), 0).source_hash();
            assert!(
                is_projection_user_deposit(&tx, 0),
                "copied system calldata is still a user deposit"
            );
        }
        assert!(!upgrades().is_empty());
        for tx in upgrades() {
            assert!(!is_projection_user_deposit(tx, 1));
            let mut copied = tx.clone();
            copied.source_hash = UserDepositSource::new(B256::ZERO, 0).source_hash();
            assert!(
                is_projection_user_deposit(&copied, 1),
                "portal deposits cannot impersonate upgrades"
            );
            let mut changed = tx.clone();
            changed.value += U256::from(1);
            assert!(
                is_projection_user_deposit(&changed, 1),
                "an upgrade source cannot authorize different contents"
            );
        }
    }
}
