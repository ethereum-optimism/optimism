// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { VmSafe } from "forge-std/Vm.sol";
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Libraries
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";

/// @title VisionPolicyV1_TestInit
/// @notice Deploys the policy with mocked module and Portal calls.
abstract contract VisionPolicyV1_TestInit is Test {
    event AllowlistUpdated(address indexed account, bool allowlisted);
    event DenylistUpdated(address indexed account, bool denylisted);
    event VerdictWriterSet(address indexed previousVerdictWriter, address indexed newVerdictWriter);
    event MaxVerdictValiditySet(uint64 maxVerdictValidity);
    event HoldDelaySet(uint64 holdDelay);
    event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /// @notice Fixed expected limits. These tests must detect changes to the policy's caps.
    uint64 internal constant MAX_VERDICT_VALIDITY_CAP = 5 days;
    uint64 internal constant HOLD_DELAY_CAP = 12 hours;

    address internal module = makeAddr("module");
    address internal portal = makeAddr("portal");
    address internal owner = makeAddr("owner");
    address internal verdictWriter = makeAddr("verdictWriter");
    address internal allowlistedSender = makeAddr("allowlistedSender");
    address internal deniedParty = makeAddr("deniedParty");
    address internal sender = makeAddr("sender");
    address internal recipient = makeAddr("recipient");

    IVisionPolicyV1 internal policy;

    function setUp() public virtual {
        vm.mockCall(module, abi.encodeCall(IComplianceModule.portal, ()), abi.encode(portal));
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.proofMaturityDelaySeconds, ()), abi.encode(7 days));

        address[] memory allowlist = new address[](1);
        allowlist[0] = allowlistedSender;
        address[] memory denylist = new address[](1);
        denylist[0] = deniedParty;

        policy = IVisionPolicyV1(
            DeployUtils.create1({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IVisionPolicyV1.__constructor__,
                        (IComplianceModule(module), owner, verdictWriter, 3 days, 1 hours, allowlist, denylist)
                    )
                )
            })
        );
    }
}

/// @title VisionPolicyV1_Constructor_Test
/// @notice Tests the constructor of the `VisionPolicyV1` contract.
contract VisionPolicyV1_Constructor_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that the constructor sets the state and emits the setters' events.
    function test_constructor_succeeds() external {
        address[] memory allowlist = new address[](2);
        allowlist[0] = allowlistedSender;
        allowlist[1] = sender;
        address[] memory denylist = new address[](2);
        denylist[0] = deniedParty;
        denylist[1] = recipient;
        address expected = vm.computeCreateAddress(address(this), vm.getNonce(address(this)));

        vm.expectEmit(expected);
        emit OwnershipTransferred(address(0), owner);
        vm.expectEmit(expected);
        emit VerdictWriterSet(address(0), verdictWriter);
        vm.expectEmit(expected);
        emit MaxVerdictValiditySet(3 days);
        vm.expectEmit(expected);
        emit HoldDelaySet(1 hours);
        vm.expectEmit(expected);
        emit AllowlistUpdated(allowlistedSender, true);
        vm.expectEmit(expected);
        emit AllowlistUpdated(sender, true);
        vm.expectEmit(expected);
        emit DenylistUpdated(deniedParty, true);
        vm.expectEmit(expected);
        emit DenylistUpdated(recipient, true);

        IVisionPolicyV1 deployed = IVisionPolicyV1(
            DeployUtils.create1({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IVisionPolicyV1.__constructor__,
                        (IComplianceModule(module), owner, verdictWriter, 3 days, 1 hours, allowlist, denylist)
                    )
                )
            })
        );

        assertEq(address(deployed), expected);
        assertEq(address(deployed.module()), module);
        assertEq(deployed.owner(), owner);
        assertEq(deployed.verdictWriter(), verdictWriter);
        assertEq(deployed.maxVerdictValidity(), 3 days);
        assertEq(deployed.holdDelay(), 1 hours);
        assertTrue(deployed.allowlisted(allowlistedSender), "allowlistedSender");
        assertTrue(deployed.allowlisted(sender), "sender");
        assertTrue(deployed.denylisted(deniedParty), "deniedParty");
        assertTrue(deployed.denylisted(recipient), "recipient");
    }

    /// @notice Tests that the constructor reverts for a zero module, owner or verdict writer.
    function test_constructor_zeroAddress_reverts() external {
        address[3][3] memory args =
            [[address(0), owner, verdictWriter], [module, address(0), verdictWriter], [module, owner, address(0)]];
        bytes[3] memory errors = [
            abi.encodePacked(IVisionPolicyV1.VisionPolicyV1_ZeroAddress.selector),
            bytes.concat(IVisionPolicyV1.OwnableInvalidOwner.selector, abi.encode(address(0))),
            abi.encodePacked(IVisionPolicyV1.VisionPolicyV1_ZeroAddress.selector)
        ];
        for (uint256 i; i < args.length; i++) {
            vm.expectRevert(errors[i]);
            DeployUtils.create1({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IVisionPolicyV1.__constructor__,
                        (
                            IComplianceModule(args[i][0]),
                            args[i][1],
                            args[i][2],
                            3 days,
                            1 hours,
                            new address[](0),
                            new address[](0)
                        )
                    )
                )
            });
        }
    }

    /// @notice Tests that the constructor applies the setters' bounds to the parameters.
    function test_constructor_parameterOutOfBounds_reverts() external {
        uint64[2][3] memory args = [
            [MAX_VERDICT_VALIDITY_CAP + 1, uint64(1 hours)],
            [uint64(3 days), HOLD_DELAY_CAP + 1],
            [uint64(3 days), uint64(7 days)]
        ];
        bytes[3] memory errors = [
            abi.encodePacked(IVisionPolicyV1.VisionPolicyV1_OutOfBounds.selector),
            abi.encodePacked(IVisionPolicyV1.VisionPolicyV1_OutOfBounds.selector),
            bytes.concat(IVisionPolicyV1.VisionPolicyV1_HoldDelayTooLong.selector, abi.encode(uint64(7 days)))
        ];
        for (uint256 i; i < args.length; i++) {
            vm.expectRevert(errors[i]);
            DeployUtils.create1({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IVisionPolicyV1.__constructor__,
                        (
                            IComplianceModule(module),
                            owner,
                            verdictWriter,
                            args[i][0],
                            args[i][1],
                            new address[](0),
                            new address[](0)
                        )
                    )
                )
            });
        }
    }
}

