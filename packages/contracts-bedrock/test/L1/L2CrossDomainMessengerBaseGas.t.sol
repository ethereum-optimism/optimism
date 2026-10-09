// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";

/// @title L2CrossDomainMessenger_BaseGas_Test
/// @notice Tests L2 message base gas against L1 execution costs.
contract L2CrossDomainMessenger_BaseGas_Test is CommonTest {
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
        uint32 _gasToUse
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
            0, // Value doesn't matter
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
        vm.prank(address(optimismPortal2));

        // Trigger the L1CrossDomainMessenger.
        // Should NOT fail.
        (bool success,) = address(l1CrossDomainMessenger).call{ gas: gasSupplied }(encoded);
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
}
