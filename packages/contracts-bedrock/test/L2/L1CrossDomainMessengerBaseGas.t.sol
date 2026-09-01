// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Encoding } from "src/libraries/Encoding.sol";

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
        // TODO(#14609): Update this test to use default.isolate = true once a new stable Foundry
        // release is available that includes #9904. That will allow us to use this test to check
        // for changes to the EVM itself that might cause our gas formula to be incorrect.

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

        // Count the number of non-zero bytes in the message.
        uint256 zeroBytesInCalldata = 0;
        uint256 nonzeroBytesInCalldata = 0;
        for (uint256 i = 0; i < encoded.length; i++) {
            if (encoded[i] != bytes1(0)) {
                nonzeroBytesInCalldata++;
            } else {
                zeroBytesInCalldata++;
            }
        }

        // Base gas must always be sufficient to cover the floor cost from EIP-7623.
        assertGt(baseGas, 21000 + ((zeroBytesInCalldata + nonzeroBytesInCalldata * 4) * 10));

        // Actual gas on L2 will be the base gas minus the intrinsic gas cost. Note that even after
        // EIP-7623, we still deduct 21k + 16 gas per calldata token from the gas limit before
        // execution happens. After execution, if the message didn't spend enough in execution gas,
        // the EVM will floor the cost of the transaction to 21k + 40 gas per calldata token.
        uint256 gasSupplied = baseGas - (21000 + ((zeroBytesInCalldata + nonzeroBytesInCalldata * 4) * 4));

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
}
