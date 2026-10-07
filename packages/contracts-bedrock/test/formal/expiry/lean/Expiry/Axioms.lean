import Expiry.Safety
import Expiry.Counterexamples

/-! Axioms used by the main theorems and the non-vacuity checks (printed at build time). -/

open Expiry Expiry.Examples

#print axioms exporterSilentBeforeUpgrade
#print axioms joinNeedsNoHistoryCheck
#print axioms noDoubleSpend
#print axioms refundImpliesExpired
#print axioms atMostOneRefund
#print axioms noForgedFact
#print axioms expiredImpliesNeverRelayable
#print axioms expired_no_relay_step
#print axioms onlyDestinationCanExport
#print axioms safety
#print axioms safety_without_targetRule
#print axioms messengerSilentAfterUpgrade
#print axioms refund_reachable
#print axioms relay_reachable_at_edge
#print axioms lateJoin_multiSource_reachable
#print axioms forgery_reachable_but_harmless
#print axioms safe_variants
#print axioms legacyResend_reachable
#print axioms cex_messengerTrusted
#print axioms cex_nonstandardJoin
#print axioms cex_periodBelowWindow
#print axioms cex_nonStrict
#print axioms cex_resendNoRestart
#print axioms cex_noRealMessengerCheck
#print axioms cex_noLockboxCheck
#print axioms cex_noLockboxCheck_fakePortal
#print axioms cex_sysConfigInconsistent
#print axioms cex_noUnsafeTargetCheck
#print axioms cex_noSenderCheck
#print axioms cex_hashCollision
#print axioms cex_duplicateChainId
#print axioms cex_exporterGovernanceUpgrade
#print axioms messengerSpeaks_without_targetRule
