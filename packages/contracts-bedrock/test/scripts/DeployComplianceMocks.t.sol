// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { DeployComplianceMocks } from "scripts/deploy/DeployComplianceMocks.s.sol";
import { MockOptimismPortal2 } from "scripts/deploy/mocks/MockOptimismPortal2.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

// Contracts
import { MockPolicy } from "scripts/deploy/mocks/MockPolicy.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @title DeployComplianceMocks_TestInit
/// @notice Shared setup for the compliance mock deployment script tests.
contract DeployComplianceMocks_TestInit is CommonTest {
    DeployComplianceMocks internal deployScript;
    address internal screener = makeAddr("screener");

    function setUp() public virtual override {
        super.setUp();
        deployScript = new DeployComplianceMocks();

        // The script broadcasts, and broadcasting is incompatible with an active prank, so the
        // test contract has to be the ProxyAdmin owner rather than impersonating it.
        vm.prank(proxyAdminOwner);
        proxyAdmin.transferOwnership(address(this));
    }

    /// @notice Builds the script input, broadcast as the ProxyAdmin owner.
    function _input() internal view returns (DeployComplianceMocks.Input memory) {
        return DeployComplianceMocks.Input({
            portal: IOptimismPortal2(payable(address(optimismPortal2))),
            proxyAdmin: IProxyAdmin(address(proxyAdmin)),
            owner: address(this),
            screener: screener,
            useMockPortal: false
        });
    }

    /// @notice Runs the script. The test contract is the ProxyAdmin owner, which every call it
    ///         makes requires.
    function _run() internal returns (DeployComplianceMocks.Output memory) {
        return deployScript.run(_input());
    }
}

/// @title DeployComplianceMocks_run_Test
/// @notice Tests the run function of the DeployComplianceMocks script.
contract DeployComplianceMocks_run_Test is DeployComplianceMocks_TestInit {
    /// @notice Tests that the script points the Portal's hook at the module and wires the two
    ///         mocks to each other.
    function test_run_wiresTheHook_succeeds() public {
        DeployComplianceMocks.Output memory out = _run();

        assertEq(address(optimismPortal2.bridgeHook()), address(out.module));
        assertEq(address(out.module.policy()), address(out.policy));
        assertEq(address(out.module.portal()), address(optimismPortal2));
        assertEq(address(out.policy.module()), address(out.module));
        assertEq(out.policy.screener(), screener);
    }

    /// @notice Tests that the Portal ends on the implementation it started with. The upgrade is
    ///         only the vehicle for re-running the initializer, so swapping in a different
    ///         implementation would be a silent and serious change.
    function test_run_keepsImplementation_succeeds() public {
        address before = proxyAdmin.getProxyImplementation(address(optimismPortal2));

        _run();

        assertEq(proxyAdmin.getProxyImplementation(address(optimismPortal2)), before);
    }

    /// @notice Tests that re-running the initializer changes nothing but the hook. This is what
    ///         makes the rewire safe on a live chain.
    function test_run_initializeIsNoOpOtherwise_succeeds() public {
        address systemConfigBefore = address(optimismPortal2.systemConfig());
        address anchorsBefore = address(optimismPortal2.anchorStateRegistry());
        address lockboxBefore = address(optimismPortal2.ethLockbox());
        uint256 delayBefore = optimismPortal2.proofMaturityDelaySeconds();
        address l2SenderBefore = optimismPortal2.l2Sender();

        _run();

        assertEq(address(optimismPortal2.systemConfig()), systemConfigBefore);
        assertEq(address(optimismPortal2.anchorStateRegistry()), anchorsBefore);
        assertEq(address(optimismPortal2.ethLockbox()), lockboxBefore);
        assertEq(optimismPortal2.proofMaturityDelaySeconds(), delayBefore);
        assertEq(optimismPortal2.l2Sender(), l2SenderBefore);
        assertEq(optimismPortal2.l2Sender(), Constants.DEFAULT_L2_SENDER);
    }

    /// @notice Tests that the initialized byte is restored, so the Portal cannot be
    ///         re-initialized by anyone who happens to call it next.
    function test_run_restoresInitializedByte_succeeds() public {
        bytes32 before = vm.load(address(optimismPortal2), bytes32(0));

        _run();

        assertEq(vm.load(address(optimismPortal2), bytes32(0)), before);
        assertEq(uint256(before) & 0xff, 4);
    }

    /// @notice Tests that an ordinary deposit still works afterwards, which is the practical
    ///         proof that the rewire did not disturb the Portal.
    function test_run_depositStillWorks_succeeds() public {
        _run();

        vm.deal(alice, 1 ether);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: 1 ether }(alice, 0, 100_000, false, "");
    }

    /// @notice Tests that a second run is refused rather than silently replacing a hook that
    ///         may already hold value.
    function test_run_secondRun_reverts() public {
        _run();

        vm.expectRevert("DeployComplianceMocks: hook already set");
        deployScript.run(_input());
    }

    /// @notice Tests that a caller who is not the ProxyAdmin owner is refused up front, rather
    ///         than failing partway through with the mocks already deployed.
    function test_run_callerNotProxyAdminOwner_reverts() public {
        DeployComplianceMocks.Input memory input = _input();
        input.owner = alice;

        vm.prank(alice);
        vm.expectRevert("DeployComplianceMocks: not the ProxyAdmin owner");
        deployScript.run(input);
    }

    /// @notice Tests that the owner must be the broadcaster, since the script wires the mocks
    ///         through owner-gated setters.
    function test_run_ownerNotBroadcaster_reverts() public {
        DeployComplianceMocks.Input memory input = _input();
        input.owner = alice;

        vm.expectRevert("DeployComplianceMocks: owner must be the broadcaster");
        deployScript.run(input);
    }
}

