// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";

/// @title L1CrossDomainMessenger_BaseGas_Test
/// @notice Tests L1 message base gas against L2 execution costs.
contract L1CrossDomainMessenger_BaseGas_Test is CommonTest {
    /// @notice Tests that the `relayMessage` function on L2 will always succeed for any potential
    ///         message, regardless of the size of the message, the minimum gas limit, or the
    ///         amount of gas used by the target contract.
    function testFuzz_relayMessage_baseGasSufficient_succeeds(
        uint24 _messageLength,
        uint32 _minGasLimit,
        uint32 _gasToUse
    )
        external
    {
        // Skip if this is a fork test, won't work.
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        // Keep the fuzz input bounded so the test remains fast. A deterministic regression test
        // covers near-max payloads.
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
        uint64 baseGas = l1CrossDomainMessenger.baseGas(_message, _minGasLimit);

        // Deploy a gas burner.
        address target = address(new GasBurner(_gasToUse));

        // Encode the message.
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            alice, // Sender doesn't matter
            target,
            0, // Value doesn't matter
            _minGasLimit,
            _message
        );

        uint256 tokens = calldataTokens(encoded);

        // Base gas must always be sufficient to cover the floor cost from EIP-7623.
        assertGt(baseGas, 21000 + tokens * 10);

        // Actual gas on L2 will be the base gas minus the intrinsic gas cost. Note that even after
        // EIP-7623, we still deduct 21k + 16 gas per calldata token from the gas limit before
        // execution happens. After execution, if the message didn't spend enough in execution gas,
        // the EVM will floor the cost of the transaction to 21k + 40 gas per calldata token.
        // TODO(#14609): This is a hand-written estimate. Isolation bills intrinsic gas to the test
        // rather than deducting it from the call's gas, so a fork that reprices intrinsic or
        // calldata gas would not fail this test.
        uint256 gasSupplied = baseGas - (21000 + tokens * 4);

        // We'll trigger the L2CrossDomainMessenger as if we're the L1CrossDomainMessenger
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        vm.prank(caller);

        // Trigger the L2CrossDomainMessenger.
        // Should NOT fail.
        (bool success,) = address(l2CrossDomainMessenger).call{ gas: gasSupplied }(encoded);
        assertTrue(success, "L2CrossDomainMessenger call should not fail");

        // Message should either be in the failed or successful messages mapping.
        bool inFailedMessages = l2CrossDomainMessenger.failedMessages(keccak256(encoded));
        bool inSuccessfulMessages = l2CrossDomainMessenger.successfulMessages(keccak256(encoded));
        assertTrue(
            inFailedMessages || inSuccessfulMessages, "message should be in either failed or successful messages"
        );
    }

    /// @notice Tests that `relayMessage` has enough base gas to finish when relaying a near-max
    ///         L1 => L2 payload to a target that consumes all forwarded gas.
    function test_relayMessage_nearMaxPayloadAllGasTarget_succeeds() external {
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        // Use the PoC's _minGasLimit value referenced from client-pod#609:
        // https://github.com/ethereum-optimism/client-pod/issues/609
        uint32 minGasLimit = 50_000;

        // Largest 32-byte-aligned user message that fits under the portal's 120k calldata cap once
        // encoded as relayMessage(uint256,address,address,uint256,uint256,bytes). The fixed
        // encoding overhead is 4 bytes for the selector, 6 ABI words for the nonce, sender,
        // target, value, minGasLimit, and message offset, and 1 ABI word for the dynamic bytes
        // length. The message length is rounded down to a 32-byte multiple because dynamic bytes
        // data is ABI word-padded.
        uint256 portalCalldataLimit = 120_000;
        uint256 relayMessageEncodingOverhead = 4 + 6 * 32 + 32;
        uint256 messageLength = ((portalCalldataLimit - relayMessageEncodingOverhead) / 32) * 32;
        bytes memory message = new bytes(messageLength);
        for (uint256 i = 0; i < message.length; i++) {
            message[i] = 0x01;
        }

        uint64 baseGas = l1CrossDomainMessenger.baseGas(message, minGasLimit);
        address target = address(new GasBurner(type(uint32).max));
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), alice, target, 0, minGasLimit, message
        );

        assertEq(encoded.length, relayMessageEncodingOverhead + message.length);
        assertLe(encoded.length, portalCalldataLimit);
        assertGt(encoded.length + 32, portalCalldataLimit);

        uint256 tokens = calldataTokens(encoded);
        // The message body is non-zero, but the ABI-encoded selector and fixed-size arguments
        // still include zero bytes.
        assertLt(tokens, encoded.length * 4);

        // Actual gas on L2 is the deposited gas limit minus L2 intrinsic gas.
        uint256 gasSupplied = baseGas - (21_000 + tokens * 4);

        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        vm.prank(caller);

        (bool success,) = address(l2CrossDomainMessenger).call{ gas: gasSupplied }(encoded);
        assertTrue(success, "L2CrossDomainMessenger call should not fail");

        bool inFailedMessages = l2CrossDomainMessenger.failedMessages(keccak256(encoded));
        bool inSuccessfulMessages = l2CrossDomainMessenger.successfulMessages(keccak256(encoded));
        assertTrue(
            inFailedMessages || inSuccessfulMessages, "message should be in either failed or successful messages"
        );
    }

    /// @notice Tests that `baseGas` budgets the relay with the L2 relay constants.
    function testFuzz_baseGas_usesRemoteRelayConstants_succeeds(
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
        view
    {
        uint64 executionGas = l1CrossDomainMessenger.REMOTE_RELAY_CONSTANT_OVERHEAD()
            + l1CrossDomainMessenger.REMOTE_RELAY_CALL_OVERHEAD() + l1CrossDomainMessenger.REMOTE_RELAY_RESERVED_GAS()
            + l1CrossDomainMessenger.REMOTE_RELAY_GAS_CHECK_BUFFER()
            + ((_minGasLimit * l1CrossDomainMessenger.MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR())
                / l1CrossDomainMessenger.MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR());
        uint64 totalMessageSize = uint64(_message.length + l1CrossDomainMessenger.ENCODING_OVERHEAD());
        uint64 expected = l1CrossDomainMessenger.TX_BASE_GAS()
            + uint64(
                Math.max(
                    executionGas + totalMessageSize * l1CrossDomainMessenger.MIN_GAS_CALLDATA_OVERHEAD(),
                    totalMessageSize * l1CrossDomainMessenger.FLOOR_CALLDATA_OVERHEAD()
                )
            );

        assertEq(l1CrossDomainMessenger.baseGas(_message, _minGasLimit), expected);
    }

    /// @notice Tests the relay constant values and that the legacy relay getters return the L2 relay constants.
    function test_baseGas_relayConstants_succeeds() external view {
        assertEq(l1CrossDomainMessenger.LOCAL_RELAY_RESERVED_GAS(), 135_000);
        assertEq(l1CrossDomainMessenger.LOCAL_RELAY_GAS_CHECK_BUFFER(), 5_000);
        assertEq(l1CrossDomainMessenger.REMOTE_RELAY_CONSTANT_OVERHEAD(), 200_000);
        assertEq(l1CrossDomainMessenger.REMOTE_RELAY_CALL_OVERHEAD(), 40_000);
        assertEq(l1CrossDomainMessenger.REMOTE_RELAY_RESERVED_GAS(), 40_000);
        assertEq(l1CrossDomainMessenger.REMOTE_RELAY_GAS_CHECK_BUFFER(), 5_000);
        assertEq(
            l1CrossDomainMessenger.RELAY_CONSTANT_OVERHEAD(), l1CrossDomainMessenger.REMOTE_RELAY_CONSTANT_OVERHEAD()
        );
        assertEq(l1CrossDomainMessenger.RELAY_CALL_OVERHEAD(), l1CrossDomainMessenger.REMOTE_RELAY_CALL_OVERHEAD());
        assertEq(l1CrossDomainMessenger.RELAY_RESERVED_GAS(), l1CrossDomainMessenger.REMOTE_RELAY_RESERVED_GAS());
        assertEq(
            l1CrossDomainMessenger.RELAY_GAS_CHECK_BUFFER(), l1CrossDomainMessenger.REMOTE_RELAY_GAS_CHECK_BUFFER()
        );
    }
}