/// @title VisionPolicyV1_RenounceOwnership_Test
/// @notice Tests the `renounceOwnership` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_RenounceOwnership_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that renounceOwnership reverts for the owner.
    function test_renounceOwnership_owner_reverts() external {
        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_OwnershipNotRenounceable.selector);
        vm.prank(owner);
        policy.renounceOwnership();
    }
}

/// @title VisionPolicyV1_TransferOwnership_Test
/// @notice Tests the `transferOwnership` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_TransferOwnership_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that the owner role moves to the new owner only when it accepts.
    function test_transferOwnership_succeeds() external {
        address newOwner = makeAddr("newOwner");

        vm.expectEmit(address(policy));
        emit OwnershipTransferStarted(owner, newOwner);
        vm.prank(owner);
        policy.transferOwnership(newOwner);

        assertEq(policy.owner(), owner);
        assertEq(policy.pendingOwner(), newOwner);
        vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(newOwner)));
        vm.prank(newOwner);
        policy.setHoldDelay(0);

        vm.expectEmit(address(policy));
        emit OwnershipTransferred(owner, newOwner);
        vm.prank(newOwner);
        policy.acceptOwnership();

        assertEq(policy.owner(), newOwner);
        vm.prank(newOwner);
        policy.setHoldDelay(0);
        assertEq(policy.holdDelay(), 0);
        vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(owner)));
        vm.prank(owner);
        policy.setHoldDelay(0);
    }
}

/// @title VisionPolicyV1_ScreenDeposit_Test
/// @notice Tests the `screenDeposit` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_ScreenDeposit_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that caller authorization takes priority over party count validation.
    function testFuzz_screenDeposit_notModule_reverts(address _caller) external {
        vm.assume(_caller != module);
        BridgeHookTypes.Item memory item;

        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
        vm.prank(_caller);
        policy.screenDeposit(item, new address[](0));
    }

    /// @notice Tests that screenDeposit reverts unless there are two parties.
    function testFuzz_screenDeposit_invalidPartiesLength_reverts(uint8 _length) external {
        vm.assume(_length != 2);
        BridgeHookTypes.Item memory item;

        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_InvalidPartiesLength.selector);
        vm.prank(module);
        policy.screenDeposit(item, new address[](_length));
    }

    /// @notice Tests that a deny list entry takes priority over an allowlisted sender.
    function test_screenDeposit_deniedParty_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory allowlist = new address[](1);
        allowlist[0] = deniedParty;
        vm.prank(owner);
        policy.addToAllowlist(allowlist);

        address[] memory deniedSender = new address[](2);
        deniedSender[0] = deniedParty;
        deniedSender[1] = recipient;
        address[] memory deniedRecipient = new address[](2);
        deniedRecipient[0] = allowlistedSender;
        deniedRecipient[1] = deniedParty;

        vm.startPrank(module);
        assertFalse(policy.screenDeposit(item, deniedSender), "denied sender");
        assertFalse(policy.screenDeposit(item, deniedRecipient), "denied recipient");
    }

    /// @notice Tests that an allowlisted sender gives true.
    function test_screenDeposit_allowlistedSender_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[0] = allowlistedSender;
        parties[1] = recipient;

        vm.prank(module);
        assertTrue(policy.screenDeposit(item, parties));
    }

    /// @notice Tests that a sender off the allowlist gives false, also with an allowlisted
    ///         recipient.
    function test_screenDeposit_notAllowlisted_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory unlisted = new address[](2);
        unlisted[0] = sender;
        unlisted[1] = recipient;
        address[] memory allowlistedRecipient = new address[](2);
        allowlistedRecipient[0] = sender;
        allowlistedRecipient[1] = allowlistedSender;

        vm.startPrank(module);
        assertFalse(policy.screenDeposit(item, unlisted), "unlisted");
        assertFalse(policy.screenDeposit(item, allowlistedRecipient), "allowlisted recipient");
    }

    /// @notice Tests that the Fee Vault exemption does not apply to deposits.
    function test_screenDeposit_feeVault_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[4] memory vaults = [
            Predeploys.SEQUENCER_FEE_WALLET,
            Predeploys.BASE_FEE_VAULT,
            Predeploys.L1_FEE_VAULT,
            Predeploys.OPERATOR_FEE_VAULT
        ];
        address[] memory parties = new address[](2);
        parties[1] = recipient;

        vm.startPrank(module);
        for (uint256 i; i < vaults.length; i++) {
            parties[0] = vaults[i];
            assertFalse(policy.screenDeposit(item, parties), vm.toString(vaults[i]));
        }
    }

    /// @notice Tests that screenDeposit gives false when either party is denied.
    function testFuzz_screenDeposit_anyDeniedParty_succeeds(
        uint256 _otherIndex,
        bool _deniedRecipient,
        bool _deniedAllowlisted
    )
        external
    {
        BridgeHookTypes.Item memory item;
        address[4] memory accounts = [sender, allowlistedSender, deniedParty, Predeploys.BASE_FEE_VAULT];
        address other = accounts[bound(_otherIndex, 0, 3)];
        address[] memory parties = new address[](2);
        (parties[0], parties[1]) = _deniedRecipient ? (other, deniedParty) : (deniedParty, other);
        if (_deniedAllowlisted) {
            address[] memory allowlist = new address[](1);
            allowlist[0] = deniedParty;
            vm.prank(owner);
            policy.addToAllowlist(allowlist);
        }

        vm.prank(module);
        assertFalse(policy.screenDeposit(item, parties));
    }

    /// @notice Tests that screenDeposit never reverts for the module with two parties.
    function testFuzz_screenDeposit_twoParties_succeeds(
        BridgeHookTypes.Item memory _item,
        address _sender,
        address _recipient,
        bool _allowlistSender,
        bool _denySender,
        bool _denyRecipient
    )
        external
    {
        address[] memory accounts = new address[](1);
        vm.startPrank(owner);
        accounts[0] = _sender;
        if (_allowlistSender) policy.addToAllowlist(accounts);
        if (_denySender) policy.addToDenylist(accounts);
        accounts[0] = _recipient;
        if (_denyRecipient) policy.addToDenylist(accounts);
        vm.stopPrank();
        address[] memory parties = new address[](2);
        parties[0] = _sender;
        parties[1] = _recipient;

        vm.prank(module);
        policy.screenDeposit(_item, parties);
    }
}

