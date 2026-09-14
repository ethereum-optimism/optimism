// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { AtomicPaymaster } from "src/periphery/AtomicPaymaster.sol";
import { SimpleAccount } from "@account-abstraction/samples/SimpleAccount.sol";

// Libraries
import { Preinstalls } from "src/libraries/Preinstalls.sol";

// Interfaces
import { PackedUserOperation } from "@account-abstraction/interfaces/PackedUserOperation.sol";
import { IAtomicCallRouter } from "interfaces/periphery/IAtomicCallRouter.sol";
import { IAtomicPaymaster } from "interfaces/periphery/IAtomicPaymaster.sol";

/// @notice Focused sponsorship tests using the existing EntryPoint bytecode and entrypoint gate.
abstract contract AtomicPaymaster_TestInit is Test {
    uint256 internal constant COST_CAP = 1 ether;
    address internal owner = address(128);
    address internal account = address(256);
    address internal router = address(512);
    AtomicPaymaster internal paymaster;

    event AccountAllowed(address indexed account, bool allowed);

    function setUp() public {
        vm.etch(Preinstalls.EntryPoint_v070, Preinstalls.EntryPoint_v070Code);
        paymaster = new AtomicPaymaster(owner, router, COST_CAP);
        vm.prank(owner);
        paymaster.setAccountAllowed(account, true);
    }

    function _operation(
        address _target,
        uint256 _value,
        bytes memory _data
    )
        internal
        view
        returns (PackedUserOperation memory operation_)
    {
        operation_.sender = account;
        operation_.callData = abi.encodeCall(SimpleAccount.execute, (_target, _value, _data));
    }

    function _validOperation() internal view returns (PackedUserOperation memory) {
        return _operation(router, 0, abi.encodePacked(IAtomicCallRouter.executeRoot.selector));
    }
}

contract AtomicPaymaster_ValidatePaymasterUserOp_Test is AtomicPaymaster_TestInit {
    function test_validatePaymasterUserOp_exactCap_succeeds() external {
        PackedUserOperation memory operation = _validOperation();
        vm.recordLogs();
        vm.prank(Preinstalls.EntryPoint_v070);
        (bytes memory context, uint256 validationData) =
            paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
        assertEq(context.length, 0, "empty context disables postOp");
        assertEq(validationData, 0);
        assertEq(vm.getRecordedLogs().length, 0, "validation must preserve the fixed log prefix");
    }

    function test_validatePaymasterUserOp_remoteSelector_succeeds() external {
        PackedUserOperation memory operation =
            _operation(router, 0, abi.encodePacked(IAtomicCallRouter.executeRemote.selector));
        vm.prank(Preinstalls.EntryPoint_v070);
        (bytes memory context, uint256 validationData) = paymaster.validatePaymasterUserOp(operation, bytes32(0), 1);
        assertEq(context.length, 0);
        assertEq(validationData, 0);
    }

    function test_validatePaymasterUserOp_notEntryPoint_reverts() external {
        PackedUserOperation memory operation = _validOperation();
        vm.expectRevert("Sender not EntryPoint");
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_unapprovedAccount_reverts() external {
        PackedUserOperation memory operation = _validOperation();
        operation.sender = address(1024);
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_overCap_reverts() external {
        PackedUserOperation memory operation = _validOperation();
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP + 1);
    }

    function test_validatePaymasterUserOp_wrongTarget_reverts() external {
        PackedUserOperation memory operation =
            _operation(address(1024), 0, abi.encodePacked(IAtomicCallRouter.executeRoot.selector));
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_nativeValue_reverts() external {
        PackedUserOperation memory operation =
            _operation(router, 1, abi.encodePacked(IAtomicCallRouter.executeRoot.selector));
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_wrongRouterSelector_reverts() external {
        PackedUserOperation memory operation =
            _operation(router, 0, abi.encodePacked(IAtomicCallRouter.proxyFor.selector));
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_shortRouterData_reverts() external {
        PackedUserOperation memory operation = _operation(router, 0, hex"123456");
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_wrongAccountSelector_reverts() external {
        PackedUserOperation memory operation = _validOperation();
        operation.callData = abi.encodePacked(SimpleAccount.executeBatch.selector);
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_shortAccountData_reverts() external {
        PackedUserOperation memory operation = _validOperation();
        operation.callData = hex"123456";
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_validatePaymasterUserOp_malformedArguments_reverts() external {
        PackedUserOperation memory operation = _validOperation();
        operation.callData = abi.encodePacked(bytes4(keccak256("execute(address,uint256,bytes)")));
        vm.expectRevert(bytes(""));
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }
}

contract AtomicPaymaster_SetAccountAllowed_Test is AtomicPaymaster_TestInit {
    function test_setAccountAllowed_ownerRevocation_succeeds() external {
        vm.expectEmit(address(paymaster));
        emit AccountAllowed(account, false);
        vm.prank(owner);
        paymaster.setAccountAllowed(account, false);
        assertFalse(paymaster.allowedAccounts(account));

        PackedUserOperation memory operation = _validOperation();
        vm.expectRevert(AtomicPaymaster.AtomicPaymaster_NotSponsored.selector);
        vm.prank(Preinstalls.EntryPoint_v070);
        paymaster.validatePaymasterUserOp(operation, bytes32(0), COST_CAP);
    }

    function test_setAccountAllowed_notOwner_reverts() external {
        vm.expectRevert(abi.encodeWithSelector(IAtomicPaymaster.OwnableUnauthorizedAccount.selector, account));
        vm.prank(account);
        paymaster.setAccountAllowed(address(1024), true);
        assertFalse(paymaster.allowedAccounts(address(1024)));
    }
}
