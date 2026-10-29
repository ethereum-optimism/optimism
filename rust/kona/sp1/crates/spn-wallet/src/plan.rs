//! Decides the steps of a write command from quoted chain state, before anything is signed.
//!
//! Every check that can reject a command runs here, so a rejected command sends nothing. For
//! `fund --from-l1` that includes the SPN fee cap, which must hold before PROVE leaves L1.

use alloy_primitives::{Address, U256};
use thiserror::Error;

use crate::fmt_prove;

/// One state-changing action, in execution order.
#[derive(Debug, PartialEq, Eq)]
pub(crate) enum Step {
    Deposit(Deposit),
    /// SPN transfer from the signer to `to`. The quoted fee comes on top of `amount`.
    Transfer {
        to: Address,
        amount: U256,
    },
}

/// Approve the `VApp` for `amount` when `approve` is set, then `VApp.deposit(amount)`. SPN
/// credits the L1 sender.
#[derive(Debug, PartialEq, Eq)]
pub(crate) struct Deposit {
    pub(crate) amount: U256,
    pub(crate) approve: bool,
}

/// The signer's SPN account and the fee SPN quotes for the operation.
#[derive(Debug)]
pub(crate) struct SpnQuote {
    pub(crate) fee: U256,
    pub(crate) balance: U256,
}

/// The signer's PROVE on L1 and the `VApp`'s deposit constraints.
#[derive(Debug)]
pub(crate) struct L1Quote {
    pub(crate) balance: U256,
    pub(crate) allowance: U256,
    pub(crate) min_deposit: U256,
}

#[derive(Debug, Error, PartialEq, Eq)]
pub(crate) enum Rejected {
    #[error("SPN quotes a fee of {}, above --max-fee {}", fmt_prove(*.fee), fmt_prove(*.cap))]
    FeeAboveCap { fee: U256, cap: U256 },
    #[error("SPN balance is {}, need {} (amount plus fee)", fmt_prove(*.balance), fmt_prove(*.need))]
    SpnBalance { balance: U256, need: U256 },
    #[error("L1 PROVE balance is {}, need {}", fmt_prove(*.balance), fmt_prove(*.need))]
    L1Balance { balance: U256, need: U256 },
    #[error("deposit of {} is below the VApp minimum of {}", fmt_prove(*.amount), fmt_prove(*.min))]
    BelowMinDeposit { amount: U256, min: U256 },
}

pub(crate) fn deposit(amount: U256, l1: &L1Quote) -> Result<Deposit, Rejected> {
    if amount < l1.min_deposit {
        return Err(Rejected::BelowMinDeposit { amount, min: l1.min_deposit });
    }
    if l1.balance < amount {
        return Err(Rejected::L1Balance { balance: l1.balance, need: amount });
    }
    Ok(Deposit { amount, approve: l1.allowance < amount })
}

/// Checks a withdraw of `amount`; SPN takes the quoted fee on top.
pub(crate) fn withdraw(
    amount: U256,
    spn: &SpnQuote,
    max_fee: Option<U256>,
) -> Result<(), Rejected> {
    check_fee(spn.fee, max_fee)?;
    check_spn_balance(spn.balance, amount.saturating_add(spn.fee))
}

/// With `from_l1`, deposits `amount` plus the transfer fee into the signer's SPN account first.
pub(crate) fn fund(
    to: Address,
    amount: U256,
    spn: &SpnQuote,
    max_fee: Option<U256>,
    from_l1: Option<&L1Quote>,
) -> Result<Vec<Step>, Rejected> {
    check_fee(spn.fee, max_fee)?;
    let total = amount.saturating_add(spn.fee);
    let mut steps = Vec::with_capacity(2);
    match from_l1 {
        Some(l1) => steps.push(Step::Deposit(deposit(total, l1)?)),
        None => check_spn_balance(spn.balance, total)?,
    }
    steps.push(Step::Transfer { to, amount });
    Ok(steps)
}

fn check_fee(fee: U256, max_fee: Option<U256>) -> Result<(), Rejected> {
    match max_fee {
        Some(cap) if fee > cap => Err(Rejected::FeeAboveCap { fee, cap }),
        _ => Ok(()),
    }
}