/// @title VisionPolicyV1_ScreenWithdrawal_Test
/// @notice Tests the `screenWithdrawal` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_ScreenWithdrawal_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that caller authorization takes priority over party count validation.
    function testFuzz_screenWithdrawal_notModule_reverts(address _caller) external {
        vm.assume(_caller != module);
        BridgeHookTypes.Item memory item;

        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
        vm.prank(_caller);
        policy.screenWithdrawal(item, new address[](0), 0, block.timestamp);
    }

    /// @notice Tests that the deny list takes priority over sender exemptions and valid verdicts.
    ///         A denied withdrawal returns false even before the Holdable Time.
    function test_screenWithdrawal_deniedParty_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory allowlist = new address[](1);
        allowlist[0] = deniedParty;
        vm.prank(owner);
        policy.addToAllowlist(allowlist);
        address[] memory parties = new address[](2);
        uint64 live = uint64(block.timestamp + 1);

        vm.startPrank(module);
        (parties[0], parties[1]) = (deniedParty, recipient);
        assertFalse(policy.screenWithdrawal(item, parties, live, block.timestamp), "denied sender");
        (parties[0], parties[1]) = (allowlistedSender, deniedParty);
        assertFalse(policy.screenWithdrawal(item, parties, 0, block.timestamp), "allowlisted sender");
        (parties[0], parties[1]) = (Predeploys.BASE_FEE_VAULT, deniedParty);
        assertFalse(policy.screenWithdrawal(item, parties, 0, block.timestamp), "Fee Vault sender");
        (parties[0], parties[1]) = (sender, deniedParty);
        assertFalse(policy.screenWithdrawal(item, parties, live, block.timestamp), "live verdict");
        assertFalse(policy.screenWithdrawal(item, parties, 0, block.timestamp), "before Holdable Time");
    }

    /// @notice Tests that an allowlisted sender gives true before the Holdable Time, with no
    ///         verdict or an expired one.
    function test_screenWithdrawal_allowlistedSender_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[0] = allowlistedSender;
        parties[1] = recipient;

        vm.startPrank(module);
        assertTrue(policy.screenWithdrawal(item, parties, 0, block.timestamp), "no verdict");
        assertTrue(policy.screenWithdrawal(item, parties, uint64(block.timestamp), block.timestamp), "expired");
    }

    /// @notice Tests that each Fee Vault sender gives true before the Holdable Time, with no
    ///         verdict or an expired one.
    function test_screenWithdrawal_feeVault_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[4] memory vaults = [
            Predeploys.SEQUENCER_FEE_WALLET,
            Predeploys.BASE_FEE_VAULT,
            Predeploys.L1_FEE_VAULT,
            Predeploys.OPERATOR_FEE_VAULT
        ];
        address[] memory parties = new address[](2);
        parties[1] = recipient;

        vm.startPrank(module);
        for (uint256 i; i < vaults.length; i++) {
            parties[0] = vaults[i];
            assertTrue(policy.screenWithdrawal(item, parties, 0, block.timestamp), vm.toString(vaults[i]));
            assertTrue(
                policy.screenWithdrawal(item, parties, uint64(block.timestamp), block.timestamp),
                string.concat(vm.toString(vaults[i]), " expired")
            );
        }
    }

    /// @notice Tests that a verdict permits withdrawal one second before expiry.
    ///         At expiry, screening returns false even before the Holdable Time.
    function test_screenWithdrawal_verdictExpiry_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[0] = sender;
        parties[1] = recipient;

        vm.startPrank(module);
        assertTrue(policy.screenWithdrawal(item, parties, uint64(block.timestamp + 1), block.timestamp), "live");
        assertFalse(policy.screenWithdrawal(item, parties, uint64(block.timestamp), block.timestamp), "expired");
    }

    /// @notice Tests that a withdrawal with no verdict reverts before the Holdable Time and gives
    ///         false at it.
    function testFuzz_screenWithdrawal_holdableTime_succeeds(uint256 _finalizableAt, uint64 _holdDelay) external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[0] = sender;
        parties[1] = recipient;
        _holdDelay = uint64(bound(_holdDelay, 0, HOLD_DELAY_CAP));
        // Reserve room for any permitted hold delay so overflow cannot mask the screening rule.
        // Keep holdableAt above zero so the test can check one second before it.
        _finalizableAt = bound(_finalizableAt, 1, type(uint256).max - HOLD_DELAY_CAP);
        vm.prank(owner);
        policy.setHoldDelay(_holdDelay);
        uint256 holdableAt = _finalizableAt + _holdDelay;

        vm.startPrank(module);
        vm.warp(holdableAt - 1);
        vm.expectRevert(bytes.concat(IVisionPolicyV1.VisionPolicyV1_NotYetHoldable.selector, abi.encode(holdableAt)));
        policy.screenWithdrawal(item, parties, 0, _finalizableAt);
        vm.warp(holdableAt);
        assertFalse(policy.screenWithdrawal(item, parties, 0, _finalizableAt));
    }

    /// @notice Tests that a zero holdDelay holds a withdrawal with no verdict at its Finalizable
    ///         Time.
    function test_screenWithdrawal_zeroHoldDelay_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[0] = sender;
        parties[1] = recipient;
        vm.prank(owner);
        policy.setHoldDelay(0);

        vm.prank(module);
        assertFalse(policy.screenWithdrawal(item, parties, 0, block.timestamp));
    }

    /// @notice Tests that screenWithdrawal gives false when either party is denied.
    function testFuzz_screenWithdrawal_anyDeniedParty_succeeds(
        uint256 _otherIndex,
        bool _deniedRecipient,
        bool _deniedAllowlisted,
        uint64 _expiresAt,
        uint256 _finalizableAt,
        uint64 _holdDelay,
        uint256 _timestamp
    )
        external
    {
        BridgeHookTypes.Item memory item;
        address[4] memory accounts = [sender, allowlistedSender, deniedParty, Predeploys.BASE_FEE_VAULT];
        address other = accounts[bound(_otherIndex, 0, 3)];
        address[] memory parties = new address[](2);
        (parties[0], parties[1]) = _deniedRecipient ? (other, deniedParty) : (deniedParty, other);
        vm.startPrank(owner);
        if (_deniedAllowlisted) {
            address[] memory allowlist = new address[](1);
            allowlist[0] = deniedParty;
            policy.addToAllowlist(allowlist);
        }
        policy.setHoldDelay(uint64(bound(_holdDelay, 0, HOLD_DELAY_CAP)));
        vm.stopPrank();
        vm.warp(_timestamp);

        vm.prank(module);
        assertFalse(policy.screenWithdrawal(item, parties, _expiresAt, _finalizableAt));
    }

    /// @notice Tests that screenWithdrawal gives true without a bypass only for a live verdict.
    function testFuzz_screenWithdrawal_passWithoutBypass_succeeds(
        uint256 _senderIndex,
        uint256 _recipientIndex,
        uint64 _expiresAt,
        uint256 _finalizableAt,
        uint64 _holdDelay,
        uint256 _timestamp
    )
        external
    {
        BridgeHookTypes.Item memory item;
        address[4] memory accounts = [sender, allowlistedSender, deniedParty, Predeploys.BASE_FEE_VAULT];
        address[] memory parties = new address[](2);
        parties[0] = accounts[bound(_senderIndex, 0, 3)];
        parties[1] = accounts[bound(_recipientIndex, 0, 3)];
        // Reserve room for any permitted hold delay so overflow cannot mask the screening rule.
        _finalizableAt = bound(_finalizableAt, 0, type(uint256).max - HOLD_DELAY_CAP);
        vm.prank(owner);
        policy.setHoldDelay(uint64(bound(_holdDelay, 0, HOLD_DELAY_CAP)));
        vm.warp(_timestamp);

        vm.prank(module);
        try policy.screenWithdrawal(item, parties, _expiresAt, _finalizableAt) returns (bool pass_) {
            if (pass_ && parties[0] != allowlistedSender && parties[0] != Predeploys.BASE_FEE_VAULT) {
                assertGt(_expiresAt, block.timestamp);
            }
        } catch { }
    }

    /// @notice Tests that screenWithdrawal reverts only with NotYetHoldable, for no verdict
    ///         before the Holdable Time.
    function testFuzz_screenWithdrawal_revertsOnlyBeforeHoldableTime_succeeds(
        uint256 _senderIndex,
        uint256 _recipientIndex,
        uint64 _expiresAt,
        uint256 _finalizableAt,
        uint64 _holdDelay,
        uint256 _timestamp
    )
        external
    {
        BridgeHookTypes.Item memory item;
        address[4] memory accounts = [sender, allowlistedSender, deniedParty, Predeploys.BASE_FEE_VAULT];
        address[] memory parties = new address[](2);
        parties[0] = accounts[bound(_senderIndex, 0, 3)];
        parties[1] = accounts[bound(_recipientIndex, 0, 3)];
        _holdDelay = uint64(bound(_holdDelay, 0, HOLD_DELAY_CAP));
        // Reserve room for any permitted hold delay so overflow cannot mask the screening rule.
        _finalizableAt = bound(_finalizableAt, 0, type(uint256).max - HOLD_DELAY_CAP);
        vm.prank(owner);
        policy.setHoldDelay(_holdDelay);
        vm.warp(_timestamp);

        uint256 holdableAt = _finalizableAt + _holdDelay;
        vm.prank(module);
        try policy.screenWithdrawal(item, parties, _expiresAt, _finalizableAt) returns (bool) { }
        catch (bytes memory reason) {
            assertEq(
                reason, bytes.concat(IVisionPolicyV1.VisionPolicyV1_NotYetHoldable.selector, abi.encode(holdableAt))
            );
            assertEq(_expiresAt, 0, "verdict");
            assertLt(block.timestamp, holdableAt, "holdable");
        }
    }
}

