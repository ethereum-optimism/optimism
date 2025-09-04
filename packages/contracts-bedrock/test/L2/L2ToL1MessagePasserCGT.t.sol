// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Interfaces
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";

/// @title L2ToL1MessagePasserCGT_TestInit
/// @notice Tests the `L2ToL1MessagePasser` contract with a custom gas token enabled.
contract L2ToL1MessagePasserCGT_TestInit is CommonTest {
    /// @notice Sets up the test suite with custom gas token enabled.
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title L2ToL1MessagePasserCGT_InitiateWithdrawal_Test
/// @notice Tests the `initiateWithdrawal` function of the `L2ToL1MessagePasser` contract with
///         custom gas token enabled.
contract L2ToL1MessagePasserCGT_InitiateWithdrawal_Test is L2ToL1MessagePasserCGT_TestInit {
    /// @notice Tests that `initiateWithdrawal` succeeds and correctly sets the state of the
    ///         message passer for the withdrawal hash.
    function testFuzz_initiateWithdrawal_withZeroValue_succeeds(
        address _sender,
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: _sender,
                target: _target,
                value: 0,
                gasLimit: _gasLimit,
                data: _data
            })
        );

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, _sender, _target, 0, _gasLimit, _data, withdrawalHash);

        vm.prank(_sender);
        l2ToL1MessagePasser.initiateWithdrawal{ value: 0 }(_target, _gasLimit, _data);

        assertEq(l2ToL1MessagePasser.sentMessages(withdrawalHash), true);

        bytes32 slot = keccak256(bytes.concat(withdrawalHash, bytes32(0)));

        assertEq(vm.load(address(l2ToL1MessagePasser), slot), bytes32(uint256(1)));
    }

    /// @notice Tests that `initiateWithdrawal` fails when called with value and custom gas token
    ///         is enabled.
    function testFuzz_initiateWithdrawal_withValue_fails(address _randomAddress, uint256 _value) external {
        // Set initial state
        _value = bound(_value, 1, type(uint256).max);
        vm.deal(_randomAddress, _value);

        // Expect revert with NotAllowedOnCGTMode
        vm.prank(_randomAddress);
        vm.expectRevert(IL2ToL1MessagePasser.L2ToL1MessagePasser_NotAllowedOnCGTMode.selector);
        l2ToL1MessagePasser.initiateWithdrawal{ value: _value }({ _target: address(0), _gasLimit: 1, _data: "" });
    }
}
