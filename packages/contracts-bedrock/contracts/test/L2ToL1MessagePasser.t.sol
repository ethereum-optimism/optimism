// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { Types } from "../libraries/Types.sol";
import { Hashing } from "../libraries/Hashing.sol";

contract L2ToL1MessagePasserTest is CommonTest {
    L2ToL1MessagePasser messagePasser;

    event MessagePassed(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data,
        bytes32 withdrawalHash
    );

    event WithdrawerBalanceBurnt(uint256 indexed amount);

    function setUp() public virtual override {
        super.setUp();
        messagePasser = new L2ToL1MessagePasser();
    }

    function testFuzz_initiateWithdrawal_succeeds(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external {
        uint256 nonce = messagePasser.messageNonce();

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

        vm.expectEmit(true, true, true, true);
        emit MessagePassed(nonce, _sender, _target, _value, _gasLimit, _data, withdrawalHash);

        vm.deal(_sender, _value);
        vm.prank(_sender);
        messagePasser.initiateWithdrawal{ value: _value }(_target, _gasLimit, _data);

        assertEq(messagePasser.sentMessages(withdrawalHash), true);

        bytes32 slot = keccak256(bytes.concat(withdrawalHash, bytes32(0)));

        assertEq(vm.load(address(messagePasser), slot), bytes32(uint256(1)));
    }

    // Test: initiateWithdrawal should emit the correct log when called by a contract
    function test_initiateWithdrawal_fromContract_succeeds() external {
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction(
                messagePasser.messageNonce(),
                address(this),
                address(4),
                100,
                64000,
                hex""
            )
        );

        vm.expectEmit(true, true, true, true);
        emit MessagePassed(
            messagePasser.messageNonce(),
            address(this),
            address(4),
            100,
            64000,
            hex"",
            withdrawalHash
        );

        vm.deal(address(this), 2**64);
        messagePasser.initiateWithdrawal{ value: 100 }(address(4), 64000, hex"");
    }

    // Test: initiateWithdrawal should emit the correct log when called by an EOA
    function test_initiateWithdrawal_fromEOA_succeeds() external {
        uint256 gasLimit = 64000;
        address target = address(4);
        uint256 value = 100;
        bytes memory data = hex"ff";
        uint256 nonce = messagePasser.messageNonce();

        // EOA emulation
        vm.prank(alice, alice);
        vm.deal(alice, 2**64);
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction(nonce, alice, target, value, gasLimit, data)
        );

        vm.expectEmit(true, true, true, true);
        emit MessagePassed(nonce, alice, target, value, gasLimit, data, withdrawalHash);

        messagePasser.initiateWithdrawal{ value: value }(target, gasLimit, data);

        // the sent messages mapping is filled
        assertEq(messagePasser.sentMessages(withdrawalHash), true);
        // the nonce increments
        assertEq(nonce + 1, messagePasser.messageNonce());
    }

    // Test: burn should destroy the ETH held in the contract
    function test_burn_succeeds() external {
        messagePasser.initiateWithdrawal{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            NON_ZERO_GASLIMIT,
            NON_ZERO_DATA
        );

        assertEq(address(messagePasser).balance, NON_ZERO_VALUE);
        vm.expectEmit(true, false, false, false);
        emit WithdrawerBalanceBurnt(NON_ZERO_VALUE);
        messagePasser.burn();

        // The Withdrawer should have no balance
        assertEq(address(messagePasser).balance, 0);
    }
}