/// @title VisionPolicyV1_ScreenRelease_Test
/// @notice Tests the `screenRelease` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_ScreenRelease_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that caller authorization takes priority over party count validation.
    function testFuzz_screenRelease_notModule_reverts(address _caller) external {
        vm.assume(_caller != module);
        BridgeHookTypes.Item memory item;

        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
        vm.prank(_caller);
        policy.screenRelease(item, new address[](0), uint64(block.timestamp + 1));
    }

    /// @notice Tests that a deny list entry prevents release even with a valid verdict.
    function test_screenRelease_deniedParty_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory deniedSender = new address[](2);
        deniedSender[0] = deniedParty;
        deniedSender[1] = recipient;
        address[] memory deniedRecipient = new address[](2);
        deniedRecipient[0] = sender;
        deniedRecipient[1] = deniedParty;
        uint64 live = uint64(block.timestamp + 1);

        vm.startPrank(module);
        assertFalse(policy.screenRelease(item, deniedSender, live), "denied sender");
        assertFalse(policy.screenRelease(item, deniedRecipient, live), "denied recipient");
    }

    /// @notice Tests that release is permitted one second before verdict expiry and denied at
    ///         expiry.
    function test_screenRelease_verdictExpiry_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[0] = sender;
        parties[1] = recipient;

        vm.startPrank(module);
        assertTrue(policy.screenRelease(item, parties, uint64(block.timestamp + 1)), "live");
        assertFalse(policy.screenRelease(item, parties, uint64(block.timestamp)), "expired");
    }

    /// @notice Tests that an allowlisted or Fee Vault sender with an expired verdict gives false.
    function test_screenRelease_bypassSender_succeeds() external {
        BridgeHookTypes.Item memory item;
        address[] memory parties = new address[](2);
        parties[1] = recipient;
        uint64 expired = uint64(block.timestamp);

        vm.startPrank(module);
        parties[0] = allowlistedSender;
        assertFalse(policy.screenRelease(item, parties, expired), "allowlisted sender");
        parties[0] = Predeploys.BASE_FEE_VAULT;
        assertFalse(policy.screenRelease(item, parties, expired), "Fee Vault sender");
    }

    /// @notice Tests that screenRelease gives false when either party is denied.
    function testFuzz_screenRelease_anyDeniedParty_succeeds(
        uint256 _otherIndex,
        bool _deniedRecipient,
        bool _deniedAllowlisted,
        uint64 _expiresAt,
        uint256 _timestamp
    )
        external
    {
        BridgeHookTypes.Item memory item;
        address[4] memory accounts = [sender, allowlistedSender, deniedParty, Predeploys.BASE_FEE_VAULT];
        address other = accounts[bound(_otherIndex, 0, 3)];
        address[] memory parties = new address[](2);
        (parties[0], parties[1]) = _deniedRecipient ? (other, deniedParty) : (deniedParty, other);
        if (_deniedAllowlisted) {
            address[] memory allowlist = new address[](1);
            allowlist[0] = deniedParty;
            vm.prank(owner);
            policy.addToAllowlist(allowlist);
        }
        vm.warp(_timestamp);

        vm.prank(module);
        assertFalse(policy.screenRelease(item, parties, _expiresAt));
    }

    /// @notice Tests that screenRelease gives true only for a live verdict, whatever the sender.
    function testFuzz_screenRelease_passOnlyWithVerdict_succeeds(
        uint256 _senderIndex,
        uint256 _recipientIndex,
        uint64 _expiresAt,
        uint256 _timestamp
    )
        external
    {
        BridgeHookTypes.Item memory item;
        address[4] memory accounts = [sender, allowlistedSender, deniedParty, Predeploys.BASE_FEE_VAULT];
        address[] memory parties = new address[](2);
        parties[0] = accounts[bound(_senderIndex, 0, 3)];
        parties[1] = accounts[bound(_recipientIndex, 0, 3)];
        vm.warp(_timestamp);

        vm.prank(module);
        if (policy.screenRelease(item, parties, _expiresAt)) assertGt(_expiresAt, block.timestamp);
    }
}

