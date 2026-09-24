//! Observational fee-settlement metrics for public Verify execution only.
//!
//! These checks must never change block validity. Produce instrumentation belongs in
//! optimism-premium, not in this shared executor.

use super::{PostExecAdjustment, PostExecState};
use alloy_primitives::U256;
use op_revm::fee_observation::FeeTransfers;

/// Reth's Prometheus recorder exports this as `reth_optimism_sdm_fee_settlements`.
const SDM_FEE_SETTLEMENTS_METRIC: &str = "optimism_sdm.fee_settlements";
const SDM_FEE_CHARGE_CHECKS_METRIC: &str = "optimism_sdm.fee_charge_checks";

/// Compare settlement's fee assumptions with independently observed balance changes at the EVM
/// fee-transfer sites. Do not substitute recomputed credits when an observation is missing.
fn fee_charge_result(
    adjustment: &PostExecAdjustment,
    fees: Option<&FeeTransfers>,
    expected_credits: [U256; 3],
) -> &'static str {
    let Some(FeeTransfers {
        sender_charge: Some(charge),
        sender_reimbursement: Some(reimbursement),
        beneficiary_credit: Some(beneficiary),
        base_fee_credit: Some(base_fee),
        operator_fee_credit: Some(operator_fee),
        l1_fee_credit: Some(l1_fee),
    }) = fees
    else {
        return "unavailable";
    };
    let refundable_credits =
        beneficiary.checked_add(*base_fee).and_then(|sum| sum.checked_add(*operator_fee));
    let total_credits = refundable_credits.and_then(|sum| sum.checked_add(*l1_fee));
    let net_charge = charge.checked_sub(*reimbursement);
    if net_charge.is_none() ||
        total_credits.is_none() ||
        net_charge != total_credits ||
        [*beneficiary, *base_fee, *operator_fee] != expected_credits ||
        adjustment.beneficiary_balance_delta > *beneficiary ||
        adjustment.base_fee_balance_delta > *base_fee ||
        adjustment.operator_fee_balance_delta > *operator_fee ||
        refundable_credits.is_none_or(|credit| adjustment.sender_balance_delta > credit)
    {
        "mismatch"
    } else {
        "ok"
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum PostExecSettlementResult {
    Ok,
    RefundExceedsGas,
    NonConserving,
    RecipientOverdebit,
}

impl PostExecSettlementResult {
    const ALL: [Self; 4] =
        [Self::Ok, Self::RefundExceedsGas, Self::NonConserving, Self::RecipientOverdebit];

    const fn as_str(self) -> &'static str {
        match self {
            Self::Ok => "ok",
            Self::RefundExceedsGas => "refund_exceeds_gas",
            Self::NonConserving => "non_conserving",
            Self::RecipientOverdebit => "recipient_overdebit",
        }
    }
}

impl PostExecState {
    pub(super) fn init_settlement_metrics(&self) {
        if !self.is_verifying() {
            return;
        }
        metrics::describe_counter!(
            SDM_FEE_SETTLEMENTS_METRIC,
            "Verify-mode SDM fee settlements: committed patch checks and rejected attempts."
        );
        for result in PostExecSettlementResult::ALL {
            metrics::counter!(SDM_FEE_SETTLEMENTS_METRIC, "result" => result.as_str()).increment(0);
        }
        metrics::describe_counter!(
            SDM_FEE_CHARGE_CHECKS_METRIC,
            "Verify-mode SDM settlement checks against observed EVM fee transfers."
        );
        for result in ["ok", "mismatch", "unavailable"] {
            metrics::counter!(SDM_FEE_CHARGE_CHECKS_METRIC, "result" => result).increment(0);
        }
    }

    pub(super) fn record_fee_charge_check(
        &self,
        adjustment: &PostExecAdjustment,
        fees: Option<&FeeTransfers>,
        expected_credits: [U256; 3],
    ) {
        if self.is_verifying() {
            let result = fee_charge_result(adjustment, fees, expected_credits);
            metrics::counter!(SDM_FEE_CHARGE_CHECKS_METRIC, "result" => result).increment(1);
        }
    }

    pub(super) fn record_settlement_result(&self, result: PostExecSettlementResult) {
        if self.is_verifying() {
            metrics::counter!(SDM_FEE_SETTLEMENTS_METRIC, "result" => result.as_str()).increment(1);
        }
    }

    /// Classify a nonzero patch for recording after commit. This checks the intended deltas,
    /// not the applied account balances, and does not impose a new consensus rule.
    pub(super) fn committed_settlement_result(
        &self,
        adjustment: Option<&PostExecAdjustment>,
    ) -> Option<PostExecSettlementResult> {
        if !self.is_verifying() {
            return None;
        }
        let adjustment = adjustment.filter(|adjustment| adjustment.refund > 0)?;
        let recipient_debits = adjustment
            .beneficiary_balance_delta
            .checked_add(adjustment.base_fee_balance_delta)
            .and_then(|sum| sum.checked_add(adjustment.operator_fee_balance_delta));
        Some(if recipient_debits == Some(adjustment.sender_balance_delta) {
            PostExecSettlementResult::Ok
        } else {
            PostExecSettlementResult::NonConserving
        })
    }
}
