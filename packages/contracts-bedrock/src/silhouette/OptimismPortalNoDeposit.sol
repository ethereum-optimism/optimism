// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";

/// @custom:proxied true
/// @title OptimismPortalNoDeposit
/// @notice Silhouette's gated portal: `OptimismPortal2` with the deposit path closed (PLAN DR-2,
///         ruling G5 D1). A silhouette chain takes NO user deposits, ever — an open portal would
///         let a stranger force a transaction into a private chain, and the guest's no-deposit
///         soundness rule (the inflation gate: no 0x7E tx other than the L1-info tx may appear in
///         any block) is enforced by ASSUMPTION in-circuit. This contract is what makes that
///         assumption true on L1 rather than promised.
///
///         Storage layout is IDENTICAL to `OptimismPortal2` — nothing is added, nothing is
///         reordered, nothing is removed. It is a behaviour-only override, which is a strictly
///         weaker claim than `OptimismPortalSTF`'s (that one appended two slots), and
///         `test/silhouette/OptimismPortalNoDepositFork.t.sol` proves it against the live proxy.
///
/// @dev    Why inherit-and-override rather than a purpose-built always-revert contract: the portal
///         still OWES its callers a working view surface, and a blanket-revert portal would take
///         P's whole settlement stack down with it. `SystemConfig.paused()` reads
///         `portal.ethLockbox()`; `SystemConfig.disputeGameFactory()` reads the DGF address THROUGH
///         the portal; `ETHLockbox._authorizePortal` reads `superchainConfig()` and
///         `proxyAdminOwner()`; `L1CrossDomainMessenger` reads `l2Sender()`; and OPCM discovery
///         reads several more. The ASR's `paused()` gates every dispute game and DelayedWETH.
///         Inheriting keeps all of it untouched and changes exactly one function.
///
/// @dev    Why not `pause()`: it does not gate deposits at all. `_assertNotPaused()` is called from
///         exactly three sites in `OptimismPortal2` — `proveWithdrawalTransaction`,
///         `migrateToSharedDisputeGame`, `finalizeWithdrawalTransactionExternalProof` — all on the
///         withdrawal side. `depositTransaction` carries no such check, and `ETHLockbox` says the
///         rule outright next to the code: unlocks are blocked when paused, locks are not. Pausing
///         P would block P's WITHDRAWALS and leave its DEPOSITS wide open — the exact inverse of
///         what DR-2 needs — and it expires by itself after `PAUSE_EXPIRY` (3 months), which is the
///         opposite of an irreversible-proof gate.
///
/// @dev    Two paths close for free, and both matter:
///         * `receive()` makes an INTERNAL call to `depositTransaction`, and Solidity dispatches
///           internal calls to `virtual` functions through the override — so plain ETH sent to the
///           portal reverts without this contract mentioning `receive()` at all.
///         * `TransactionDeposited` is emitted at exactly ONE place in the whole `src/` tree (the
///           last statement of `depositTransaction`), which is what makes a one-function override a
///           COMPLETE gate rather than a partial one. There is no second emitter to also close.
///
/// @dev    The base's `depositTransaction` needed one keyword — `virtual` — and nothing else. That
///         one-word patch to `OptimismPortal2.sol` is the entire diff to shared code, and it is the
///         same patch `OptimismPortalSTF` took.
///
/// @dev    A stub alone is undoable in one transaction by the OpChainProxyAdmin owner, and worse,
///         `OPContractsManagerV2._upgrade(...)` sets the portal implementation UNCONDITIONALLY
///         without reading the current one — so a routine hardfork upgrade would silently restore
///         the stock, deposit-accepting portal under a chain whose guest assumes deposits are
///         impossible. The deployment therefore pairs this implementation with a PROXY FREEZE:
///         `changeProxyAdmin(portalProxy, <ProxyAdmin whose ownership is renounced>)`. This
///         contract is only half the gate; see `rotation/RUNBOOK.md`.
contract OptimismPortalNoDeposit is OptimismPortal2 {
    /// @notice Thrown by `depositTransaction` on every call. A silhouette chain has no deposit path.
    /// @dev    A NAMED error rather than an arithmetic panic (which is what the rejected
    ///         `maxResourceLimit = 0` alternative would have produced) so that the refusal is
    ///         legible both to a caller and to anyone reading the portal.
    error OptimismPortalNoDeposit_DepositsDisabled();

    /// @notice Semantic version.
    /// @custom:semver 5.8.0+silhouette.nodeposit.1
    function version() public pure override returns (string memory) {
        return "5.8.0+silhouette.nodeposit.1";
    }

    /// @param _proofMaturityDelaySeconds The proof maturity delay in seconds. MUST match the value
    ///        the proxy was deployed against, because it is an immutable read by the withdrawal
    ///        path this contract leaves entirely intact.
    constructor(uint256 _proofMaturityDelaySeconds) OptimismPortal2(_proofMaturityDelaySeconds) { }

    /// @notice Always reverts. This is the gate.
    /// @dev    Reverts BEFORE any state is touched and before the `metered` modifier's gas burn, so
    ///         a caller loses nothing but the intrinsic gas of a failed call. The parameters are
    ///         unnamed because none of them is read.
    function depositTransaction(address, uint256, uint64, bool, bytes memory) public payable override {
        revert OptimismPortalNoDeposit_DepositsDisabled();
    }
}
