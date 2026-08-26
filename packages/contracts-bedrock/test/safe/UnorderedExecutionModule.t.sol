// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import "test/safe-tools/SafeTestTools.sol";

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IUnorderedExecutionModule } from "interfaces/safe/IUnorderedExecutionModule.sol";

/// @notice Simple call target for module execution tests.
contract UnorderedExecutionModule_Target_Harness {
    uint256 public value;
    address public lastSender;
    uint256 public lastMsgValue;
    bool public shouldRevert;

    function setValue(uint256 _value) external payable {
        value = _value;
        lastSender = msg.sender;
        lastMsgValue = msg.value;
    }

    function setShouldRevert(bool _shouldRevert) external {
        shouldRevert = _shouldRevert;
    }

    function conditionalRevert() external view {
        require(!shouldRevert, "Target: forced revert");
    }
}

/// @notice Emits an event bound to address(this), so a delegatecall from the Safe emits from the
///         Safe's own address.
contract UnorderedExecutionModule_Emitter_Harness {
    event Emitted(address self);

    function emitEvent() external {
        emit Emitted(address(this));
    }
}

/// @title UnorderedExecutionModule_TestInit
/// @notice Reusable test initialization for `UnorderedExecutionModule` tests.
abstract contract UnorderedExecutionModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event TransactionExecuted(address indexed safe, bytes32 indexed txHash);
    event Emitted(address self);

    uint256 internal constant INIT_TIME = 1_000_000;
    uint256 internal constant NUM_OWNERS = 3;
    uint256 internal constant THRESHOLD = 2;
    uint256 internal constant SAFE_BALANCE = 100 ether;

    /// @notice Default hash-once value, as derived from a superchain-ops hashOnceInput string.
    uint256 internal constant HASH_ONCE = uint256(keccak256("task: default"));

    IUnorderedExecutionModule module;
    SafeInstance safeInstance;
    Safe safe;
    UnorderedExecutionModule_Target_Harness target;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        module = IUnorderedExecutionModule(
            DeployUtils.create1({
                _name: "UnorderedExecutionModule",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IUnorderedExecutionModule.__constructor__, ()))
            })
        );

        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("UnorderedExecutionModule_test_", NUM_OWNERS);
        safeInstance = _setupSafe(keys, THRESHOLD, SAFE_BALANCE);
        safe = Safe(payable(address(safeInstance.safe)));
        safeInstance.enableModule(address(module));

        target = new UnorderedExecutionModule_Target_Harness();
    }

    /// @notice Sets up an additional Safe with the given owners and a unique CREATE2 salt.
    function _setupExtraSafe(
        uint256[] memory _keys,
        uint256 _threshold,
        uint256 _salt
    )
        internal
        returns (SafeInstance memory)
    {
        saltNonce = _salt;
        return _setupSafe(_keys, _threshold, SAFE_BALANCE);
    }

    /// @notice Returns transaction params calling target.setValue(_value).
    function _makeParams(uint256 _value)
        internal
        view
        returns (IUnorderedExecutionModule.ExecTransactionParams memory)
    {
        return IUnorderedExecutionModule.ExecTransactionParams({
            to: address(target),
            value: 0,
            data: abi.encodeCall(UnorderedExecutionModule_Target_Harness.setValue, (_value)),
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0))
        });
    }

    /// @notice Signs the transaction's hash with the first _numSigs owner keys of _instance, in
    ///         the ascending-owner-address order required by the Safe's checkSignatures().
    function _signTx(
        SafeInstance memory _instance,
        IUnorderedExecutionModule.ExecTransactionParams memory _params,
        uint256 _hashOnce,
        uint256 _numSigs
    )
        internal
        view
        returns (bytes memory sigs_)
    {
        bytes32 digest = module.transactionHash(Safe(payable(address(_instance.safe))), _params, _hashOnce);
        for (uint256 i; i < _numSigs; i++) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(_instance.ownerPKs[i], digest);
            sigs_ = bytes.concat(sigs_, abi.encodePacked(r, s, v));
        }
    }

    /// @notice Signs a transaction with a threshold of the default Safe's owners.
    function _signTx(
        IUnorderedExecutionModule.ExecTransactionParams memory _params,
        uint256 _hashOnce
    )
        internal
        view
        returns (bytes memory)
    {
        return _signTx(safeInstance, _params, _hashOnce, THRESHOLD);
    }
}