fn check_spn_balance(balance: U256, need: U256) -> Result<(), Rejected> {
    if balance < need {
        return Err(Rejected::SpnBalance { balance, need });
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    const PROVE: U256 = U256::from_limbs([1_000_000_000_000_000_000, 0, 0, 0]);
    const TO: Address = Address::repeat_byte(0x22);

    fn prove(n: u64) -> U256 {
        U256::from(n) * PROVE
    }

    fn spn(fee: u64, balance: u64) -> SpnQuote {
        SpnQuote { fee: prove(fee), balance: prove(balance) }
    }

    fn l1(balance: u64, allowance: u64, min_deposit: u64) -> L1Quote {
        L1Quote {
            balance: prove(balance),
            allowance: prove(allowance),
            min_deposit: prove(min_deposit),
        }
    }

    #[test]
    fn fund_from_spn_transfers_when_balance_covers_amount_and_fee() {
        let steps = fund(TO, prove(10), &spn(1, 11), Some(prove(1)), None).unwrap();
        assert_eq!(steps, vec![Step::Transfer { to: TO, amount: prove(10) }]);
    }

    #[test]
    fn fund_from_spn_rejects_balance_that_misses_the_fee() {
        let err = fund(TO, prove(10), &spn(1, 10), Some(prove(1)), None).unwrap_err();
        assert_eq!(err, Rejected::SpnBalance { balance: prove(10), need: prove(11) });
    }

    #[test]
    fn fund_rejects_an_amount_whose_total_overflows() {
        let err = fund(TO, U256::MAX, &spn(1, 50), None, Some(&l1(50, 50, 1))).unwrap_err();
        assert_eq!(err, Rejected::L1Balance { balance: prove(50), need: U256::MAX });
    }

    #[test]
    fn fund_from_l1_deposits_amount_plus_fee_before_the_transfer() {
        let steps = fund(TO, prove(10), &spn(1, 0), Some(prove(1)), Some(&l1(50, 0, 5))).unwrap();
        assert_eq!(
            steps,
            vec![
                Step::Deposit(Deposit { amount: prove(11), approve: true }),
                Step::Transfer { to: TO, amount: prove(10) },
            ]
        );
    }

    #[test]
    fn fund_from_l1_rejects_a_fee_above_the_cap_before_any_deposit() {
        let err =
            fund(TO, prove(10), &spn(2, 0), Some(prove(1)), Some(&l1(50, 50, 5))).unwrap_err();
        assert_eq!(err, Rejected::FeeAboveCap { fee: prove(2), cap: prove(1) });
    }

    #[test]
    fn fund_from_l1_checks_the_minimum_against_amount_plus_fee() {
        let steps = fund(TO, prove(4), &spn(1, 0), None, Some(&l1(50, 50, 5))).unwrap();
        assert_eq!(steps[0], Step::Deposit(Deposit { amount: prove(5), approve: false }));

        let err = fund(TO, prove(3), &spn(1, 0), None, Some(&l1(50, 50, 5))).unwrap_err();
        assert_eq!(err, Rejected::BelowMinDeposit { amount: prove(4), min: prove(5) });
    }

    #[test]
    fn deposit_approves_only_when_the_allowance_is_short() {
        assert_eq!(
            deposit(prove(10), &l1(10, 9, 1)).unwrap(),
            Deposit { amount: prove(10), approve: true }
        );
        assert_eq!(
            deposit(prove(10), &l1(10, 10, 1)).unwrap(),
            Deposit { amount: prove(10), approve: false }
        );
        assert_eq!(
            deposit(prove(10), &l1(9, 10, 1)).unwrap_err(),
            Rejected::L1Balance { balance: prove(9), need: prove(10) }
        );
    }

    #[test]
    fn withdraw_needs_amount_plus_fee_and_respects_the_cap() {
        assert_eq!(withdraw(prove(5), &spn(1, 6), Some(prove(1))), Ok(()));
        assert_eq!(
            withdraw(prove(5), &spn(1, 5), Some(prove(1))).unwrap_err(),
            Rejected::SpnBalance { balance: prove(5), need: prove(6) }
        );
        assert_eq!(
            withdraw(prove(5), &spn(1, 6), Some(U256::ZERO)).unwrap_err(),
            Rejected::FeeAboveCap { fee: prove(1), cap: U256::ZERO }
        );
    }
}