/// @title VisionPolicyV1_Clear_Test
/// @notice Tests the `clear` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_Clear_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that clear reverts for a caller other than the verdict writer.
    function testFuzz_clear_notVerdictWriter_reverts(address _caller) external {
        vm.assume(_caller != verdictWriter);
        address[2] memory callers = [_caller, owner];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
            vm.prank(callers[i]);
            policy.clear(keccak256("item"), uint64(block.timestamp + 1));
        }
    }

    /// @notice Tests that clear calls recordVerdict at both permitted expiry limits:
    ///         one second after the current timestamp and maxVerdictValidity seconds after it.
    function testFuzz_clear_expiryBounds_succeeds(uint64 _maxVerdictValidity) external {
        _maxVerdictValidity = uint64(bound(_maxVerdictValidity, 1, MAX_VERDICT_VALIDITY_CAP));
        vm.prank(owner);
        policy.setMaxVerdictValidity(_maxVerdictValidity);
        bytes32 id = keccak256("item");
        uint64 earliest = uint64(block.timestamp + 1);
        uint64 latest = uint64(block.timestamp + _maxVerdictValidity);

        vm.expectCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, earliest)));
        vm.expectCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, latest)));
        vm.startPrank(verdictWriter);
        policy.clear(id, earliest);
        policy.clear(id, latest);
    }

    /// @notice Tests that clear rejects expiry at or before the current timestamp, or more than
    ///         maxVerdictValidity seconds after it. Covers zero, one day, and the validity cap.
    function test_clear_invalidExpiry_reverts() external {
        uint64[3] memory validities = [0, 1 days, MAX_VERDICT_VALIDITY_CAP];
        for (uint256 i; i < validities.length; i++) {
            vm.prank(owner);
            policy.setMaxVerdictValidity(validities[i]);
            uint64[3] memory expiries =
                [uint64(block.timestamp), uint64(block.timestamp - 1), uint64(block.timestamp + validities[i] + 1)];
            for (uint256 j; j < expiries.length; j++) {
                vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_InvalidExpiry.selector);
                vm.prank(verdictWriter);
                policy.clear(keccak256("item"), expiries[j]);
            }
        }
    }

    /// @notice Tests that only the new verdict writer can clear after setVerdictWriter.
    function test_clear_replacedVerdictWriter_succeeds() external {
        address newVerdictWriter = makeAddr("newVerdictWriter");
        bytes32 id = keccak256("item");
        uint64 expiresAt = uint64(block.timestamp + 1);
        vm.prank(owner);
        policy.setVerdictWriter(newVerdictWriter);

        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
        vm.prank(verdictWriter);
        policy.clear(id, expiresAt);

        vm.expectCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, expiresAt)), 1);
        vm.prank(newVerdictWriter);
        policy.clear(id, expiresAt);
    }
}

/// @title VisionPolicyV1_Revoke_Test
/// @notice Tests the `revoke` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_Revoke_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that revoke reverts for a caller other than the verdict writer.
    function testFuzz_revoke_notVerdictWriter_reverts(address _caller) external {
        vm.assume(_caller != verdictWriter);
        address[2] memory callers = [_caller, owner];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
            vm.prank(callers[i]);
            policy.revoke(keccak256("item"));
        }
    }

    function test_revoke_succeeds() external {
        bytes32 id = keccak256("item");

        vm.expectCall(module, abi.encodeCall(IComplianceModule.revokeVerdict, (id)), 1);
        vm.prank(verdictWriter);
        policy.revoke(id);
    }
}