/// @title UnorderedExecutionModule_Execute_Test
/// @notice Tests for the `execute` function.
contract UnorderedExecutionModule_Execute_Test is UnorderedExecutionModule_TestInit {
    using SafeTestLib for SafeInstance;

    /// @notice A signed transaction executes the call, is marked executed, and emits
    ///         TransactionExecuted.
    function test_execute_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(42);
        bytes memory sigs = _signTx(params, HASH_ONCE);
        bytes32 txHash = module.transactionHash(safe, params, HASH_ONCE);

        vm.expectEmit(address(module));
        emit TransactionExecuted(address(safe), txHash);
        module.execute(safe, params, HASH_ONCE, sigs);

        assertEq(target.value(), 42);
        assertEq(target.lastSender(), address(safe));
        assertTrue(module.executed(safe, txHash));
    }

    /// @notice Transactions execute in any order relative to the order they were signed in, and
    ///         the Safe's nonce is never consumed.
    function test_execute_outOfOrder_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params1 = _makeParams(1);
        IUnorderedExecutionModule.ExecTransactionParams memory params2 = _makeParams(2);
        uint256 hashOnce1 = uint256(keccak256("task: one"));
        uint256 hashOnce2 = uint256(keccak256("task: two"));
        bytes memory sigs1 = _signTx(params1, hashOnce1);
        bytes memory sigs2 = _signTx(params2, hashOnce2);

        uint256 nonceBefore = safe.nonce();

        // Execute in the reverse of signing order.
        module.execute(safe, params2, hashOnce2, sigs2);
        assertEq(target.value(), 2);
        module.execute(safe, params1, hashOnce1, sigs1);
        assertEq(target.value(), 1);

        assertEq(safe.nonce(), nonceBefore);
    }

    /// @notice The transaction hash does not depend on the Safe's nonce: signatures collected
    ///         before other (nonce-based) transactions execute remain valid afterwards.
    function test_execute_nonceIndependent_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(7);
        bytes32 hashBefore = module.transactionHash(safe, params, HASH_ONCE);
        bytes memory sigs = _signTx(params, HASH_ONCE);

        // Execute an ordinary nonce-based Safe transaction in the meantime.
        safeInstance.execTransaction(
            address(target), 0, abi.encodeCall(UnorderedExecutionModule_Target_Harness.setValue, (999))
        );
        assertEq(target.value(), 999);

        // The transaction hash is unchanged and the old signatures still execute.
        assertEq(module.transactionHash(safe, params, HASH_ONCE), hashBefore);
        module.execute(safe, params, HASH_ONCE, sigs);
        assertEq(target.value(), 7);
    }

    /// @notice Transactions can carry ETH value from the Safe.
    function test_execute_withValue_succeeds() external {
        address recipient = makeAddr("recipient");
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(0);
        params.to = recipient;
        params.value = 1 ether;
        params.data = "";
        bytes memory sigs = _signTx(params, HASH_ONCE);

        module.execute(safe, params, HASH_ONCE, sigs);

        assertEq(recipient.balance, 1 ether);
        assertEq(address(safe).balance, SAFE_BALANCE - 1 ether);
    }

    /// @notice Delegatecall transactions run in the Safe's own context.
    function test_execute_delegatecall_succeeds() external {
        UnorderedExecutionModule_Emitter_Harness emitter = new UnorderedExecutionModule_Emitter_Harness();
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(0);
        params.to = address(emitter);
        params.data = abi.encodeCall(UnorderedExecutionModule_Emitter_Harness.emitEvent, ());
        params.operation = Enum.Operation.DelegateCall;
        bytes memory sigs = _signTx(params, HASH_ONCE);

        vm.expectEmit(address(safe));
        emit Emitted(address(safe));
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice The same call can be authorized and executed twice using distinct hash-once
    ///         values.
    function test_execute_sameParamsDifferentHashOnce_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(5);
        uint256 hashOnce1 = uint256(keccak256("task: first run"));
        uint256 hashOnce2 = uint256(keccak256("task: second run"));

        assertTrue(module.transactionHash(safe, params, hashOnce1) != module.transactionHash(safe, params, hashOnce2));

        module.execute(safe, params, hashOnce1, _signTx(params, hashOnce1));
        module.execute(safe, params, hashOnce2, _signTx(params, hashOnce2));
        assertEq(target.value(), 5);
    }

    /// @notice Owners can authorize a transaction with Safe.approveHash() instead of an ECDSA
    ///         signature.
    function test_execute_approvedHash_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(11);
        bytes32 txHash = module.transactionHash(safe, params, HASH_ONCE);

        // Owners are sorted ascending, matching the order checkSignatures() requires.
        bytes memory sigs;
        for (uint256 i; i < THRESHOLD; i++) {
            address owner = safeInstance.owners[i];
            vm.prank(owner);
            safe.approveHash(txHash);
            sigs = bytes.concat(sigs, abi.encodePacked(bytes32(uint256(uint160(owner))), bytes32(0), uint8(1)));
        }

        module.execute(safe, params, HASH_ONCE, sigs);
        assertEq(target.value(), 11);
    }

    /// @notice A nested Safe owner can authorize a transaction with an EIP-1271 contract
    ///         signature over the Safe's encoded transaction data.
    function test_execute_contractSignature_succeeds() external {
        // Child Safe that will act as the sole owner of a parent Safe, as with a nested Safe
        // setup like the L1PAO.
        (, uint256[] memory childKeys) = SafeTestLib.makeAddrsAndKeys("UnorderedExecutionModule_child_", 2);
        SafeInstance memory childInstance = _setupExtraSafe(childKeys, 2, 100);

        uint256[] memory parentPKs = new uint256[](1);
        parentPKs[0] = SafeTestLib.encodeSmartContractWalletAsPK(address(childInstance.safe));
        SafeInstance memory parentInstance = _setupExtraSafe(parentPKs, 1, 101);
        Safe parentSafe = Safe(payable(address(parentInstance.safe)));
        // SafeTestLib's execTransaction cannot auto-sign for contract owners, so enable the
        // module by impersonating the Safe itself.
        vm.prank(address(parentSafe));
        parentSafe.enableModule(address(module));

        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(21);

        // The child Safe signs the parent Safe's encoded transaction data as a Safe message.
        bytes memory txHashData = parentSafe.encodeTransactionData(
            params.to,
            params.value,
            params.data,
            params.operation,
            params.safeTxGas,
            params.baseGas,
            params.gasPrice,
            params.gasToken,
            params.refundReceiver,
            HASH_ONCE
        );
        SafeTestLib.EIP1271Sign(childInstance, txHashData);

        // Contract signature: r = child Safe address, s = offset of the (empty) dynamic part,
        // v = 0.
        bytes memory sigs = abi.encodePacked(
            bytes32(uint256(uint160(address(childInstance.safe)))), bytes32(uint256(65)), uint8(0), uint256(0)
        );

        module.execute(parentSafe, params, HASH_ONCE, sigs);
        assertEq(target.value(), 21);
        assertEq(target.lastSender(), address(parentSafe));
    }

    /// @notice A Safe reporting version 1.3.0 is accepted.
    function test_execute_safeVersion130_succeeds() external {
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(safe), abi.encodeWithSignature("VERSION()"), abi.encode("1.3.0"));
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(9);
        module.execute(safe, params, HASH_ONCE, _signTx(params, HASH_ONCE));
        assertEq(target.value(), 9);
    }

    /// @notice The smallest allowed hash-once value is accepted.
    function test_execute_hashOnceAtBoundary_succeeds() external {
        uint256 hashOnce = uint256(type(uint128).max) + 1;
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(4);
        module.execute(safe, params, hashOnce, _signTx(params, hashOnce));
        assertEq(target.value(), 4);
    }

    /// @notice A transaction with a signed safeTxGas executes when enough gas is supplied.
    function test_execute_safeTxGasSatisfied_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(13);
        params.safeTxGas = 100_000;
        module.execute(safe, params, HASH_ONCE, _signTx(params, HASH_ONCE));
        assertEq(target.value(), 13);
    }

    /// @notice A failed transaction reverts, remains unexecuted, and can be retried.
    function test_execute_retryAfterFailure_succeeds() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(0);
        params.data = abi.encodeCall(UnorderedExecutionModule_Target_Harness.conditionalRevert, ());
        bytes memory sigs = _signTx(params, HASH_ONCE);
        bytes32 txHash = module.transactionHash(safe, params, HASH_ONCE);

        target.setShouldRevert(true);
        vm.expectPartialRevert(IUnorderedExecutionModule.UnorderedExecutionModule_ExecutionFailed.selector);
        module.execute(safe, params, HASH_ONCE, sigs);

        // The failed execution left the transaction unexecuted.
        assertFalse(module.executed(safe, txHash));

        // The same signatures work once the call can succeed.
        target.setShouldRevert(false);
        module.execute(safe, params, HASH_ONCE, sigs);
        assertTrue(module.executed(safe, txHash));
    }

    /// @notice A transaction cannot be executed twice with the same signatures.
    function test_execute_replay_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(params, HASH_ONCE);

        module.execute(safe, params, HASH_ONCE, sigs);

        vm.expectRevert(IUnorderedExecutionModule.UnorderedExecutionModule_AlreadyExecuted.selector);
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice A hash-once value small enough to collide with a real Safe nonce is rejected, so
    ///         the same signatures can never execute through both this module and
    ///         execTransaction().
    function test_execute_hashOnceTooSmall_reverts() external {
        uint256 hashOnce = uint256(type(uint128).max);
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(params, hashOnce);

        vm.expectRevert(IUnorderedExecutionModule.UnorderedExecutionModule_HashOnceTooSmall.selector);
        module.execute(safe, params, hashOnce, sigs);
    }

    /// @notice A transaction promising a gas refund is rejected, since the module pays none.
    function test_execute_refundNotSupported_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        params.gasPrice = 1;
        bytes memory sigs = _signTx(params, HASH_ONCE);

        vm.expectRevert(IUnorderedExecutionModule.UnorderedExecutionModule_RefundNotSupported.selector);
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice A transaction cannot be executed with less gas available than its signed
    ///         safeTxGas, so a relayer cannot consume the transaction with a partial, low-gas
    ///         execution.
    function test_execute_insufficientGas_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(13);
        params.safeTxGas = 30_000_000;
        bytes memory sigs = _signTx(params, HASH_ONCE);
        bytes32 txHash = module.transactionHash(safe, params, HASH_ONCE);

        vm.expectRevert(IUnorderedExecutionModule.UnorderedExecutionModule_InsufficientGas.selector);
        module.execute{ gas: 1_000_000 }(safe, params, HASH_ONCE, sigs);

        // The transaction remains executable with sufficient gas.
        assertFalse(module.executed(safe, txHash));
    }

    /// @notice Fewer signatures than the Safe's threshold are rejected.
    function test_execute_insufficientSignatures_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(safeInstance, params, HASH_ONCE, THRESHOLD - 1);

        vm.expectRevert(bytes("GS020"));
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice Signatures from non-owners are rejected.
    function test_execute_invalidSigner_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes32 digest = module.transactionHash(safe, params, HASH_ONCE);

        (, uint256 badKey) = makeAddrAndKey("notAnOwner");
        (uint8 v1, bytes32 r1, bytes32 s1) = vm.sign(badKey, digest);
        (uint8 v2, bytes32 r2, bytes32 s2) = vm.sign(safeInstance.ownerPKs[0], digest);
        bytes memory sigs = abi.encodePacked(r1, s1, v1, r2, s2, v2);

        vm.expectRevert(bytes("GS026"));
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice Signatures do not carry over to modified transaction params or a modified
    ///         hash-once value.
    function test_execute_tamperedParams_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(params, HASH_ONCE);

        params.value = 1 ether;
        vm.expectRevert(bytes("GS026"));
        module.execute(safe, params, HASH_ONCE, sigs);

        params.value = 0;
        vm.expectRevert(bytes("GS026"));
        module.execute(safe, params, uint256(keccak256("task: other")), sigs);
    }

    /// @notice A signature for one Safe cannot be replayed against another Safe with the same
    ///         owners.
    function test_execute_wrongSafe_reverts() external {
        SafeInstance memory otherInstance = _setupExtraSafe(safeInstance.ownerPKs, THRESHOLD, 102);
        Safe otherSafe = Safe(payable(address(otherInstance.safe)));
        otherInstance.enableModule(address(module));

        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        // Signed for `safe`, executed against `otherSafe`.
        bytes memory sigs = _signTx(safeInstance, params, HASH_ONCE, THRESHOLD);

        vm.expectRevert(bytes("GS026"));
        module.execute(otherSafe, params, HASH_ONCE, sigs);
    }

    /// @notice Signatures are checked against the Safe's current threshold, not the threshold at
    ///         signing time.
    function test_execute_thresholdRaisedAfterSigning_reverts() external {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(params, HASH_ONCE);

        safeInstance.changeThreshold(THRESHOLD + 1);

        vm.expectRevert(bytes("GS020"));
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice Execution requires the module to be enabled on the Safe.
    function test_execute_moduleNotEnabled_reverts() external {
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("UnorderedExecutionModule_noModule_", 2);
        SafeInstance memory otherInstance = _setupExtraSafe(keys, 2, 103);
        Safe otherSafe = Safe(payable(address(otherInstance.safe)));

        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(otherInstance, params, HASH_ONCE, 2);

        vm.expectRevert(IUnorderedExecutionModule.UnorderedExecutionModule_ModuleNotEnabled.selector);
        module.execute(otherSafe, params, HASH_ONCE, sigs);
    }

    /// @notice Safes with unsupported versions are rejected.
    function test_execute_invalidSafeVersion_reverts() external {
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(safe), abi.encodeWithSignature("VERSION()"), abi.encode("1.5.0"));

        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        bytes memory sigs = _signTx(params, HASH_ONCE);

        vm.expectRevert(IUnorderedExecutionModule.UnorderedExecutionModule_InvalidSafeVersion.selector);
        module.execute(safe, params, HASH_ONCE, sigs);
    }

    /// @notice Any signed transaction with valid parameters executes.
    function testFuzz_execute_succeeds(uint256 _value, uint256 _hashOnce) external {
        _hashOnce = bound(_hashOnce, uint256(type(uint128).max) + 1, type(uint256).max);

        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(_value);
        bytes memory sigs = _signTx(params, _hashOnce);
        bytes32 txHash = module.transactionHash(safe, params, _hashOnce);

        module.execute(safe, params, _hashOnce, sigs);

        assertEq(target.value(), _value);
        assertTrue(module.executed(safe, txHash));
    }
}

