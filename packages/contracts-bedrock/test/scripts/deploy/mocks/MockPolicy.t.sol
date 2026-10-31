// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test, Vm } from "forge-std/Test.sol";

// Contracts
import { MockPolicy } from "scripts/deploy/mocks/MockPolicy.sol";

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";

/// @title MockPolicy_TestInit
/// @notice Shared setup for the mock policy tests.
contract MockPolicy_TestInit is Test {
    MockPolicy internal policy;

    address internal owner = makeAddr("owner");
    address internal screener = makeAddr("screener");
    address internal module = makeAddr("module");

    bytes32 internal constant OBSERVED_TOPIC = keccak256(
        "ItemObserved(bytes32,(uint8,uint8,address,bool,address,uint256,uint256,uint256,bool,bytes,uint256),uint8,bool)"
    );

    address[] internal noParties;

    function setUp() public virtual {
        policy = new MockPolicy(owner, screener);
        vm.prank(owner);
        policy.setModule(IComplianceModule(module));
    }

    /// @notice An item with the given direction and nonce.
    function _item(
        BridgeHookTypes.Direction _direction,
        uint256 _nonce
    )
        internal
        pure
        returns (BridgeHookTypes.Item memory)
    {
        return BridgeHookTypes.Item({
            direction: _direction,
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

    /// @notice Screens as the module, which is the only caller the policy accepts.
    function _screen(BridgeHookTypes.Item memory _i, address[] memory _p) internal returns (bool) {
        vm.prank(module);
        return policy.screen(_i, _p);
    }

    /// @notice Screens a release as the module.
    function _screenRelease(
        BridgeHookTypes.Item memory _i,
        address[] memory _p,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        internal
        returns (bool)
    {
        vm.prank(module);
        return policy.screenRelease(_i, _p, _expiresAt, _finalizableAt);
    }

    /// @notice Counts the logs carrying the given topic0.
    function _countTopic(Vm.Log[] memory _logs, bytes32 _topic) internal pure returns (uint256 count_) {
        for (uint256 i; i < _logs.length; i++) {
            if (_logs[i].topics.length > 0 && _logs[i].topics[0] == _topic) count_++;
        }
    }
}

/// @title MockPolicy_screen_Test
/// @notice Tests the screen function of the MockPolicy contract.
contract MockPolicy_screen_Test is MockPolicy_TestInit {
    /// @notice Tests that the default answer for a deposit is to proceed, so a freshly deployed
    ///         policy does not silently hold every deposit on the chain.
    function test_screen_defaultProceeds_succeeds() public {
        assertTrue(_screen(_item(BridgeHookTypes.Direction.Deposit, 0), noParties));
    }

    /// @notice Tests that flipping the deposit default holds deposits instead.
    function test_screen_defaultHold_succeeds() public {
        vm.prank(owner);
        policy.setDefaultDeposit(MockPolicy.Outcome.Hold);

        assertFalse(_screen(_item(BridgeHookTypes.Direction.Deposit, 0), noParties));
    }

    /// @notice Tests that a Reject default refuses the item outright, which the Portal turns
    ///         into a reverted deposit that consumes nothing.
    function test_screen_defaultReject_reverts() public {
        vm.prank(owner);
        policy.setDefaultDeposit(MockPolicy.Outcome.Reject);

        BridgeHookTypes.Item memory item = _item(BridgeHookTypes.Direction.Deposit, 0);
        vm.prank(module);
        vm.expectRevert(
            abi.encodeWithSelector(MockPolicy.MockPolicy_Rejected.selector, BridgeHookHashing.hashItem(item))
        );
        policy.screen(item, noParties);
    }

    /// @notice Tests that an override beats the default, which is what lets one deposit be held
    ///         while another proceeds without racing a global toggle between transactions.
    function test_screen_overrideBeatsDefault_succeeds() public {
        BridgeHookTypes.Item memory pinned = _item(BridgeHookTypes.Direction.Deposit, 1);

        vm.startPrank(owner);
        policy.setDefaultDeposit(MockPolicy.Outcome.Proceed);
        policy.setOverride(BridgeHookHashing.hashItem(pinned), MockPolicy.Override.Hold);
        vm.stopPrank();

        assertTrue(_screen(_item(BridgeHookTypes.Direction.Deposit, 0), noParties));
        assertFalse(_screen(pinned, noParties));
    }

    /// @notice Tests that clearing an override returns the item to the default.
    function test_screen_overrideUnset_succeeds() public {
        BridgeHookTypes.Item memory item = _item(BridgeHookTypes.Direction.Deposit, 0);
        bytes32 id = BridgeHookHashing.hashItem(item);

        vm.prank(owner);
        policy.setOverride(id, MockPolicy.Override.Hold);
        assertFalse(_screen(item, noParties));

        vm.prank(owner);
        policy.setOverride(id, MockPolicy.Override.Unset);
        assertTrue(_screen(item, noParties));
    }

    /// @notice Tests that only the module may screen. Anyone else could otherwise emit an
    ///         ItemObserved for an item the Portal never saw, and an indexer that treats the
    ///         feed as authoritative would ingest it.
    function test_screen_callerNotModule_reverts() public {
        vm.expectRevert(MockPolicy.MockPolicy_NotModule.selector);
        policy.screen(_item(BridgeHookTypes.Direction.Deposit, 0), noParties);
    }

    /// @notice Tests that the same guard covers the release path.
    function test_screen_releaseCallerNotModule_reverts() public {
        vm.expectRevert(MockPolicy.MockPolicy_NotModule.selector);
        policy.screenRelease(_item(BridgeHookTypes.Direction.Withdrawal, 0), noParties, 0, 1000);
    }

    /// @notice Tests that every screened item is reported, so the screener can recover the IDs
    ///         of deposits that proceeded and therefore left no other on-chain trace.
    function test_screen_emitsItemObserved_succeeds() public {
        vm.recordLogs();
        _screen(_item(BridgeHookTypes.Direction.Deposit, 0), noParties);

        assertEq(_countTopic(vm.getRecordedLogs(), OBSERVED_TOPIC), 1);
    }
}

/// @title MockPolicy_screenRelease_Test
/// @notice Tests the screenRelease function of the MockPolicy contract.
contract MockPolicy_screenRelease_Test is MockPolicy_TestInit {
    /// @notice Tests that a non-zero finalizable time is read as a withdrawal arriving, so the
    ///         withdrawal default applies rather than the completion default.
    function test_screenRelease_withdrawalUsesWithdrawalDefault_succeeds() public {
        vm.startPrank(owner);
        policy.setDefaultWithdrawal(MockPolicy.Outcome.Hold);
        policy.setDefaultRelease(MockPolicy.Outcome.Proceed);
        vm.stopPrank();

        assertFalse(_screenRelease(_item(BridgeHookTypes.Direction.Withdrawal, 0), noParties, 0, 1000));
    }

    /// @notice Tests that a zero finalizable time is read as a completion, which is how the
    ///         module distinguishes the two callers of this function.
    function test_screenRelease_completionUsesReleaseDefault_succeeds() public {
        vm.startPrank(owner);
        policy.setDefaultWithdrawal(MockPolicy.Outcome.Hold);
        policy.setDefaultRelease(MockPolicy.Outcome.Proceed);
        vm.stopPrank();

        assertTrue(_screenRelease(_item(BridgeHookTypes.Direction.Withdrawal, 0), noParties, 0, 0));
    }

    /// @notice Tests that an expired clearance counts as none, so the item is held like any
    ///         other uncleared one rather than proceeding.
    function test_screenRelease_expiredClearance_succeeds() public {
        vm.warp(2000);
        BridgeHookTypes.Item memory item = _item(BridgeHookTypes.Direction.Withdrawal, 0);

        assertTrue(_screenRelease(item, noParties, uint64(block.timestamp + 1), 1000));
        assertFalse(_screenRelease(item, noParties, uint64(block.timestamp - 1), 1000));
    }

    /// @notice Tests that a zero expiry means the clearance never ages out.
    function test_screenRelease_zeroExpiryNeverExpires_succeeds() public {
        vm.warp(2000);
        assertTrue(_screenRelease(_item(BridgeHookTypes.Direction.Withdrawal, 0), noParties, 0, 1000));
    }
}

/// @title MockPolicy_clear_Test
/// @notice Tests the clear function of the MockPolicy contract.
contract MockPolicy_clear_Test is MockPolicy_TestInit {
    /// @notice Tests that the screener's clearance reaches the module carrying the expiry the
    ///         screener chose, since expiry is decided entirely off chain.
    function test_clear_forwardsExpiryToModule_succeeds() public {
        bytes32 id = keccak256("item");
        vm.mockCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, 4242)), "");
        vm.expectCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, 4242)));

        vm.prank(screener);
        policy.clear(id, 4242);
    }

    /// @notice Tests that a zero expiry is refused. Zero is the module record's sentinel for
    ///         carrying no verdict, so recording it would emit VerdictRecorded for a clearance
    ///         that leaves the item uncleared.
    function test_clear_zeroExpiry_reverts() public {
        vm.warp(1000);

        vm.prank(screener);
        vm.expectRevert(MockPolicy.MockPolicy_InvalidExpiry.selector);
        policy.clear(keccak256("item"), 0);
    }

    /// @notice Tests that an already-passed expiry is refused, since the policy would treat it
    ///         as expired the moment it read it back.
    function test_clear_pastExpiry_reverts() public {
        vm.warp(1000);

        vm.prank(screener);
        vm.expectRevert(MockPolicy.MockPolicy_InvalidExpiry.selector);
        policy.clear(keccak256("item"), 999);
    }

    /// @notice Tests that an expiry landing exactly on the current timestamp is accepted, since
    ///         a clearance only counts as expired once the timestamp is past it.
    function test_clear_expiryAtNow_succeeds() public {
        vm.warp(1000);
        bytes32 id = keccak256("item");
        vm.mockCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, 1000)), "");
        vm.expectCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, 1000)));

        vm.prank(screener);
        policy.clear(id, 1000);
    }

    /// @notice Tests that only the screener may clear, since the policy is the only address the
    ///         module accepts verdicts from.
    function test_clear_callerNotScreener_reverts() public {
        vm.expectRevert(MockPolicy.MockPolicy_NotScreener.selector);
        policy.clear(keccak256("item"), 1);
    }

    /// @notice Tests that revoking reaches the module.
    function test_clear_revokeForwardsToModule_succeeds() public {
        bytes32 id = keccak256("item");
        vm.mockCall(module, abi.encodeCall(IComplianceModule.revokeVerdict, (id)), "");
        vm.expectCall(module, abi.encodeCall(IComplianceModule.revokeVerdict, (id)));

        vm.prank(screener);
        policy.revoke(id);
    }

    /// @notice Tests that a verdict cannot be written before the module is wired, rather than
    ///         silently calling the zero address.
    function test_clear_moduleNotSet_reverts() public {
        MockPolicy fresh = new MockPolicy(owner, screener);

        vm.prank(screener);
        vm.expectRevert(MockPolicy.MockPolicy_ModuleNotSet.selector);
        fresh.clear(keccak256("item"), 1);
    }
}

/// @title MockPolicy_setDefaultDeposit_Test
/// @notice Tests the owner-gated setters of the MockPolicy contract.
contract MockPolicy_setDefaultDeposit_Test is MockPolicy_TestInit {
    /// @notice Tests that only the owner may change a default.
    function test_setDefaultDeposit_callerNotOwner_reverts() public {
        vm.expectRevert(MockPolicy.MockPolicy_NotOwner.selector);
        policy.setDefaultDeposit(MockPolicy.Outcome.Hold);
    }

    /// @notice Tests that only the owner may pin an item.
    function test_setDefaultDeposit_overrideCallerNotOwner_reverts() public {
        vm.expectRevert(MockPolicy.MockPolicy_NotOwner.selector);
        policy.setOverride(keccak256("item"), MockPolicy.Override.Hold);
    }

    /// @notice Tests that only the owner may rotate the screener.
    function test_setDefaultDeposit_screenerCallerNotOwner_reverts() public {
        vm.expectRevert(MockPolicy.MockPolicy_NotOwner.selector);
        policy.setScreener(address(1));
    }
}