/// @title VisionPolicyV1_ClearAndCompleteDeposit_Test
/// @notice Tests the `clearAndCompleteDeposit` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_ClearAndCompleteDeposit_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that clearAndCompleteDeposit reverts for a caller other than the verdict
    ///         writer.
    function testFuzz_clearAndCompleteDeposit_notVerdictWriter_reverts(address _caller) external {
        vm.assume(_caller != verdictWriter);
        BridgeHookTypes.Item memory item;
        address[2] memory callers = [_caller, owner];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
            vm.prank(callers[i]);
            policy.clearAndCompleteDeposit(item, uint64(block.timestamp + 1));
        }
    }

    /// @notice Tests that recordVerdict receives the item's hash before completeDeposit
    ///         receives the item.
    function test_clearAndCompleteDeposit_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: sender,
            aliased: false,
            to: recipient,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 0
        });
        uint64 expiresAt = uint64(block.timestamp + 1);
        bytes[2] memory expected = [
            abi.encodeCall(IComplianceModule.recordVerdict, (BridgeHookHashing.hashItem(item), expiresAt)),
            abi.encodeCall(IComplianceModule.completeDeposit, (item))
        ];

        vm.startStateDiffRecording();
        vm.prank(verdictWriter);
        policy.clearAndCompleteDeposit(item, expiresAt);
        VmSafe.AccountAccess[] memory accesses = vm.stopAndReturnStateDiff();

        uint256 calls;
        for (uint256 i; i < accesses.length; i++) {
            if (accesses[i].account != module || accesses[i].kind != VmSafe.AccountAccessKind.Call) continue;
            assertLt(calls, expected.length, "extra call");
            assertEq(accesses[i].data, expected[calls], vm.toString(calls));
            calls++;
        }
        assertEq(calls, expected.length, "calls");
    }

    /// @notice Tests that clearAndCompleteDeposit rejects expiry at or before the current
    ///         timestamp, or more than maxVerdictValidity seconds after it.
    function test_clearAndCompleteDeposit_invalidExpiry_reverts() external {
        BridgeHookTypes.Item memory item;
        uint64[3] memory expiries = [
            uint64(block.timestamp),
            uint64(block.timestamp - 1),
            uint64(block.timestamp + policy.maxVerdictValidity() + 1)
        ];
        vm.startPrank(verdictWriter);
        for (uint256 i; i < expiries.length; i++) {
            vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_InvalidExpiry.selector);
            policy.clearAndCompleteDeposit(item, expiries[i]);
        }
    }
}

/// @title VisionPolicyV1_ClearAndCompleteWithdrawal_Test
/// @notice Tests the `clearAndCompleteWithdrawal` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_ClearAndCompleteWithdrawal_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that clearAndCompleteWithdrawal reverts for a caller other than the verdict
    ///         writer.
    function testFuzz_clearAndCompleteWithdrawal_notVerdictWriter_reverts(address _caller) external {
        vm.assume(_caller != verdictWriter);
        BridgeHookTypes.Item memory item;
        address[2] memory callers = [_caller, owner];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
            vm.prank(callers[i]);
            policy.clearAndCompleteWithdrawal(item, uint64(block.timestamp + 1));
        }
    }

    /// @notice Tests that recordVerdict receives the item's hash before completeWithdrawal
    ///         receives the item.
    function test_clearAndCompleteWithdrawal_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: sender,
            aliased: false,
            to: recipient,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 0
        });
        uint64 expiresAt = uint64(block.timestamp + 1);
        bytes[2] memory expected = [
            abi.encodeCall(IComplianceModule.recordVerdict, (BridgeHookHashing.hashItem(item), expiresAt)),
            abi.encodeCall(IComplianceModule.completeWithdrawal, (item))
        ];

        vm.startStateDiffRecording();
        vm.prank(verdictWriter);
        policy.clearAndCompleteWithdrawal(item, expiresAt);
        VmSafe.AccountAccess[] memory accesses = vm.stopAndReturnStateDiff();

        uint256 calls;
        for (uint256 i; i < accesses.length; i++) {
            if (accesses[i].account != module || accesses[i].kind != VmSafe.AccountAccessKind.Call) continue;
            assertLt(calls, expected.length, "extra call");
            assertEq(accesses[i].data, expected[calls], vm.toString(calls));
            calls++;
        }
        assertEq(calls, expected.length, "calls");
    }

    /// @notice Tests that clearAndCompleteWithdrawal rejects expiry at or before the current
    ///         timestamp, or more than maxVerdictValidity seconds after it.
    function test_clearAndCompleteWithdrawal_invalidExpiry_reverts() external {
        BridgeHookTypes.Item memory item;
        uint64[3] memory expiries = [
            uint64(block.timestamp),
            uint64(block.timestamp - 1),
            uint64(block.timestamp + policy.maxVerdictValidity() + 1)
        ];
        vm.startPrank(verdictWriter);
        for (uint256 i; i < expiries.length; i++) {
            vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_InvalidExpiry.selector);
            policy.clearAndCompleteWithdrawal(item, expiries[i]);
        }
    }
}

/// @title VisionPolicyV1_AddToAllowlist_Test
/// @notice Tests the `addToAllowlist` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_AddToAllowlist_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that addToAllowlist reverts for a caller other than the owner.
    function testFuzz_addToAllowlist_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.addToAllowlist(new address[](0));
        }
    }

    /// @notice Tests that adding allowlist entries emits an event for each input entry,
    ///         including duplicates and addresses already on the list.
    function test_addToAllowlist_succeeds() external {
        address[] memory accounts = new address[](3);
        accounts[0] = sender;
        accounts[1] = sender;
        accounts[2] = allowlistedSender;

        for (uint256 i; i < accounts.length; i++) {
            vm.expectEmit(address(policy));
            emit AllowlistUpdated(accounts[i], true);
        }
        vm.prank(owner);
        policy.addToAllowlist(accounts);

        assertTrue(policy.allowlisted(sender), "sender");
        assertTrue(policy.allowlisted(allowlistedSender), "allowlistedSender");
    }
}

/// @title VisionPolicyV1_RemoveFromAllowlist_Test
/// @notice Tests the `removeFromAllowlist` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_RemoveFromAllowlist_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that removeFromAllowlist reverts for a caller other than the owner.
    function testFuzz_removeFromAllowlist_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.removeFromAllowlist(new address[](0));
        }
    }

    /// @notice Tests that removing allowlist entries emits an event for each input entry,
    ///         including duplicates and addresses already absent from the list.
    function test_removeFromAllowlist_succeeds() external {
        address[] memory accounts = new address[](3);
        accounts[0] = allowlistedSender;
        accounts[1] = allowlistedSender;
        accounts[2] = sender;

        for (uint256 i; i < accounts.length; i++) {
            vm.expectEmit(address(policy));
            emit AllowlistUpdated(accounts[i], false);
        }
        vm.prank(owner);
        policy.removeFromAllowlist(accounts);

        assertFalse(policy.allowlisted(allowlistedSender), "allowlistedSender");
        assertFalse(policy.allowlisted(sender), "sender");
    }
}

