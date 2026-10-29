// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";

// Contracts
import { MockComplianceModule } from "scripts/deploy/mocks/MockComplianceModule.sol";

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @title StubPortal
/// @notice Stands in for the Portal. Its completion entry points are empty, matching the real
///         Portal until the hook paths land.
contract StubPortal {
    address public bridgeHook;
    address public systemConfig;
    bool public paused;

    function setBridgeHook(address _hook) external {
        bridgeHook = _hook;
    }

    function setSystemConfig(address _systemConfig) external {
        systemConfig = _systemConfig;
    }

    function setPaused(bool _paused) external {
        paused = _paused;
    }

    function completeDepositTransaction(BridgeHookTypes.Item memory) external payable { }

    function completeWithdrawalTransaction(BridgeHookTypes.Item memory) external payable { }

    /// @notice Calls the hook the way the Portal would, so tests can reach the Portal-gated
    ///         entry points without impersonating an address that has no code.
    function callHold(MockComplianceModule _module, BridgeHookTypes.Item calldata _item) external payable {
        _module.holdDeposit{ value: msg.value }(_item);
    }

    function callScreenDeposit(
        MockComplianceModule _module,
        BridgeHookTypes.Item calldata _item
    )
        external
        returns (bool)
    {
        return _module.screenDeposit(_item);
    }

    function callScreenWithdrawal(
        MockComplianceModule _module,
        BridgeHookTypes.Item calldata _item,
        uint256 _finalizableAt
    )
        external
        returns (bool)
    {
        return _module.screenWithdrawal(_item, _finalizableAt);
    }

    receive() external payable { }
}

/// @title StubPolicy
/// @notice Answers from a settable flag and can be told to re-enter the module.
contract StubPolicy {
    bool public pass = true;
    bool public reenter;
    MockComplianceModule public module;
    BridgeHookTypes.Item internal reentryItem;

    error StubPolicy_Rejected();

    bool public rejects;

    function setPass(bool _pass) external {
        pass = _pass;
    }

    function setRejects(bool _rejects) external {
        rejects = _rejects;
    }

    function setModule(MockComplianceModule _module) external {
        module = _module;
    }

    function armReentry(BridgeHookTypes.Item calldata _item) external {
        reenter = true;
        reentryItem = _item;
    }

    function screen(BridgeHookTypes.Item calldata, address[] calldata) external view returns (bool) {
        if (rejects) revert StubPolicy_Rejected();
        return pass;
    }

    function screenRelease(BridgeHookTypes.Item calldata, address[] calldata, uint64, uint256) external returns (bool) {
        if (rejects) revert StubPolicy_Rejected();
        if (reenter) {
            reenter = false;
            // Should fail: the module deletes the record before it calls out.
            module.completeDeposit(reentryItem);
        }
        return pass;
    }

    function clear(bytes32 _id, uint64 _expiresAt) external {
        module.recordVerdict(_id, _expiresAt);
    }

    function revokeOne(bytes32 _id) external {
        module.revokeVerdict(_id);
    }
}

