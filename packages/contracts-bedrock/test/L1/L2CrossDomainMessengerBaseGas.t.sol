// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";

// Libraries
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import "src/libraries/GasConstants.sol" as GasConstants;
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";

/// @title L2CrossDomainMessenger_BaseGas_Test
/// @notice Tests L2 message base gas against L1 execution costs.
contract L2CrossDomainMessenger_BaseGas_Test is CommonTest {
    /// @notice EIP-7702 delegated EOA whose delegate consumes all forwarded gas. Created in `setUp` so
    ///         that it is cold during the test.
    address delegatedTarget;

    function setUp() public virtual override {
        super.setUp();
        delegatedTarget = makeAddr("delegatedTarget");
        vm.etch(delegatedTarget, abi.encodePacked(hex"ef0100", address(new GasBurner(type(uint32).max))));
    }

    /// @notice Tests that `relayMessage` on L1 will always succeed for any potential message,
    ///         regardless of the size of the message, the minimum gas limit, or the amount of gas
    ///         used by the target contract.
    ///         Not isolated: isolation treats the gas limit as a transaction's, which must cover
    ///         intrinsic gas. The relay is OptimismPortal2's internal call, whose gas limit never
    ///         has to.
    /// forge-config: default.isolate = false
    function testFuzz_relayMessage_baseGasSufficient_succeeds(
        uint24 _messageLength,
        uint32 _minGasLimit,
        uint32 _gasToUse,
        uint96 _value
    )
        external
    {
        // TODO(#14609): Update this test to use default.isolate = true once a new stable Foundry
        // release is available that includes #9904. That will allow us to use this test to check
        // for changes to the EVM itself that might cause our gas formula to be incorrect.

        // Skip if this is a fork test, won't work.
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        // Keep the fuzz input bounded so the test remains fast. Near-max payloads are covered by a
        // deterministic regression test below.
        _messageLength = uint24(bound(_messageLength, 0, 34_000));

        // Need more than 500 since GasBurner requires it.
        // It's ok to try to use more than minGasLimit since the L2CrossDomainMessenger should
        // catch the revert and store the message hash in the failedMessages mapping.
        _gasToUse = uint32(bound(_gasToUse, 500, type(uint32).max));

        // Generate random bytes, more useful than having a _message parameter.
        bytes memory _message = vm.randomBytes(_messageLength);

        // Compute the base gas.
        // Base gas should really be computed on the fully encoded message but that would break the
        // expected API, so we instead just add the encoding overhead to the message length inside
        // of the baseGas function.
        uint64 baseGas = l2CrossDomainMessenger.baseGas(_message, _minGasLimit);

        // Deploy a gas burner.
        address target = address(new GasBurner(_gasToUse));

        // Encode the message.
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            alice, // Sender doesn't matter
            target,
            _value,
            _minGasLimit,
            _message
        );

        uint256 tokens = calldataTokens(encoded);

        // Base gas must always be sufficient to cover the floor cost from EIP-7623.
        assertGt(baseGas, 21000 + tokens * 10);

        // In the L2 => L1 direction we actually get all of the base gas supplied, nothing is
        // deducted. This is an advantage over the L1 => L2 direction because it means the base
        // gas goes a lot further.
        uint256 gasSupplied = baseGas;

        // Store the value of l2Sender in the OptimismPortal2 contract.
        StorageSlot memory slot = ForgeArtifacts.getSlot("OptimismPortal2", "l2Sender");
        vm.store(address(optimismPortal2), bytes32(slot.slot), bytes32(abi.encode(address(l2CrossDomainMessenger))));

        // We'll trigger the L1CrossDomainMessenger as if we're the OptimismPortal2
        vm.deal(address(optimismPortal2), _value);
        vm.prank(address(optimismPortal2));

        // Trigger the L1CrossDomainMessenger.
        // Should NOT fail.
        (bool success,) = address(l1CrossDomainMessenger).call{ gas: gasSupplied, value: _value }(encoded);
        assertTrue(success, "L1CrossDomainMessenger call should not fail");

        // Message should either be in the failed or successful messages mapping.
        bool inFailedMessages = l1CrossDomainMessenger.failedMessages(keccak256(encoded));
        bool inSuccessfulMessages = l1CrossDomainMessenger.successfulMessages(keccak256(encoded));
        assertTrue(
            inFailedMessages || inSuccessfulMessages, "message should be in either failed or successful messages"
        );
    }

    /// @notice Tests that `relayMessage` has enough base gas to finish when relaying a near-max
    ///         L2 => L1 payload to a target that consumes all forwarded gas.
    ///         Not isolated: isolation treats the gas limit as a transaction's, which must cover
    ///         intrinsic gas. The relay is OptimismPortal2's internal call, whose gas limit never
    ///         has to.
    /// forge-config: default.isolate = false
    function test_relayMessage_nearMaxPayloadAllGasTarget_succeeds() external {
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        // Use the PoC's _minGasLimit value referenced from client-pod#609:
        // https://github.com/ethereum-optimism/client-pod/issues/609
        uint32 minGasLimit = 50_000;

        // Match the largest 32-byte-aligned user message that fits under the portal's 120k
        // calldata cap once encoded as relayMessage(uint256,address,address,uint256,uint256,bytes).
        // The fixed encoding overhead is 4 bytes for the selector, 6 ABI words for the nonce,
        // sender, target, value, minGasLimit, and message offset, and 1 ABI word for the dynamic
        // bytes length. The message length is rounded down to a 32-byte multiple because dynamic
        // bytes data is ABI word-padded.
        uint256 portalCalldataLimit = 120_000;
        uint256 relayMessageEncodingOverhead = 4 + 6 * 32 + 32;
        uint256 messageLength = ((portalCalldataLimit - relayMessageEncodingOverhead) / 32) * 32;
        bytes memory message = new bytes(messageLength);
        for (uint256 i = 0; i < message.length; i++) {
            message[i] = 0x01;
        }

        uint64 baseGas = l2CrossDomainMessenger.baseGas(message, minGasLimit);
        address target = address(new GasBurner(type(uint32).max));
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), alice, target, 0, minGasLimit, message
        );

        assertEq(encoded.length, relayMessageEncodingOverhead + message.length);
        assertLe(encoded.length, portalCalldataLimit);
        assertGt(encoded.length + 32, portalCalldataLimit);

        StorageSlot memory slot = ForgeArtifacts.getSlot("OptimismPortal2", "l2Sender");
        bytes32 l2SenderSlot = bytes32(slot.slot);
        bytes32 l2Sender = bytes32(abi.encode(address(l2CrossDomainMessenger)));
        vm.store(address(optimismPortal2), l2SenderSlot, l2Sender);
        vm.prank(address(optimismPortal2));

        (bool success,) = address(l1CrossDomainMessenger).call{ gas: baseGas }(encoded);
        assertTrue(success, "L1CrossDomainMessenger call should not fail");

        bool inFailedMessages = l1CrossDomainMessenger.failedMessages(keccak256(encoded));
        bool inSuccessfulMessages = l1CrossDomainMessenger.successfulMessages(keccak256(encoded));
        assertTrue(
            inFailedMessages || inSuccessfulMessages, "message should be in either failed or successful messages"
        );
    }

    /// @notice Tests that `relayMessage` has enough base gas to finish when relaying value to a cold,
    ///         EIP-7702 delegated target that consumes all forwarded gas. The value transfer and the
    ///         target and delegate accesses are paid from the relay's reserved gas.
    ///         Not isolated: isolation treats the gas limit as a transaction's, which must cover
    ///         intrinsic gas. The relay is OptimismPortal2's internal call, whose gas limit never
    ///         has to.
    /// forge-config: default.isolate = false
    function test_relayMessage_valueToColdDelegatedTarget_succeeds() external {
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        uint256 value = 1 ether;
        bytes memory message = hex"";
        uint64 baseGas = l2CrossDomainMessenger.baseGas(message, 0);
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), alice, delegatedTarget, value, 0, message
        );

        StorageSlot memory slot = ForgeArtifacts.getSlot("OptimismPortal2", "l2Sender");
        vm.store(address(optimismPortal2), bytes32(slot.slot), bytes32(abi.encode(address(l2CrossDomainMessenger))));
        vm.deal(address(optimismPortal2), value);
        vm.prank(address(optimismPortal2));

        (bool success,) = address(l1CrossDomainMessenger).call{ gas: baseGas, value: value }(encoded);
        assertTrue(success, "L1CrossDomainMessenger call should not fail");

        bool inFailedMessages = l1CrossDomainMessenger.failedMessages(keccak256(encoded));
        bool inSuccessfulMessages = l1CrossDomainMessenger.successfulMessages(keccak256(encoded));
        assertTrue(
            inFailedMessages || inSuccessfulMessages, "message should be in either failed or successful messages"
        );
    }

    /// @notice Tests that `baseGas` budgets the relay with the L1 relay constants.
    function testFuzz_baseGas_usesRemoteRelayConstants_succeeds(bytes calldata _message, uint32 _minGasLimit) external {
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        uint64 executionGas = l2CrossDomainMessenger.REMOTE_RELAY_CONSTANT_OVERHEAD()
            + l2CrossDomainMessenger.REMOTE_RELAY_CALL_OVERHEAD() + l2CrossDomainMessenger.REMOTE_RELAY_RESERVED_GAS()
            + l2CrossDomainMessenger.REMOTE_RELAY_GAS_CHECK_BUFFER()
            + ((_minGasLimit * l2CrossDomainMessenger.MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR())
                / l2CrossDomainMessenger.MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR());
        uint64 totalMessageSize = uint64(_message.length + l2CrossDomainMessenger.ENCODING_OVERHEAD());
        uint64 expected = l2CrossDomainMessenger.TX_BASE_GAS()
            + uint64(
                Math.max(
                    executionGas + totalMessageSize * l2CrossDomainMessenger.MIN_GAS_CALLDATA_OVERHEAD(),
                    totalMessageSize * l2CrossDomainMessenger.FLOOR_CALLDATA_OVERHEAD()
                )
            );

        assertEq(l2CrossDomainMessenger.baseGas(_message, _minGasLimit), expected);
    }

    /// @notice Tests that the relay constants are the L2 (local) and L1 (remote) `GasConstants` values, and
    ///         that the legacy relay getters return the remote ones.
    function test_baseGas_relayConstants_succeeds() external {
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        assertEq(l2CrossDomainMessenger.LOCAL_RELAY_RESERVED_GAS(), GasConstants.L2_RELAY_RESERVED_GAS);
        assertEq(l2CrossDomainMessenger.LOCAL_RELAY_GAS_CHECK_BUFFER(), GasConstants.L2_RELAY_GAS_CHECK_BUFFER);
        assertEq(l2CrossDomainMessenger.REMOTE_RELAY_CONSTANT_OVERHEAD(), GasConstants.L1_RELAY_CONSTANT_OVERHEAD);
        assertEq(l2CrossDomainMessenger.REMOTE_RELAY_CALL_OVERHEAD(), GasConstants.L1_RELAY_CALL_OVERHEAD);
        assertEq(l2CrossDomainMessenger.REMOTE_RELAY_RESERVED_GAS(), GasConstants.L1_RELAY_RESERVED_GAS);
        assertEq(l2CrossDomainMessenger.REMOTE_RELAY_GAS_CHECK_BUFFER(), GasConstants.L1_RELAY_GAS_CHECK_BUFFER);
        assertEq(
            l2CrossDomainMessenger.RELAY_CONSTANT_OVERHEAD(), l2CrossDomainMessenger.REMOTE_RELAY_CONSTANT_OVERHEAD()
        );
        assertEq(l2CrossDomainMessenger.RELAY_CALL_OVERHEAD(), l2CrossDomainMessenger.REMOTE_RELAY_CALL_OVERHEAD());
        assertEq(l2CrossDomainMessenger.RELAY_RESERVED_GAS(), l2CrossDomainMessenger.REMOTE_RELAY_RESERVED_GAS());
        assertEq(
            l2CrossDomainMessenger.RELAY_GAS_CHECK_BUFFER(), l2CrossDomainMessenger.REMOTE_RELAY_GAS_CHECK_BUFFER()
        );
    }
}