/// @title VisionPolicyV1_AddToDenylist_Test
/// @notice Tests the `addToDenylist` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_AddToDenylist_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that addToDenylist reverts for a caller other than the owner.
    function testFuzz_addToDenylist_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.addToDenylist(new address[](0));
        }
    }

    /// @notice Tests that adding deny list entries emits an event for each input entry,
    ///         including duplicates and addresses already on the list.
    function test_addToDenylist_succeeds() external {
        address[] memory accounts = new address[](3);
        accounts[0] = recipient;
        accounts[1] = recipient;
        accounts[2] = deniedParty;

        for (uint256 i; i < accounts.length; i++) {
            vm.expectEmit(address(policy));
            emit DenylistUpdated(accounts[i], true);
        }
        vm.prank(owner);
        policy.addToDenylist(accounts);

        assertTrue(policy.denylisted(recipient), "recipient");
        assertTrue(policy.denylisted(deniedParty), "deniedParty");
    }
}

/// @title VisionPolicyV1_RemoveFromDenylist_Test
/// @notice Tests the `removeFromDenylist` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_RemoveFromDenylist_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that removeFromDenylist reverts for a caller other than the owner.
    function testFuzz_removeFromDenylist_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.removeFromDenylist(new address[](0));
        }
    }

    /// @notice Tests that removing deny list entries emits an event for each input entry,
    ///         including duplicates and addresses already absent from the list.
    function test_removeFromDenylist_succeeds() external {
        address[] memory accounts = new address[](3);
        accounts[0] = deniedParty;
        accounts[1] = deniedParty;
        accounts[2] = recipient;

        for (uint256 i; i < accounts.length; i++) {
            vm.expectEmit(address(policy));
            emit DenylistUpdated(accounts[i], false);
        }
        vm.prank(owner);
        policy.removeFromDenylist(accounts);

        assertFalse(policy.denylisted(deniedParty), "deniedParty");
        assertFalse(policy.denylisted(recipient), "recipient");
    }
}

/// @title VisionPolicyV1_SetVerdictWriter_Test
/// @notice Tests the `setVerdictWriter` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_SetVerdictWriter_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that setVerdictWriter reverts for a caller other than the owner.
    function testFuzz_setVerdictWriter_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.setVerdictWriter(makeAddr("newVerdictWriter"));
        }
    }

    /// @notice Tests that setVerdictWriter sets the writer and emits, also for an unchanged value.
    function test_setVerdictWriter_succeeds() external {
        address newVerdictWriter = makeAddr("newVerdictWriter");

        vm.expectEmit(address(policy));
        emit VerdictWriterSet(verdictWriter, newVerdictWriter);
        vm.prank(owner);
        policy.setVerdictWriter(newVerdictWriter);
        assertEq(policy.verdictWriter(), newVerdictWriter);

        vm.expectEmit(address(policy));
        emit VerdictWriterSet(newVerdictWriter, newVerdictWriter);
        vm.prank(owner);
        policy.setVerdictWriter(newVerdictWriter);
    }
}

/// @title VisionPolicyV1_SetMaxVerdictValidity_Test
/// @notice Tests the `setMaxVerdictValidity` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_SetMaxVerdictValidity_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that setMaxVerdictValidity reverts for a caller other than the owner.
    function testFuzz_setMaxVerdictValidity_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.setMaxVerdictValidity(1 days);
        }
    }

    /// @notice Tests that setMaxVerdictValidity sets the value and emits at zero and at the cap,
    ///         also for an unchanged value.
    function test_setMaxVerdictValidity_succeeds() external {
        uint64[3] memory values = [0, MAX_VERDICT_VALIDITY_CAP, MAX_VERDICT_VALIDITY_CAP];
        vm.startPrank(owner);
        for (uint256 i; i < values.length; i++) {
            vm.expectEmit(address(policy));
            emit MaxVerdictValiditySet(values[i]);
            policy.setMaxVerdictValidity(values[i]);
            assertEq(policy.maxVerdictValidity(), values[i]);
        }
    }

    /// @notice Tests that setMaxVerdictValidity reverts above the cap.
    function test_setMaxVerdictValidity_aboveCap_reverts() external {
        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_OutOfBounds.selector);
        vm.prank(owner);
        policy.setMaxVerdictValidity(MAX_VERDICT_VALIDITY_CAP + 1);
    }
}

/// @title VisionPolicyV1_SetHoldDelay_Test
/// @notice Tests the `setHoldDelay` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_SetHoldDelay_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that setHoldDelay reverts for a caller other than the owner.
    function testFuzz_setHoldDelay_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);
        address[2] memory callers = [_caller, verdictWriter];
        for (uint256 i; i < callers.length; i++) {
            vm.expectRevert(bytes.concat(IVisionPolicyV1.OwnableUnauthorizedAccount.selector, abi.encode(callers[i])));
            vm.prank(callers[i]);
            policy.setHoldDelay(0);
        }
    }

    /// @notice Tests that setHoldDelay accepts zero and the cap, and emits even for an unchanged
    ///         value.
    ///         The cap is one second below the Portal's proof maturity delay in this test.
    function test_setHoldDelay_succeeds() external {
        vm.mockCall(
            portal, abi.encodeCall(IOptimismPortal2.proofMaturityDelaySeconds, ()), abi.encode(HOLD_DELAY_CAP + 1)
        );
        uint64[3] memory values = [0, HOLD_DELAY_CAP, HOLD_DELAY_CAP];
        vm.startPrank(owner);
        for (uint256 i; i < values.length; i++) {
            vm.expectEmit(address(policy));
            emit HoldDelaySet(values[i]);
            policy.setHoldDelay(values[i]);
            assertEq(policy.holdDelay(), values[i]);
        }
    }

    /// @notice Tests that a delay at or above the Portal's proof maturity delay causes
    ///         HoldDelayTooLong.
    ///         This error takes priority when the delay also exceeds HOLD_DELAY_CAP.
    function test_setHoldDelay_tooLong_reverts() external {
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.proofMaturityDelaySeconds, ()), abi.encode(1 hours));
        uint64[2] memory values = [uint64(1 hours), HOLD_DELAY_CAP + 1];
        vm.startPrank(owner);
        for (uint256 i; i < values.length; i++) {
            vm.expectRevert(
                bytes.concat(IVisionPolicyV1.VisionPolicyV1_HoldDelayTooLong.selector, abi.encode(uint64(1 hours)))
            );
            policy.setHoldDelay(values[i]);
        }
    }
}

