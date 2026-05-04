// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Bytes32AddressLib } from "@rari-capital/solmate/src/utils/Bytes32AddressLib.sol";

// Target contract dependencies
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Target contract
import { CrossDomainOwnable2 } from "src/L2/CrossDomainOwnable2.sol";

/// @title CrossDomainOwnable2_Setter_Harness
/// @notice Test harness exposing an `onlyOwner`-guarded setter on `CrossDomainOwnable2`.
contract CrossDomainOwnable2_Setter_Harness is CrossDomainOwnable2 {
    uint256 public value;

    function set(uint256 _value) external onlyOwner {
        value = _value;
    }
}

/// @title CrossDomainOwnable2_TestInit
/// @notice Reusable test initialization for `CrossDomainOwnable2` tests.
abstract contract CrossDomainOwnable2_TestInit is CommonTest {
    CrossDomainOwnable2_Setter_Harness setter;

    function setUp() public virtual override {
        super.setUp();
        vm.prank(alice);
        setter = new CrossDomainOwnable2_Setter_Harness();
    }
}

/// @title CrossDomainOwnable2_CheckOwner_Test
/// @notice Tests for the `_checkOwner` override of `CrossDomainOwnable2`.
contract CrossDomainOwnable2_CheckOwner_Test is CrossDomainOwnable2_TestInit {
    /// @notice Tests that `_checkOwner` reverts when the caller is not the L2CrossDomainMessenger.
    function test_checkOwner_notMessenger_reverts() external {
        vm.expectRevert("CrossDomainOwnable2: caller is not the messenger");
        setter.set(1);
    }

    /// @notice Tests that `_checkOwner` reverts when the `xDomainMessageSender` is not the owner.
    function test_checkOwner_notOwner_reverts() external {
        // Set the L2CrossDomainMessenger's `xDomainMsgSender` storage slot.
        bytes32 key = bytes32(uint256(204));
        bytes32 value = Bytes32AddressLib.fillLast12Bytes(address(alice));
        vm.store(address(l2CrossDomainMessenger), key, value);

        vm.prank(address(l2CrossDomainMessenger));
        vm.expectRevert("CrossDomainOwnable2: caller is not the owner");
        setter.set(1);
    }

    /// @notice Tests that messages relayed from a non-owner cause the relayed message to fail.
    function test_checkOwner_notOwnerViaRelay_reverts() external {
        uint240 nonce = 0;
        address sender = bob;
        address target = address(setter);
        uint256 value = 0;
        uint256 minGasLimit = 0;
        bytes memory message = abi.encodeCall(CrossDomainOwnable2_Setter_Harness.set, (1));

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(nonce, 1), sender, target, value, minGasLimit, message
        );

        // The revert is caught by relayMessage, so we cannot expectRevert here.
        vm.expectEmit(true, true, true, true);
        emit FailedRelayedMessage(hash);

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(nonce, 1), sender, target, value, minGasLimit, message
        );

        assertEq(setter.value(), 0);
    }

    /// @notice Tests that `_checkOwner` succeeds when called by the owner via the messenger.
    function test_checkOwner_succeeds() external {
        address owner = setter.owner();

        // Simulate the L2 execution where the call is coming from the L1CrossDomainMessenger.
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(1, 1),
            owner,
            address(setter),
            0,
            0,
            abi.encodeCall(CrossDomainOwnable2_Setter_Harness.set, (2))
        );

        assertEq(setter.value(), 2);
    }
}
