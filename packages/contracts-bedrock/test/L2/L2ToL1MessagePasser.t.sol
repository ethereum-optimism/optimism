// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

/// @title L2ToL1MessagePasser_InitiateWithdrawal_Test
/// @notice Tests the `initiateWithdrawal` function of the `L2ToL1MessagePasser` contract.
contract L2ToL1MessagePasser_InitiateWithdrawal_Test is CommonTest {
    /// @notice Tests that `initiateWithdrawal` succeeds and correctly sets the state of the
    ///         message passer for the withdrawal hash.
    function testFuzz_initiateWithdrawal_succeeds(
        address _sender,
        address _target,
        uint256 _value,
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
                value: _value,
                gasLimit: _gasLimit,
                data: _data
            })
        );

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, _sender, _target, _value, _gasLimit, _data, withdrawalHash);

        vm.deal(_sender, _value);
        vm.prank(_sender);
        l2ToL1MessagePasser.initiateWithdrawal{ value: _value }(_target, _gasLimit, _data);

        assertEq(l2ToL1MessagePasser.sentMessages(withdrawalHash), true);
        assertEq(l2ToL1MessagePasser.messageNonce(), nonce + 1);

        bytes32 slot = keccak256(bytes.concat(withdrawalHash, bytes32(0)));

        assertEq(vm.load(address(l2ToL1MessagePasser), slot), bytes32(uint256(1)));
    }

}

/// @title L2ToL1MessagePasser_Burn_Test
/// @notice Tests the `burn` function of the `L2ToL1MessagePasser` contract.
contract L2ToL1MessagePasser_Burn_Test is CommonTest {
    /// @notice Tests that `burn` succeeds and destroys the ETH held in the contract.
    function testFuzz_burn_succeeds(uint256 _value, address _target, uint256 _gasLimit, bytes memory _data) external {
        vm.deal(address(this), _value);

        l2ToL1MessagePasser.initiateWithdrawal{ value: _value }({ _target: _target, _gasLimit: _gasLimit, _data: _data });

        assertEq(address(l2ToL1MessagePasser).balance, _value);
        
        vm.expectEmit(address(l2ToL1MessagePasser));
        emit WithdrawerBalanceBurnt(_value);
        l2ToL1MessagePasser.burn();

        // The Withdrawer should have no balance
        assertEq(address(l2ToL1MessagePasser).balance, 0);
    }
}
