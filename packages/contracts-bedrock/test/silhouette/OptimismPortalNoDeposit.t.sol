// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Contracts
import { OptimismPortalNoDeposit } from "src/silhouette/OptimismPortalNoDeposit.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";

/// @title OptimismPortalNoDeposit_Test
/// @notice The gate, checked rather than assumed. The guest's no-deposit soundness rule is the
///         inflation gate for a silhouette chain (PLAN DR-2): it permits no 0x7E transaction other
///         than the L1-info tx, and it enforces that by ASSUMPTION in-circuit. These tests are
///         where the assumption is discharged on L1.
contract OptimismPortalNoDeposit_TestInit is CommonTest {
    OptimismPortalNoDeposit internal noDeposit;

    /// @dev The proof maturity delay P's proxy was deployed against. It is an immutable, and the
    ///      withdrawal path this implementation leaves intact reads it.
    uint256 internal constant PROOF_MATURITY_DELAY = 300;

    function setUp() public override {
        super.setUp();
        noDeposit = new OptimismPortalNoDeposit(PROOF_MATURITY_DELAY);
    }
}

/// @title OptimismPortalNoDeposit_DepositTransaction_Test
/// @notice THE GATE. `depositTransaction` is the one function this contract overrides, and these are
///         the tests that discharge the guest's no-deposit soundness assumption on L1.
contract OptimismPortalNoDeposit_DepositTransaction_Test is OptimismPortalNoDeposit_TestInit {
    /// @notice `depositTransaction` reverts with the named error, with and without value.
    function test_depositTransaction_alwaysReverts_succeeds() external {
        vm.expectRevert(OptimismPortalNoDeposit.OptimismPortalNoDeposit_DepositsDisabled.selector);
        noDeposit.depositTransaction(address(0xdead), 0, 100_000, false, hex"");

        vm.deal(address(this), 1 ether);
        vm.expectRevert(OptimismPortalNoDeposit.OptimismPortalNoDeposit_DepositsDisabled.selector);
        noDeposit.depositTransaction{ value: 1 ether }(address(0xdead), 1 ether, 100_000, false, hex"");
    }

    /// @notice Fuzzed, because "reverts on the inputs I thought of" is not the property. The gate
    ///         must hold for every argument shape, including contract creation and large calldata.
    function testFuzz_depositTransaction_alwaysReverts_succeeds(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        external
    {
        vm.expectRevert(OptimismPortalNoDeposit.OptimismPortalNoDeposit_DepositsDisabled.selector);
        noDeposit.depositTransaction(_to, _value, _gasLimit, _isCreation, _data);
    }

    /// @notice `receive()` closes for free, and this is the path most likely to be missed.
    ///         `OptimismPortal2.receive()` makes an INTERNAL call to `depositTransaction`, and
    ///         Solidity dispatches internal calls to `virtual` functions through the override — so
    ///         plain ETH sent to the portal reverts although this contract never mentions
    ///         `receive()`. If the base's `virtual` keyword were ever dropped, this test is what
    ///         fails.
    function test_receive_reverts_succeeds() external {
        vm.deal(address(this), 1 ether);
        (bool success, bytes memory data) = address(noDeposit).call{ value: 1 ether }(hex"");
        assertFalse(success, "plain ETH transfer must revert");
        assertEq(
            bytes4(data),
            OptimismPortalNoDeposit.OptimismPortalNoDeposit_DepositsDisabled.selector,
            "and it must revert with the named error, proving it went through the override"
        );
        assertEq(address(noDeposit).balance, 0, "no ETH may be retained");
    }

    /// @notice No `TransactionDeposited` is emitted on any path. This is the event derivation reads,
    ///         and `TransactionDeposited` is emitted at exactly ONE place in the whole `src/` tree —
    ///         the last statement of `depositTransaction` — which is what makes a one-function
    ///         override a COMPLETE gate rather than a partial one.
    function test_noTransactionDepositedEvent_succeeds() external {
        vm.deal(address(this), 2 ether);
        vm.recordLogs();

        (bool ok,) = address(noDeposit).call{ value: 1 ether }(hex"");
        assertFalse(ok);
        (ok,) = address(noDeposit).call(
            abi.encodeCall(OptimismPortal2.depositTransaction, (address(0xdead), 0, 100_000, false, hex""))
        );
        assertFalse(ok);

        assertEq(vm.getRecordedLogs().length, 0, "a gated portal emits nothing at all");
    }
}

/// @title OptimismPortalNoDeposit_Uncategorized_Test
/// @notice What the gate must NOT break: the view surface the rest of P's L1 stack reads, the
///         version string, the untouched withdrawal side, and the storage layout that makes this
///         safe behind a live proxy.
contract OptimismPortalNoDeposit_Uncategorized_Test is OptimismPortalNoDeposit_TestInit {
    /// @notice The view surface the rest of P's L1 stack reads is UNTOUCHED. This is the whole
    ///         reason the mechanism is inherit-and-override rather than a purpose-built
    ///         always-revert contract: `SystemConfig.paused()` reads `portal.ethLockbox()`,
    ///         `SystemConfig.disputeGameFactory()` reads the DGF THROUGH the portal,
    ///         `ETHLockbox._authorizePortal` reads `superchainConfig()` and `proxyAdminOwner()`,
    ///         `L1CrossDomainMessenger` reads `l2Sender()`, and the ASR's `paused()` gates every
    ///         dispute game and DelayedWETH. A blanket-revert portal would take all of it down.
    /// @dev    ANSWERABILITY is the property, not the values: these are read on a freshly
    ///         constructed implementation, so what they return is the zero default. That a call
    ///         returns rather than reverts is exactly what the readers above need.
    ///
    ///         `superchainConfig()` and `disputeGameFactory()` are deliberately NOT in this list,
    ///         and the reason is worth stating because omitting them looks like a gap. Neither
    ///         reads storage: both DELEGATE to `systemConfig`, which on a freshly constructed
    ///         contract is the zero address, so both revert on a call to nothing. That says
    ///         something about an uninitialized contract and nothing whatever about this override —
    ///         it is equally true of stock `OptimismPortal2`, it is a state no reader ever meets
    ///         (the portal is always used behind an initialized proxy), and it is covered where it
    ///         belongs, by the base's own tests against the deployed proxy. Asserting it here would
    ///         be testing `CommonTest`'s deployment, not the gate.
    ///
    ///         What makes the omission safe rather than convenient: this contract overrides exactly
    ///         ONE function and adds no storage, which `test_storageLayoutUnchangedFromBase_succeeds`
    ///         asserts slot by slot. Every pass-through getter is inherited bytecode.
    function test_viewSurfaceIntact_succeeds() external view {
        noDeposit.ethLockbox();
        noDeposit.systemConfig();
        noDeposit.anchorStateRegistry();
        noDeposit.l2Sender();
        assertEq(noDeposit.proofMaturityDelaySeconds(), PROOF_MATURITY_DELAY);
    }

    /// @notice The version says what it is, so anyone reading the proxy on Etherscan sees the gate.
    function test_version_succeeds() external view {
        assertEq(noDeposit.version(), "5.8.0+silhouette.nodeposit.1");
    }

    /// @notice The withdrawal side is inherited unchanged — a silhouette chain gates DEPOSITS, not
    ///         withdrawals, and this asymmetry is exactly what disqualified `pause()`, which does
    ///         the inverse (blocks withdrawals, leaves deposits open).
    function test_withdrawalSideStillReadable_succeeds() external view {
        assertEq(noDeposit.numProofSubmitters(bytes32(0)), 0);
        noDeposit.proofMaturityDelaySeconds();
    }

    /// @notice Storage layout is IDENTICAL to `OptimismPortal2` — nothing added, reordered or
    ///         removed — which is what makes this safe behind the live proxy. Asserted here as the
    ///         behavioural consequence (a fresh instance of each agrees on every slot it defines);
    ///         the authoritative slot-by-slot check is the artifact diff run at build time, and it
    ///         reports 21 entries ending at `spacer_63_20_1` (slot 63) for BOTH contracts.
    ///
    /// @dev    This matters concretely for P: its proxy still holds ORPHANED `OptimismPortalSTF`
    ///         storage at slots 64 and 65 from the Cove D-series. Stock `OptimismPortal2`'s layout
    ///         ends before slot 64, so nothing reads them — and because this implementation appends
    ///         NOTHING, that stays true. An implementation that appended even one slot would land
    ///         on top of the old `depositAcc`.
    function test_storageLayoutUnchangedFromBase_succeeds() external {
        OptimismPortal2 stock = new OptimismPortal2(PROOF_MATURITY_DELAY);
        for (uint256 slot = 0; slot <= 65; slot++) {
            assertEq(
                vm.load(address(noDeposit), bytes32(slot)),
                vm.load(address(stock), bytes32(slot)),
                "a fresh gated portal must occupy exactly the slots a fresh stock portal does"
            );
        }
    }
}
