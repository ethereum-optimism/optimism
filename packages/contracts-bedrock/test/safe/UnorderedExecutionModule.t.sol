// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing
import { Test } from "forge-std/Test.sol";
import { console } from "forge-std/console.sol";

// Safe contracts
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { GnosisSafeProxyFactory } from "safe-contracts/proxies/GnosisSafeProxyFactory.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Test utilities
import { SafeTestTools, SafeInstance, SafeTestLib } from "test/safe-tools/SafeTestTools.sol";

// Contracts under test
import { UnorderedExecutionModule } from "src/safe/UnorderedExecutionModule.sol";
import { IUnorderedExecutionModule } from "interfaces/safe/IUnorderedExecutionModule.sol";

// Mock contracts for testing
contract MockTarget {
    uint256 public value;
    bool public called;

    function setValue(uint256 _value) external payable {
        value = _value;
        called = true;
    }

    function revertingFunction() external pure {
        revert("MockTarget: intentional revert");
    }
}

contract UnorderedExecutionModuleTest is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    Safe internal safe;
    UnorderedExecutionModule internal module;
    MockTarget internal target;
    SafeInstance internal safeInstance;

    address internal owner1;
    address internal owner2;
    address internal owner3;

    uint256 internal owner1Key;
    uint256 internal owner2Key;
    uint256 internal owner3Key;

    function setUp() public {
        // Create test accounts
        (owner1, owner1Key) = makeAddrAndKey("owner1");
        (owner2, owner2Key) = makeAddrAndKey("owner2");
        (owner3, owner3Key) = makeAddrAndKey("owner3");

        // Deploy contracts
        module = new UnorderedExecutionModule();
        target = new MockTarget();

        // Create Safe with 2/3 threshold using SafeTestTools
        uint256[] memory ownerPKs = new uint256[](3);
        ownerPKs[0] = owner1Key;
        ownerPKs[1] = owner2Key;
        ownerPKs[2] = owner3Key;

        safeInstance = _setupSafe(ownerPKs, 2);
        safe = Safe(payable(safeInstance.safe));

        // Enable the module on the Safe
        safeInstance.enableModule(address(module));

        // Give Safe some ETH for gas
        vm.deal(address(safe), 10 ether);
    }

    function test_isNoncelessEnabled() public view {
        assertTrue(module.isNoncelessEnabled());
    }

    function test_isEnabledOnSafe() public view {
        assertTrue(module.isEnabledOnSafe(safe));
    }

    function test_isEnabledOnSafe_NotEnabled() public {
        uint256[] memory singlePK = new uint256[](1);
        singlePK[0] = 0x123; // dummy key
        SafeInstance memory newSafeInstance = _setupSafe(singlePK, 1);
        Safe newSafe = Safe(payable(newSafeInstance.safe));
        assertFalse(module.isEnabledOnSafe(newSafe));
    }

    function test_execTransactionOnSafe_Success() public {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        // Generate signatures from 2 owners (threshold is 2)
        bytes memory signatures = _generateSignatures(params, owner1Key, owner2Key);

        // Execute transaction
        bool success = module.execTransactionOnSafe(safe, params, signatures);

        assertTrue(success);
        assertEq(target.value(), 42);
        assertTrue(target.called());
    }

    function test_execTransactionOnSafe_WithValue() public {
        uint256 valueToSend = 1 ether;

        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: valueToSend,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        bytes memory signatures = _generateSignatures(params, owner1Key, owner2Key);

        uint256 targetBalanceBefore = address(target).balance;
        bool success = module.execTransactionOnSafe(safe, params, signatures);

        assertTrue(success);
        assertEq(target.value(), 42);
        assertEq(address(target).balance, targetBalanceBefore + valueToSend);
    }

    function test_execTransactionOnSafe_ReplayProtection() public {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        bytes memory signatures = _generateSignatures(params, owner1Key, owner2Key);

        // First execution should succeed
        bool success = module.execTransactionOnSafe(safe, params, signatures);
        assertTrue(success);

        // Second execution should revert with TransactionAlreadyExecuted
        vm.expectRevert(UnorderedExecutionModule.TransactionAlreadyExecuted.selector);
        module.execTransactionOnSafe(safe, params, signatures);
    }

    function test_execTransactionOnSafe_InvalidSignature() public {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        // Use wrong signature (only 1 signature when threshold is 2)
        bytes memory signatures = _generateSignatures(params, owner1Key);

        vm.expectRevert(UnorderedExecutionModule.InvalidSignature.selector);
        module.execTransactionOnSafe(safe, params, signatures);
    }

    function test_execTransactionOnSafe_ExecutionFailure() public {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("revertingFunction()"),
            operation: Enum.Operation.Call
        });

        bytes memory signatures = _generateSignatures(params, owner1Key, owner2Key);

        vm.expectRevert(UnorderedExecutionModule.ModuleExecutionFailed.selector);
        module.execTransactionOnSafe(safe, params, signatures);
    }

    function test_getTransactionHashes() public view {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        (bytes32 txHash, bytes32 perSafeTxHash) = module.getTransactionHashes(safe, params);

        assertNotEq(txHash, bytes32(0));
        assertNotEq(perSafeTxHash, bytes32(0));
        assertNotEq(txHash, perSafeTxHash);
    }

    function test_differentSafes_DifferentHashes() public {
        uint256[] memory singlePK = new uint256[](1);
        singlePK[0] = 0x456; // different dummy key
        SafeInstance memory safe2Instance = _setupSafe(singlePK, 1);
        Safe safe2 = Safe(payable(safe2Instance.safe));

        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        (bytes32 txHash1, bytes32 perSafeTxHash1) = module.getTransactionHashes(safe, params);
        (bytes32 txHash2, bytes32 perSafeTxHash2) = module.getTransactionHashes(safe2, params);

        // Different Safes should produce different hashes
        assertNotEq(txHash1, txHash2);
        assertNotEq(perSafeTxHash1, perSafeTxHash2);
    }

    function test_differentChains_DifferentHashes() public {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        // Get hash on current chain
        (bytes32 txHash1, bytes32 perSafeTxHash1) = module.getTransactionHashes(safe, params);

        // Mock different chain ID
        vm.chainId(999);
        (bytes32 txHash2, bytes32 perSafeTxHash2) = module.getTransactionHashes(safe, params);

        // Different chains should produce different hashes
        assertNotEq(txHash1, txHash2);
        assertNotEq(perSafeTxHash1, perSafeTxHash2);
    }

    function test_eventEmission() public {
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params = IUnorderedExecutionModule
            .ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", 42),
            operation: Enum.Operation.Call
        });

        bytes memory signatures = _generateSignatures(params, owner1Key, owner2Key);

        (bytes32 expectedTxHash,) = module.getTransactionHashes(safe, params);

        vm.expectEmit(true, true, false, true);
        emit IUnorderedExecutionModule.TransactionExecuted(
            address(safe), expectedTxHash, params.to, params.value, params.data, params.operation
        );

        module.execTransactionOnSafe(safe, params, signatures);
    }

    // Helper function to generate signatures for a transaction
    function _generateSignatures(
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params,
        uint256 key1
    )
        internal
        view
        returns (bytes memory)
    {
        (bytes32 txHash,) = module.getTransactionHashes(safe, params);

        (uint8 v, bytes32 r, bytes32 s) = vm.sign(key1, txHash);
        return abi.encodePacked(r, s, v);
    }

    function _generateSignatures(
        IUnorderedExecutionModule.ExecTransactionFromModuleParams memory params,
        uint256 key1,
        uint256 key2
    )
        internal
        view
        returns (bytes memory)
    {
        (bytes32 txHash,) = module.getTransactionHashes(safe, params);

        (uint8 v1, bytes32 r1, bytes32 s1) = vm.sign(key1, txHash);
        (uint8 v2, bytes32 r2, bytes32 s2) = vm.sign(key2, txHash);

        // Sort signatures by address (Safe requires this)
        address signer1 = vm.addr(key1);
        address signer2 = vm.addr(key2);

        if (signer1 < signer2) {
            return abi.encodePacked(r1, s1, v1, r2, s2, v2);
        } else {
            return abi.encodePacked(r2, s2, v2, r1, s1, v1);
        }
    }
}
