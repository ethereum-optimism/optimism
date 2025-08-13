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
contract UnorderedExecutionModule_Testing_Harness {
    uint256 public value;
    bool public called;

    function setValue(uint256 _value) external payable {
        value = _value;
        called = true;
    }

    function revertingFunction() external pure {
        revert("UnorderedExecutionModule_Testing_Harness: intentional revert");
    }
}

contract UnorderedExecutionModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    UnorderedExecutionModule internal module;
    UnorderedExecutionModule_Testing_Harness internal target;
    SafeInstance internal safeInstance;

    function setUp() public virtual {
        // Create test accounts
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("UnorderedExecutionModule_test_", 10);

        // Create a Safe with 10 owners
        safeInstance = _setupSafe(keys, 10);

        // Deploy contracts
        module = new UnorderedExecutionModule();
        target = new UnorderedExecutionModule_Testing_Harness();

        // Enable the module on the Safe
        safeInstance.enableModule(address(module));
    }

    /// @notice Create transaction parameters for setValue call
    function _createSetValueParams(uint256 _value)
        internal
        view
        returns (UnorderedExecutionModule.ExecTransactionFromModuleParams memory)
    {
        return UnorderedExecutionModule.ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("setValue(uint256)", _value),
            operation: Enum.Operation.Call
        });
    }

    /// @notice Create transaction parameters for reverting call
    function _createRevertingParams()
        internal
        view
        returns (UnorderedExecutionModule.ExecTransactionFromModuleParams memory)
    {
        return UnorderedExecutionModule.ExecTransactionFromModuleParams({
            to: address(target),
            value: 0,
            data: abi.encodeWithSignature("revertingFunction()"),
            operation: Enum.Operation.Call
        });
    }

    /// @notice Generate transaction hash for given params and nonce
    function _getTransactionHash(
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory _params,
        bytes32 _hashOnce
    )
        internal
        view
        returns (bytes32)
    {
        return safeInstance.safe.getTransactionHash({
            to: _params.to,
            value: _params.value,
            data: _params.data,
            operation: _params.operation,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: address(0),
            _nonce: uint256(_hashOnce)
        });
    }

    /// @notice Generate signatures for transaction hash
    function _generateSignatures(bytes32 _txHash) internal view returns (bytes memory) {
        bytes memory signatures;
        for (uint256 i; i < safeInstance.ownerPKs.length; ++i) {
            uint256 pk = safeInstance.ownerPKs[i];
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(pk, _txHash);
            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }
        return signatures;
    }

    /// @notice Generate null prereq
    function _getNullPrereq() internal view returns (UnorderedExecutionModule.PrereqTxData memory) {
        return UnorderedExecutionModule.PrereqTxData({
            prevHashOnce: bytes32(0),
            mixHash: bytes32(0)
        });
    }
}

contract UnorderedExecutionModule_ExecTransactionOnSafe_Test is UnorderedExecutionModule_TestInit {
    /// @notice Test successful transaction execution
    function test_execTransactionOnSafe_succeeds() public {
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params = _createSetValueParams(42);
        uint256 nonce = uint256(keccak256(bytes("test nonce 1")));
        bytes32 safeTxHash = _getTransactionHash(params, bytes32(nonce));
        bytes memory signatures = _generateSignatures(safeTxHash);

        vm.expectEmit(true, true, true, true);
        emit UnorderedExecutionModule.TransactionExecuted(
            address(safeInstance.safe), safeTxHash, params.to, params.value, params.data, params.operation
        );

        bool success = module.execTransactionOnSafe(
            safeInstance.safe,
            bytes32(nonce),
            _getNullPrereq(),
            params,
            signatures
        );

        assertTrue(success);
        assertEq(target.value(), 42);
        assertTrue(target.called());
        assertTrue(module.executedTransactions(safeTxHash));
    }

    /// @notice Test transaction replay prevention
    function test_execTransactionOnSafe_alreadyExecuted_reverts() public {
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params = _createSetValueParams(42);
        uint256 nonce = uint256(keccak256(bytes("test nonce 2")));
        bytes32 safeTxHash = _getTransactionHash(params, bytes32(nonce));
        bytes memory signatures = _generateSignatures(safeTxHash);

        bool success = module.execTransactionOnSafe(
            safeInstance.safe,
            bytes32(nonce),
            _getNullPrereq(),
            params,
            signatures
        );
        assertTrue(success);

        // Execute second time - should revert
        vm.expectRevert(UnorderedExecutionModule.TransactionAlreadyExecuted.selector);
        module.execTransactionOnSafe(
            safeInstance.safe,
            bytes32(nonce),
            _getNullPrereq(),
            params,
            signatures
        );
    }

    /// @notice Test module execution failure
    function test_execTransactionOnSafe_moduleExecutionFailed_reverts() public {
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params = _createRevertingParams();
        uint256 nonce = uint256(keccak256(bytes("test nonce 3")));
        bytes32 safeTxHash = _getTransactionHash(params, bytes32(nonce));
        bytes memory signatures = _generateSignatures(safeTxHash);

        vm.expectRevert(UnorderedExecutionModule.ModuleExecutionFailed.selector);
        module.execTransactionOnSafe(safeInstance.safe, bytes32(nonce), _getNullPrereq(), params, signatures);
    }

    /// @notice Fuzz test with different nonces and values
    function testFuzz_execTransactionOnSafe_succeeds(uint256 _nonce, uint256 _value) public {
        _nonce = bound(_nonce, 1, type(uint256).max);
        _value = bound(_value, 0, type(uint256).max / 2);

        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params = _createSetValueParams(_value);
        bytes32 safeTxHash = _getTransactionHash(params, bytes32(_nonce));
        bytes memory signatures = _generateSignatures(safeTxHash);

        bool success = module.execTransactionOnSafe(
            safeInstance.safe,
            bytes32(_nonce),
            _getNullPrereq(),
            params,
            signatures
        );

        assertTrue(success);
        assertEq(target.value(), _value);
        assertTrue(target.called());
        assertTrue(module.executedTransactions(safeTxHash));
    }
}

