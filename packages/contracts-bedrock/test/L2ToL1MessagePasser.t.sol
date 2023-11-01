// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/CommonTest.t.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Target contract
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";

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

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        messagePasser = new L2ToL1MessagePasser();
    }

    /// @dev Tests that `initiateWithdrawal` succeeds and correctly sets the state
    ///      of the message passer for the withdrawal hash.
    function testFuzz_initiateWithdrawal_succeeds(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
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

    /// @dev Tests that `initiateWithdrawal` succeeds and emits the correct MessagePassed
    ///      log when called by a contract.
    function testFuzz_initiateWithdrawal_fromContract_succeeds(
        address _target,
        uint256 _gasLimit,
        uint256 _value,
        bytes memory _data
    )
        external
    {
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: messagePasser.messageNonce(),
                sender: address(this),
                target: _target,
                value: _value,
                gasLimit: _gasLimit,
                data: _data
            })
        );

        vm.expectEmit(address(messagePasser));
        emit MessagePassed(
            messagePasser.messageNonce(), address(this), _target, _value, _gasLimit, _data, withdrawalHash
        );

        vm.deal(address(this), _value);
        messagePasser.initiateWithdrawal{ value: _value }(_target, _gasLimit, _data);
    }

    /// @dev Tests that `initiateWithdrawal` succeeds and emits the correct MessagePassed
    ///      log when called by an EOA.
    function testFuzz_initiateWithdrawal_fromEOA_succeeds(
        uint256 _gasLimit,
        address _target,
        uint256 _value,
        bytes memory _data
    )
        external
    {
        uint256 nonce = messagePasser.messageNonce();

        // EOA emulation
        vm.prank(alice, alice);
        vm.deal(alice, _value);
        bytes32 withdrawalHash =
            Hashing.hashWithdrawal(Types.WithdrawalTransaction(nonce, alice, _target, _value, _gasLimit, _data));

        vm.expectEmit(address(messagePasser));
        emit MessagePassed(nonce, alice, _target, _value, _gasLimit, _data, withdrawalHash);

        messagePasser.initiateWithdrawal{ value: _value }({ _target: _target, _gasLimit: _gasLimit, _data: _data });

        // the sent messages mapping is filled
        assertEq(messagePasser.sentMessages(withdrawalHash), true);
        // the nonce increments
        assertEq(nonce + 1, messagePasser.messageNonce());
    }

    /// @dev Tests that `burn` succeeds and destroys the ETH held in the contract.
    function testFuzz_burn_succeeds(uint256 _value, address _target, uint256 _gasLimit, bytes memory _data) external {
        vm.deal(address(this), _value);

        messagePasser.initiateWithdrawal{ value: _value }({ _target: _target, _gasLimit: _gasLimit, _data: _data });

        assertEq(address(messagePasser).balance, _value);
        vm.expectEmit(true, false, false, false);
        emit WithdrawerBalanceBurnt(_value);
        messagePasser.burn();

        // The Withdrawer should have no balance
        assertEq(address(messagePasser).balance, 0);
    }
}
