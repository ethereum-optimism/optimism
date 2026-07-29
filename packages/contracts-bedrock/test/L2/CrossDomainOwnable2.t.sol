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

/// @title XDomainSetter2
/// @notice A test contract that extends `CrossDomainOwnable2` to test ownership functionality.
contract XDomainSetter2 is CrossDomainOwnable2 {
    uint256 public value;

    function set(uint256 _value) external onlyOwner {
        value = _value;
    }
}

/// @title CrossDomainOwnable2_TestInit
/// @notice Reusable test initialization for `CrossDomainOwnable2` tests.
abstract contract CrossDomainOwnable2_TestInit is CommonTest {
    XDomainSetter2 setter;

    function setUp() public virtual override {
        super.setUp();
        vm.prank(alice);
        setter = new XDomainSetter2();
    }
}

/// @title CrossDomainOwnable2_RevertConditions_Test
/// @notice Tests various revert conditions for the `CrossDomainOwnable2` contract.
contract CrossDomainOwnable2_RevertConditions_Test is CrossDomainOwnable2_TestInit {
    /// @notice Tests that the `onlyOwner` modifier reverts when the caller is not the messenger.
    function test_onlyOwner_notMessenger_reverts() external {
        vm.expectRevert("CrossDomainOwnable2: caller is not the messenger");
        setter.set(1);
    }

    /// @notice Tests that the `onlyOwner` modifier reverts when not called by the owner.
    function test_onlyOwner_notOwner_reverts() external {
        // set the xDomainMsgSender storage slot
        bytes32 key = bytes32(uint256(204));
        bytes32 value = Bytes32AddressLib.fillLast12Bytes(address(alice));
        vm.store(address(l2CrossDomainMessenger), key, value);

        vm.prank(address(l2CrossDomainMessenger));
        vm.expectRevert("CrossDomainOwnable2: caller is not the owner");
        setter.set(1);
    }

    /// @notice Tests that the `onlyOwner` modifier causes the relayed message to fail.
    function test_onlyOwner_notOwner2_reverts() external {
        uint240 nonce = 0;
        address sender = bob;
        address target = address(setter);
        uint256 value = 0;
        uint256 minGasLimit = 0;
        bytes memory message = abi.encodeCall(XDomainSetter2.set, (1));

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(nonce, 1), sender, target, value, minGasLimit, message
        );

        // It should be a failed message. The revert is caught, so we cannot expectRevert here.
        vm.expectEmit(true, true, true, true);
        emit FailedRelayedMessage(hash);

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(nonce, 1), sender, target, value, minGasLimit, message
        );

        assertEq(setter.value(), 0);
    }
}

/// @title CrossDomainOwnable2_SuccessConditions_Test
/// @notice Tests successful operations of the `CrossDomainOwnable2` contract.
contract CrossDomainOwnable2_SuccessConditions_Test is CrossDomainOwnable2_TestInit {
    /// @notice Tests that the `onlyOwner` modifier succeeds when called by the messenger.
    function test_onlyOwner_succeeds() external {
        address owner = setter.owner();

        // Simulate the L2 execution where the call is coming from the L1CrossDomainMessenger
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(1, 1), owner, address(setter), 0, 0, abi.encodeCall(XDomainSetter2.set, (2))
        );

        assertEq(setter.value(), 2);
    }
}