contract UnorderedExecutionModule_IsEnabledOnSafe_Test is UnorderedExecutionModule_TestInit {
    /// @notice Test module is enabled on safe
    function test_isEnabledOnSafe_enabled_succeeds() public view {
        assertTrue(module.isEnabledOnSafe(safeInstance.safe));
    }

    /// @notice Test module is not enabled on different safe
    function test_isEnabledOnSafe_notEnabled_succeeds() public {
        uint256[] memory singlePK = new uint256[](1);
        singlePK[0] = 0x123456789abcdef123456789abcdef123456789abcdef123456789abcdef12;
        SafeInstance memory newSafeInstance = _setupSafe(singlePK, 1);
        Safe newSafe = Safe(payable(newSafeInstance.safe));

        assertFalse(module.isEnabledOnSafe(newSafe));
    }
}

contract UnorderedExecutionModule_Prereq_Test is UnorderedExecutionModule_TestInit {
    /// @notice Test successful execution when prereq is provided and valid
    function test_execTransactionOnSafe_withPrereq_succeeds() public {
        // First transaction (prerequisite)
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params1 = _createSetValueParams(1);
        bytes32 hashOnce1 = bytes32(uint256(keccak256(bytes("prev tx"))));
        bytes32 safeTxHash1 = _getTransactionHash(params1, hashOnce1);
        bytes memory signatures1 = _generateSignatures(safeTxHash1);

        bool ok1 = module.execTransactionOnSafe(
            safeInstance.safe,
            hashOnce1,
            _getNullPrereq(),
            params1,
            signatures1
        );
        assertTrue(ok1);
        assertTrue(module.executedTransactions(safeTxHash1));

        // Second transaction that depends on the first
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params2 = _createSetValueParams(2);
        UnorderedExecutionModule.PrereqTxData memory prereq = UnorderedExecutionModule.PrereqTxData({
            prevHashOnce: safeTxHash1,
            mixHash: keccak256(bytes("mix"))
        });
        bytes32 hashOnce2 = keccak256(abi.encode(prereq));
        bytes32 safeTxHash2 = _getTransactionHash(params2, hashOnce2);
        bytes memory signatures2 = _generateSignatures(safeTxHash2);

        vm.expectEmit(true, true, true, true);
        emit UnorderedExecutionModule.TransactionExecuted(
            address(safeInstance.safe), safeTxHash2, params2.to, params2.value, params2.data, params2.operation
        );

        bool ok2 = module.execTransactionOnSafe(
            safeInstance.safe,
            hashOnce2,
            prereq,
            params2,
            signatures2
        );

        assertTrue(ok2);
        assertEq(target.value(), 2);
        assertTrue(target.called());
        assertTrue(module.executedTransactions(safeTxHash2));
    }

    /// @notice Test revert when provided hashOnce does not match prereq hash
    function test_execTransactionOnSafe_prereqHashOnceMismatch_reverts() public {
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params = _createSetValueParams(100);

        // Construct a prereq with non-zero prevHashOnce
        UnorderedExecutionModule.PrereqTxData memory prereq = UnorderedExecutionModule.PrereqTxData({
            prevHashOnce: bytes32(uint256(123)),
            mixHash: keccak256(bytes("abc"))
        });

        // Intentionally use a different hashOnce than keccak256(abi.encode(prereq))
        bytes32 wrongHashOnce = bytes32(uint256(999));

        // Sign for wrongHashOnce
        bytes32 safeTxHash = _getTransactionHash(params, wrongHashOnce);
        bytes memory signatures = _generateSignatures(safeTxHash);

        vm.expectRevert(bytes("UnorderedExecutionModule: prereq hashOnce mismatch"));
        module.execTransactionOnSafe(safeInstance.safe, wrongHashOnce, prereq, params, signatures);
    }

    /// @notice Test revert when prereq.prevHashOnce has not been executed
    function test_execTransactionOnSafe_prereqPrevNotExecuted_reverts() public {
        UnorderedExecutionModule.ExecTransactionFromModuleParams memory params = _createSetValueParams(200);

        // Refers to a tx-hash that has not been executed
        UnorderedExecutionModule.PrereqTxData memory prereq = UnorderedExecutionModule.PrereqTxData({
            prevHashOnce: keccak256(bytes("never executed")),
            mixHash: keccak256(bytes("xyz"))
        });

        // Correctly set hashOnce to the prereq hash to pass the first check
        bytes32 hashOnce = keccak256(abi.encode(prereq));

        // Sign for hashOnce
        bytes32 safeTxHash = _getTransactionHash(params, hashOnce);
        bytes memory signatures = _generateSignatures(safeTxHash);

        vm.expectRevert(bytes("UnorderedExecutionModule: prereq prevHashOnce not executed"));
        module.execTransactionOnSafe(safeInstance.safe, hashOnce, prereq, params, signatures);
    }
}
