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
    /// @notice Tests that `_checkOwner` reverts for any caller that is not the
    ///         L2CrossDomainMessenger.
    /// @param _caller Random caller address.
    function testFuzz_checkOwner_notMessenger_reverts(address _caller) external {
        vm.assume(_caller != address(l2CrossDomainMessenger));
        vm.prank(_caller);
        vm.expectRevert("CrossDomainOwnable2: caller is not the messenger");
        setter.set(1);
    }

    /// @notice Tests that `_checkOwner` reverts when the messenger's `xDomainMessageSender` is any
    ///         address other than the contract owner.
    /// @param _xDomainSender Random `xDomainMessageSender` value.
    function testFuzz_checkOwner_notOwner_reverts(address _xDomainSender) external {
        vm.assume(_xDomainSender != setter.owner());
        // Set the L2CrossDomainMessenger's `xDomainMsgSender` storage slot.
        bytes32 key = bytes32(uint256(204));
        bytes32 value = Bytes32AddressLib.fillLast12Bytes(_xDomainSender);
        vm.store(address(l2CrossDomainMessenger), key, value);

        vm.prank(address(l2CrossDomainMessenger));
        vm.expectRevert("CrossDomainOwnable2: caller is not the owner");
        setter.set(1);
    }

    /// @notice Tests that messages relayed from any non-owner sender cause the relayed message to
    ///         fail without mutating state.
    /// @param _sender Random L1 sender that is not the contract owner.
    function testFuzz_checkOwner_notOwnerViaRelay_reverts(address _sender) external {
        vm.assume(_sender != setter.owner());

        uint240 nonce = 0;
        address target = address(setter);
        uint256 value = 0;
        uint256 minGasLimit = 0;
        bytes memory message = abi.encodeCall(CrossDomainOwnable2_Setter_Harness.set, (1));

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(nonce, 1), _sender, target, value, minGasLimit, message
        );

        // The revert is caught by relayMessage, so we cannot expectRevert here.
        vm.expectEmit(true, true, true, true);
        emit FailedRelayedMessage(hash);

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(nonce, 1), _sender, target, value, minGasLimit, message
        );

        assertEq(setter.value(), 0);
    }

    /// @notice Tests that `_checkOwner` succeeds when the call originates from the owner via the
    ///         L2CrossDomainMessenger for any payload value.
    /// @param _value Random value forwarded to the harness setter.
    function testFuzz_checkOwner_succeeds(uint256 _value) external {
        address owner = setter.owner();

        // Simulate the L2 execution where the call is coming from the L1CrossDomainMessenger.
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(1, 1),
            owner,
            address(setter),
            0,
            0,
            abi.encodeCall(CrossDomainOwnable2_Setter_Harness.set, (_value))
        );

        assertEq(setter.value(), _value);
    }
}
