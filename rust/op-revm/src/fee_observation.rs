//! Opt-in, synchronous observation of actual fee balance changes.
//!
//! No account loads, journal writes, or fee calculations are performed by the observer. A scoped
//! thread-local sink avoids adding diagnostic state to consensus results or the L1-info cache.

use revm::{
    primitives::{Address, U256},
    state::EvmState,
};
use std::cell::RefCell;

/// Balance changes measured at the charging, reimbursement, and fee-distribution sites.
/// Missing values indicate an unobserved operation or an unexpected direction of balance change.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct FeeTransfers {
    /// Sender's upfront fee debit, excluding transaction value.
    pub sender_charge: Option<U256>,
    /// Unused-gas and operator-fee reimbursement, before any SDM settlement.
    pub sender_reimbursement: Option<U256>,
    /// Priority fee actually credited to the beneficiary.
    pub beneficiary_credit: Option<U256>,
    /// Base fee actually credited to its vault.
    pub base_fee_credit: Option<U256>,
    /// Operator fee actually credited to its vault.
    pub operator_fee_credit: Option<U256>,
    /// Non-refundable L1 fee actually credited to its vault.
    pub l1_fee_credit: Option<U256>,
}

std::thread_local! {
    static TRANSFERS: RefCell<Option<FeeTransfers>> = const { RefCell::new(None) };
}

/// Observe one synchronous EVM execution. Each scope starts empty, and the previous sink is
/// restored on return or unwind, so failed transactions cannot leak observations into later ones.
pub fn observe<T>(execute: impl FnOnce() -> T) -> (T, FeeTransfers) {
    struct Restore(Option<FeeTransfers>);
    impl Drop for Restore {
        fn drop(&mut self) {
            TRANSFERS.with(|slot| *slot.borrow_mut() = self.0);
        }
    }

    let previous = TRANSFERS.with(|slot| slot.replace(Some(FeeTransfers::default())));
    let _restore = Restore(previous);
    let result = execute();
    let transfers = TRANSFERS.with(|slot| slot.borrow_mut().take().unwrap_or_default());
    (result, transfers)
}

pub(crate) fn record(update: impl FnOnce(&mut FeeTransfers)) {
    TRANSFERS.with(|slot| {
        if let Some(transfers) = slot.borrow_mut().as_mut() {
            update(transfers);
        }
    });
}

/// A read-only snapshot immediately around one fee credit. Separate snapshots keep fee legs
/// distinct even when the sender, beneficiary, and vault addresses overlap.
pub(crate) struct CreditSnapshot {
    address: Address,
    before: Option<U256>,
}

impl CreditSnapshot {
    pub(crate) fn capture(state: &EvmState, address: Address) -> Option<Self> {
        TRANSFERS.with(|slot| slot.borrow().is_some()).then(|| Self {
            address,
            before: state.get(&address).map(|account| account.info.balance),
        })
    }

    pub(crate) fn credit(self, state: &EvmState) -> Option<U256> {
        let Some(account) = state.get(&self.address) else {
            // A skipped credit (e.g. fee charging disabled) need not load the account.
            return self.before.is_none().then_some(U256::ZERO);
        };
        // If the fee operation itself loaded this account, original_info is its balance on load.
        let before = self.before.unwrap_or_else(|| account.original_info().balance);
        account.info.balance.checked_sub(before)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use revm::state::{Account, AccountInfo};

    #[test]
    fn credit_snapshots_exclude_existing_funds_and_separate_overlapping_legs() {
        observe(|| {
            for already_loaded in [false, true] {
                let address = Address::ZERO;
                let mut state = EvmState::default();
                let account =
                    Account::from(AccountInfo { balance: U256::from(100), ..Default::default() });
                if already_loaded {
                    state.insert(address, account.clone());
                }
                let before = CreditSnapshot::capture(&state, address).unwrap();
                let account = state.entry(address).or_insert(account);
                account.info.balance += U256::from(5);
                assert_eq!(before.credit(&state), Some(U256::from(5)));
                // Another fee leg credits the same address. Do not count the previous credit.
                let before = CreditSnapshot::capture(&state, address).unwrap();
                state.get_mut(&address).unwrap().info.balance += U256::from(7);
                assert_eq!(before.credit(&state), Some(U256::from(7)));
            }
        });
    }

    #[test]
    fn observation_scopes_do_not_leak_on_nesting_errors_or_unwind() {
        let (_, outer) = observe(|| {
            record(|fees| fees.sender_charge = Some(U256::from(1)));
            let (result, inner) = observe(|| {
                record(|fees| fees.sender_charge = Some(U256::from(2)));
                Err::<(), _>("execution error")
            });
            assert!(result.is_err());
            assert_eq!(inner.sender_charge, Some(U256::from(2)));
            assert!(
                std::panic::catch_unwind(|| observe(|| {
                    record(|fees| fees.sender_charge = Some(U256::from(3)));
                    panic!("execution panic");
                }))
                .is_err()
            );
        });
        assert_eq!(outer.sender_charge, Some(U256::from(1)));
        // A subsequent transaction starts empty, not with the last transaction's observation.
        let (_, next) = observe(|| {});
        assert_eq!(next, FeeTransfers::default());
        assert!(CreditSnapshot::capture(&EvmState::default(), Address::ZERO).is_none());
    }
}