/// @title MockComplianceModule_TestInit
/// @notice Shared setup for the mock compliance module tests.
contract MockComplianceModule_TestInit is Test {
    /// @notice Mirrors of the module's events, which 0.8.15 cannot reference by qualified name.
    event Held(bytes32 indexed id, BridgeHookTypes.Item item);
    event Completed(bytes32 indexed id);

    MockComplianceModule internal module;
    StubPortal internal portal;
    StubPolicy internal policy;

    address internal owner = makeAddr("owner");
    address internal systemConfig = makeAddr("systemConfig");
    address internal messenger = makeAddr("messenger");
    address internal l1Bridge = makeAddr("l1Bridge");

    function setUp() public virtual {
        portal = new StubPortal();
        portal.setSystemConfig(systemConfig);

        module = new MockComplianceModule(IOptimismPortal2(payable(address(portal))), owner);
        portal.setBridgeHook(address(module));

        policy = new StubPolicy();
        policy.setModule(module);
        vm.prank(owner);
        module.setPolicy(IPolicy(address(policy)));

        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l1StandardBridge, ()), abi.encode(l1Bridge));
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(messenger));

        vm.deal(address(portal), 100 ether);
    }

    /// @notice An ETH item with the given nonce.
    function _item(uint256 _nonce) internal pure returns (BridgeHookTypes.Item memory) {
        return BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: address(0xa11ce),
            aliased: false,
            to: address(0xb0b),
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: "",
            nonce: _nonce
        });
    }

    /// @notice Holds an item through the Portal and clears it through the policy.
    function _holdAndClear(BridgeHookTypes.Item memory _i) internal returns (bytes32 id_) {
        portal.callHold{ value: _i.ethAmount }(module, _i);
        id_ = BridgeHookHashing.hashItem(_i);
        policy.clear(id_, uint64(block.timestamp + 1 days));
    }
}

/// @title MockComplianceModule_screenDeposit_Test
/// @notice Tests the screenDeposit function of the MockComplianceModule contract.
contract MockComplianceModule_screenDeposit_Test is MockComplianceModule_TestInit {
    /// @notice Tests that the module answers with whatever the policy decides.
    function test_screenDeposit_followsPolicy_succeeds() public {
        assertTrue(portal.callScreenDeposit(module, _item(0)));

        policy.setPass(false);
        assertFalse(portal.callScreenDeposit(module, _item(0)));
    }

    /// @notice Tests that a policy revert refuses the deposit rather than being swallowed into
    ///         a hold, which would turn a policy fault into a permanent deferral.
    function test_screenDeposit_policyReverts_reverts() public {
        policy.setRejects(true);

        vm.expectRevert(StubPolicy.StubPolicy_Rejected.selector);
        portal.callScreenDeposit(module, _item(0));
    }

    /// @notice Tests that only the Portal may screen.
    function test_screenDeposit_callerNotPortal_reverts() public {
        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPortal.selector);
        module.screenDeposit(_item(0));
    }

    /// @notice Tests that the module refuses once it is no longer the Portal's hook. A removed
    ///         hook becomes an ordinary withdrawal target with the Portal as its caller, so the
    ///         caller check alone is not enough.
    function test_screenDeposit_notCurrentHook_reverts() public {
        portal.setBridgeHook(address(0xdead));

        vm.expectRevert(MockComplianceModule.MockComplianceModule_NotCurrentHook.selector);
        portal.callScreenDeposit(module, _item(0));
    }

    /// @notice Tests that the disable switch lets deposits through without asking the policy,
    ///         so a failing policy cannot block new deposits indefinitely.
    function test_screenDeposit_disabled_succeeds() public {
        policy.setRejects(true);
        vm.prank(owner);
        module.disable(0);

        assertTrue(portal.callScreenDeposit(module, _item(0)));
    }
}

/// @title MockComplianceModule_screenWithdrawal_Test
/// @notice Tests the screenWithdrawal function of the MockComplianceModule contract.
contract MockComplianceModule_screenWithdrawal_Test is MockComplianceModule_TestInit {
    /// @notice Tests that while disabled, withdrawals initiated before the cutoff are still
    ///         screened, because the disable protects new items rather than releasing held ones.
    function test_screenWithdrawal_disabledRespectsCutoff_succeeds() public {
        vm.prank(owner);
        module.disable(100);
        policy.setPass(false);

        // At or above the cutoff: skips the policy entirely.
        assertTrue(portal.callScreenWithdrawal(module, _item(100), 1));
        // Below it: still screened, so the policy's hold stands.
        assertFalse(portal.callScreenWithdrawal(module, _item(99), 1));
    }

    /// @notice Tests that enabling restores screening for everything.
    function test_screenWithdrawal_enabled_succeeds() public {
        vm.startPrank(owner);
        module.disable(0);
        module.enable();
        vm.stopPrank();
        policy.setPass(false);

        assertFalse(portal.callScreenWithdrawal(module, _item(1000), 1));
    }
}