/// @title DeployComplianceMocks_mockPortal_Test
/// @notice Tests installing the mock Portal implementation behind the live proxy.
contract DeployComplianceMocks_mockPortal_Test is DeployComplianceMocks_TestInit {
    function _mockInput() internal view returns (DeployComplianceMocks.Input memory input_) {
        input_ = _input();
        input_.useMockPortal = true;
    }

    /// @notice Tests that the mock reads the live Portal's state through the real Portal's
    ///         storage layout. This is the assertion the whole padding exercise exists for: a
    ///         misaligned slot would surface here as a wrong address rather than a revert.
    function test_mockPortal_preservesLiveState_succeeds() public {
        address systemConfigBefore = address(optimismPortal2.systemConfig());
        address anchorsBefore = address(optimismPortal2.anchorStateRegistry());
        address lockboxBefore = address(optimismPortal2.ethLockbox());
        uint256 delayBefore = optimismPortal2.proofMaturityDelaySeconds();
        address l2SenderBefore = optimismPortal2.l2Sender();
        uint256 minBefore = optimismPortal2.minProofMaturityDelaySeconds();
        uint256 maxBefore = optimismPortal2.maxProofMaturityDelaySeconds();

        DeployComplianceMocks.Output memory out = deployScript.run(_mockInput());

        MockOptimismPortal2 mocked = MockOptimismPortal2(payable(address(optimismPortal2)));
        assertEq(address(mocked.systemConfig()), systemConfigBefore);
        assertEq(address(mocked.anchorStateRegistry()), anchorsBefore);
        assertEq(address(mocked.ethLockbox()), lockboxBefore);
        assertEq(mocked.proofMaturityDelaySeconds(), delayBefore);
        assertEq(mocked.l2Sender(), l2SenderBefore);
        assertEq(address(mocked.bridgeHook()), address(out.module));

        // The immutables come from the new implementation, so they must have been constructed
        // with the values the Portal was already using.
        assertEq(mocked.minProofMaturityDelaySeconds(), minBefore);
        assertEq(mocked.maxProofMaturityDelaySeconds(), maxBefore);
    }

    /// @notice Tests that the proxy really is running the mock, not the original.
    function test_mockPortal_installsMockImplementation_succeeds() public {
        address before = proxyAdmin.getProxyImplementation(address(optimismPortal2));

        DeployComplianceMocks.Output memory out = deployScript.run(_mockInput());

        assertTrue(out.portalImpl != before);
        assertEq(proxyAdmin.getProxyImplementation(address(optimismPortal2)), out.portalImpl);
    }

    /// @notice Tests that a deposit now flows through the hook, which is the point of swapping
    ///         the implementation at all.
    function test_mockPortal_depositIsHeld_succeeds() public {
        DeployComplianceMocks.Output memory out = deployScript.run(_mockInput());

        // Hold every deposit.
        out.policy.setDefaultDeposit(MockPolicy.Outcome.Hold);

        MockOptimismPortal2 mocked = MockOptimismPortal2(payable(address(optimismPortal2)));
        vm.deal(alice, 1 ether);
        vm.prank(alice, alice);
        mocked.depositTransaction{ value: 1 ether }(alice, 0, 100_000, false, "");

        assertEq(address(out.module).balance, 1 ether);
        assertEq(mocked.depositCount(), 1);
    }

    /// @notice Tests that a deposit the policy lets through still reaches L2 as normal.
    function test_mockPortal_depositProceeds_succeeds() public {
        deployScript.run(_mockInput());

        MockOptimismPortal2 mocked = MockOptimismPortal2(payable(address(optimismPortal2)));
        vm.deal(alice, 1 ether);
        vm.prank(alice, alice);
        mocked.depositTransaction{ value: 1 ether }(alice, 0, 100_000, false, "");

        assertEq(mocked.depositCount(), 1);
    }
}
