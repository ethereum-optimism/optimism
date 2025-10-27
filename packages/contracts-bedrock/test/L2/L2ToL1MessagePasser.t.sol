// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title L2ToL1MessagePasser_Version_Test
/// @notice Tests the `version` function of the `L2ToL1MessagePasser` contract.
contract L2ToL1MessagePasser_Version_Test is CommonTest {
    /// @notice Tests that the version follows valid semver format.
    function test_version_validFormat_succeeds() external view {
        SemverComp.parse(l2ToL1MessagePasser.version());
    }
}

/// @title L2ToL1MessagePasser_Receive_Test
/// @notice Tests the `receive` function of the `L2ToL1MessagePasser` contract.
contract L2ToL1MessagePasser_Receive_Test is CommonTest {
    /// @notice Tests that receive() initiates withdrawal with default gas limit.
    function testFuzz_receive_initiatesWithdrawal_succeeds(uint256 _value) external {
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: address(this),
                target: address(this),
                value: _value,
                gasLimit: 100_000, // RECEIVE_DEFAULT_GAS_LIMIT
                data: bytes("")
            })
        );

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, address(this), address(this), _value, 100_000, bytes(""), withdrawalHash);

        vm.deal(address(this), _value);
        (bool success,) = address(l2ToL1MessagePasser).call{ value: _value }("");
        
        assertTrue(success);
        assertEq(l2ToL1MessagePasser.sentMessages(withdrawalHash), true);
        assertEq(l2ToL1MessagePasser.messageNonce(), nonce + 1);
    }
}

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

    /// @notice Tests that `burn` succeeds when contract has zero balance.
    function test_burn_zeroBalance_succeeds() external {
        assertEq(address(l2ToL1MessagePasser).balance, 0);
        
        vm.expectEmit(address(l2ToL1MessagePasser));
        emit WithdrawerBalanceBurnt(0);
        l2ToL1MessagePasser.burn();

        assertEq(address(l2ToL1MessagePasser).balance, 0);
    }
}

/// @title L2ToL1MessagePasser_MessageNonce_Test
/// @notice Tests the `messageNonce` function of the `L2ToL1MessagePasser` contract.
contract L2ToL1MessagePasser_MessageNonce_Test is CommonTest {
    /// @notice Tests that messageNonce encodes version in upper bytes.
    function test_messageNonce_encodesVersion_succeeds() external view {
        uint256 nonce = l2ToL1MessagePasser.messageNonce();
        
        // MESSAGE_VERSION is 1, should be in upper 2 bytes
        // Version is stored in bits 240-255 (upper 2 bytes of uint256)
        uint256 version = nonce >> 240;
        assertEq(version, 1);
    }
}