/// @title VisionPolicyV1_Multicall_Test
/// @notice Tests the `multicall` function of the `VisionPolicyV1` contract.
contract VisionPolicyV1_Multicall_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that a verdict writer's batch of the verdict functions calls the module for
    ///         each entry.
    function test_multicall_verdictWriter_succeeds() external {
        BridgeHookTypes.Item memory deposit = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: sender,
            aliased: false,
            to: recipient,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 0
        });
        BridgeHookTypes.Item memory withdrawal = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: sender,
            aliased: false,
            to: recipient,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 0
        });
        bytes32 id = keccak256("item");
        uint64 expiresAt = uint64(block.timestamp + 1);
        bytes[] memory data = new bytes[](4);
        data[0] = abi.encodeCall(IVisionPolicyV1.clear, (id, expiresAt));
        data[1] = abi.encodeCall(IVisionPolicyV1.revoke, (id));
        data[2] = abi.encodeCall(IVisionPolicyV1.clearAndCompleteDeposit, (deposit, expiresAt));
        data[3] = abi.encodeCall(IVisionPolicyV1.clearAndCompleteWithdrawal, (withdrawal, expiresAt));

        vm.expectCall(module, abi.encodeCall(IComplianceModule.recordVerdict, (id, expiresAt)), 1);
        vm.expectCall(module, abi.encodeCall(IComplianceModule.revokeVerdict, (id)), 1);
        vm.expectCall(
            module, abi.encodeCall(IComplianceModule.recordVerdict, (BridgeHookHashing.hashItem(deposit), expiresAt)), 1
        );
        vm.expectCall(module, abi.encodeCall(IComplianceModule.completeDeposit, (deposit)), 1);
        vm.expectCall(
            module,
            abi.encodeCall(IComplianceModule.recordVerdict, (BridgeHookHashing.hashItem(withdrawal), expiresAt)),
            1
        );
        vm.expectCall(module, abi.encodeCall(IComplianceModule.completeWithdrawal, (withdrawal)), 1);
        vm.prank(verdictWriter);
        policy.multicall(data);
    }

    /// @notice Tests that a batch of the verdict functions reverts for a caller other than the
    ///         verdict writer.
    function testFuzz_multicall_notVerdictWriter_reverts(address _caller) external {
        vm.assume(_caller != verdictWriter);
        BridgeHookTypes.Item memory item;
        bytes32 id = keccak256("item");
        uint64 expiresAt = uint64(block.timestamp + 1);
        bytes[] memory data = new bytes[](4);
        data[0] = abi.encodeCall(IVisionPolicyV1.clear, (id, expiresAt));
        data[1] = abi.encodeCall(IVisionPolicyV1.revoke, (id));
        data[2] = abi.encodeCall(IVisionPolicyV1.clearAndCompleteDeposit, (item, expiresAt));
        data[3] = abi.encodeCall(IVisionPolicyV1.clearAndCompleteWithdrawal, (item, expiresAt));

        vm.expectRevert(IVisionPolicyV1.VisionPolicyV1_Unauthorized.selector);
        vm.prank(_caller);
        policy.multicall(data);
    }

    /// @notice Tests that a failing entry reverts the batch with that entry's revert data.
    function test_multicall_failingEntry_reverts() external {
        vm.mockCallRevert(module, abi.encodePacked(IComplianceModule.revokeVerdict.selector), hex"c0ffee");
        bytes[] memory data = new bytes[](2);
        data[0] = abi.encodeCall(IVisionPolicyV1.clear, (keccak256("item"), uint64(block.timestamp + 1)));
        data[1] = abi.encodeCall(IVisionPolicyV1.revoke, (keccak256("item")));

        vm.expectRevert(bytes(hex"c0ffee"));
        vm.prank(verdictWriter);
        policy.multicall(data);
    }

    /// @notice Tests that owner functions in a batch pass for the owner and run in order.
    function test_multicall_ownerFunction_succeeds() external {
        address[] memory accounts = new address[](1);
        accounts[0] = sender;
        bytes[] memory data = new bytes[](2);
        data[0] = abi.encodeCall(IVisionPolicyV1.addToAllowlist, (accounts));
        data[1] = abi.encodeCall(IVisionPolicyV1.removeFromAllowlist, (accounts));

        vm.expectEmit(address(policy));
        emit AllowlistUpdated(sender, true);
        vm.expectEmit(address(policy));
        emit AllowlistUpdated(sender, false);
        vm.prank(owner);
        policy.multicall(data);
        assertFalse(policy.allowlisted(sender));
    }

    /// @notice Tests that multicall rejects ETH.
    function test_multicall_withValue_reverts() external {
        bytes[] memory data = new bytes[](1);
        data[0] = abi.encodeCall(IVisionPolicyV1.revoke, (keccak256("item")));
        vm.deal(verdictWriter, 1);

        vm.prank(verdictWriter);
        (bool success,) = address(policy).call{ value: 1 }(abi.encodeCall(IVisionPolicyV1.multicall, (data)));
        assertFalse(success);
    }
}

/// @title VisionPolicyV1_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `VisionPolicyV1`
///         contract.
contract VisionPolicyV1_Uncategorized_Test is VisionPolicyV1_TestInit {
    /// @notice Tests that a plain ETH transfer to the policy reverts.
    function test_receive_ethTransfer_reverts() external {
        vm.deal(sender, 1);

        vm.prank(sender);
        (bool success,) = address(policy).call{ value: 1 }("");
        assertFalse(success);
    }
}