/// @title MockComplianceModule_holdDeposit_Test
/// @notice Tests the holdDeposit function of the MockComplianceModule contract.
contract MockComplianceModule_holdDeposit_Test is MockComplianceModule_TestInit {
    /// @notice Tests that holding records the item and emits a Held whose item hashes back to
    ///         the emitted ID, which is what makes the log a usable preimage.
    function test_holdDeposit_succeeds() public {
        BridgeHookTypes.Item memory item = _item(0);
        bytes32 id = BridgeHookHashing.hashItem(item);

        vm.expectEmit(true, true, true, true, address(module));
        emit Held(id, item);
        portal.callHold{ value: 1 ether }(module, item);

        (uint64 heldAt, uint64 expiresAt) = module.items(id);
        assertEq(heldAt, uint64(block.timestamp));
        assertEq(expiresAt, 0);
        assertEq(address(module).balance, 1 ether);
    }

    /// @notice Tests that the ETH sent must match the item, so the module cannot be left owing
    ///         more than it holds.
    function test_holdDeposit_ethMismatch_reverts() public {
        vm.expectRevert(IComplianceModule.ComplianceModule_ETHAmountMismatch.selector);
        portal.callHold{ value: 0.5 ether }(module, _item(0));
    }

    /// @notice Tests that only the Portal may hand the module custody.
    function test_holdDeposit_callerNotPortal_reverts() public {
        vm.deal(address(this), 1 ether);
        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPortal.selector);
        module.holdDeposit{ value: 1 ether }(_item(0));
    }
}

/// @title MockComplianceModule_recordVerdict_Test
/// @notice Tests the recordVerdict function of the MockComplianceModule contract.
contract MockComplianceModule_recordVerdict_Test is MockComplianceModule_TestInit {
    /// @notice Tests that a verdict can be recorded before the item exists, so the screener can
    ///         pre-clear.
    function test_recordVerdict_beforeItemExists_succeeds() public {
        bytes32 id = keccak256("not yet held");
        policy.clear(id, 4242);

        (uint64 heldAt, uint64 expiresAt) = module.items(id);
        assertEq(heldAt, 0);
        assertEq(expiresAt, 4242);
    }

    /// @notice Tests that only the policy may write a verdict, since that is what keeps
    ///         clearance under the chain's control.
    function test_recordVerdict_callerNotPolicy_reverts() public {
        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPolicy.selector);
        module.recordVerdict(keccak256("item"), 1);
    }

    /// @notice Tests that revoking clears the expiry but leaves the hold in place.
    function test_recordVerdict_revokeKeepsHold_succeeds() public {
        BridgeHookTypes.Item memory item = _item(0);
        bytes32 id = _holdAndClear(item);

        policy.revokeOne(id);

        (uint64 heldAt, uint64 expiresAt) = module.items(id);
        assertGt(heldAt, 0);
        assertEq(expiresAt, 0);
    }

    /// @notice Tests that only the policy may revoke.
    function test_recordVerdict_revokeCallerNotPolicy_reverts() public {
        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPolicy.selector);
        module.revokeVerdict(keccak256("item"));
    }
}