/// @title UnorderedExecutionModule_TransactionHash_Test
/// @notice Tests for the `transactionHash` function.
contract UnorderedExecutionModule_TransactionHash_Test is UnorderedExecutionModule_TestInit {
    /// @notice transactionHash() is exactly the Safe's own getTransactionHash() with the
    ///         hash-once value in the nonce slot, so existing signing tooling works unchanged.
    function test_transactionHash_matchesSafe_works() external view {
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        assertEq(
            module.transactionHash(safe, params, HASH_ONCE),
            safe.getTransactionHash(
                params.to,
                params.value,
                params.data,
                params.operation,
                params.safeTxGas,
                params.baseGas,
                params.gasPrice,
                params.gasToken,
                params.refundReceiver,
                HASH_ONCE
            )
        );
    }

    /// @notice The transaction hash binds the Safe address.
    function test_transactionHash_bindsSafe_works() external {
        SafeInstance memory otherInstance = _setupExtraSafe(safeInstance.ownerPKs, THRESHOLD, 105);
        Safe otherSafe = Safe(payable(address(otherInstance.safe)));

        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        assertTrue(
            module.transactionHash(safe, params, HASH_ONCE) != module.transactionHash(otherSafe, params, HASH_ONCE)
        );
    }

    /// @notice The transaction hash binds the hash-once value.
    function testFuzz_transactionHash_bindsHashOnce_works(uint256 _hashOnce1, uint256 _hashOnce2) external view {
        vm.assume(_hashOnce1 != _hashOnce2);
        IUnorderedExecutionModule.ExecTransactionParams memory params = _makeParams(1);
        assertTrue(module.transactionHash(safe, params, _hashOnce1) != module.transactionHash(safe, params, _hashOnce2));
    }
}

/// @title UnorderedExecutionModule_Uncategorized_Test
/// @notice Tests for simple getters.
contract UnorderedExecutionModule_Uncategorized_Test is UnorderedExecutionModule_TestInit {
    /// @notice deriveHashOnce() is the keccak256 of the input string, always above the hash-once
    ///         floor for realistic inputs.
    function test_deriveHashOnce_works() external view {
        string memory input = "eip/upgrade-99: totally unique string";
        assertEq(module.deriveHashOnce(input), uint256(keccak256(bytes(input))));
        assertTrue(module.deriveHashOnce(input) > uint256(type(uint128).max));
    }

    /// @notice Unknown transaction hashes are unexecuted.
    function test_executed_default_works() external view {
        assertFalse(module.executed(safe, bytes32(uint256(123))));
    }
}