/// @title MockComplianceModule_completeDeposit_Test
/// @notice Tests the completeDeposit function of the MockComplianceModule contract.
contract MockComplianceModule_completeDeposit_Test is MockComplianceModule_TestInit {
    /// @notice Tests that completion is permissionless, clears the record and forwards the
    ///         item's ETH to the Portal.
    function test_completeDeposit_succeeds() public {
        BridgeHookTypes.Item memory item = _item(0);
        bytes32 id = _holdAndClear(item);

        vm.expectEmit(true, true, true, true, address(module));
        emit Completed(id);
        vm.prank(makeAddr("anyone"));
        module.completeDeposit(item);

        (uint64 heldAt, uint64 expiresAt) = module.items(id);
        assertEq(heldAt, 0);
        assertEq(expiresAt, 0);
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that an item that was never held cannot be completed, so the hook cannot
    ///         run terms the Portal never committed.
    function test_completeDeposit_notHeld_reverts() public {
        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(_item(0));
    }

    /// @notice Tests that a held item without a verdict cannot be completed.
    function test_completeDeposit_notCleared_reverts() public {
        BridgeHookTypes.Item memory item = _item(0);
        portal.callHold{ value: 1 ether }(module, item);

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotCleared.selector);
        module.completeDeposit(item);
    }

    /// @notice Tests that the policy gets a second say when value actually moves, and can still
    ///         refuse.
    function test_completeDeposit_policyRefuses_reverts() public {
        BridgeHookTypes.Item memory item = _item(0);
        _holdAndClear(item);
        policy.setPass(false);

        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeDeposit(item);
    }

    /// @notice Tests that a held item completes at most once.
    function test_completeDeposit_twice_reverts() public {
        BridgeHookTypes.Item memory item = _item(0);
        _holdAndClear(item);
        module.completeDeposit(item);

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(item);
    }

    /// @notice Tests that a policy calling back into completion cannot move the same item
    ///         twice, because the record is deleted before the module calls out.
    function test_completeDeposit_policyReentrancy_reverts() public {
        BridgeHookTypes.Item memory item = _item(0);
        _holdAndClear(item);
        policy.armReentry(item);

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(item);
    }
}

/// @title MockComplianceModule_completeWithdrawal_Test
/// @notice Tests the completeWithdrawal function of the MockComplianceModule contract.
contract MockComplianceModule_completeWithdrawal_Test is MockComplianceModule_TestInit {
    /// @notice Tests that a held withdrawal cannot complete while the chain is paused.
    function test_completeWithdrawal_paused_reverts() public {
        BridgeHookTypes.Item memory item = _item(0);
        _holdAndClear(item);
        portal.setPaused(true);

        vm.expectRevert(IComplianceModule.ComplianceModule_Paused.selector);
        module.completeWithdrawal(item);
    }

    /// @notice Tests that it completes once the pause is lifted.
    function test_completeWithdrawal_succeeds() public {
        BridgeHookTypes.Item memory item = _item(0);
        bytes32 id = _holdAndClear(item);

        module.completeWithdrawal(item);

        (uint64 heldAt,) = module.items(id);
        assertEq(heldAt, 0);
    }
}

/// @title MockComplianceModule_setPolicy_Test
/// @notice Tests the owner-gated setters of the MockComplianceModule contract.
contract MockComplianceModule_setPolicy_Test is MockComplianceModule_TestInit {
    /// @notice Tests that swapping the policy leaves records and verdicts untouched, so held
    ///         items survive the change.
    function test_setPolicy_keepsRecords_succeeds() public {
        BridgeHookTypes.Item memory item = _item(0);
        bytes32 id = _holdAndClear(item);

        StubPolicy next = new StubPolicy();
        next.setModule(module);
        vm.prank(owner);
        module.setPolicy(IPolicy(address(next)));

        (uint64 heldAt, uint64 expiresAt) = module.items(id);
        assertGt(heldAt, 0);
        assertGt(expiresAt, 0);
    }

    /// @notice Tests that only the owner may swap the policy.
    function test_setPolicy_callerNotOwner_reverts() public {
        vm.expectRevert(MockComplianceModule.MockComplianceModule_NotOwner.selector);
        module.setPolicy(IPolicy(address(1)));
    }

    /// @notice Tests that only the owner may disable.
    function test_setPolicy_disableCallerNotOwner_reverts() public {
        vm.expectRevert(MockComplianceModule.MockComplianceModule_NotOwner.selector);
        module.disable(0);
    }
}
