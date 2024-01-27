// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Setup } from "test/setup/Setup.sol";
import { Events } from "test/setup/Events.sol";
import { FFIInterface } from "test/setup/FFIInterface.sol";

import "src/libraries/DisputeTypes.sol";

/// @title CommonTest
/// @dev An extenstion to `Test` that sets up the optimism smart contracts.
contract CommonTest is Test, Setup, Events {
    address alice;
    address bob;

    bytes32 constant nonZeroHash = keccak256(abi.encode("NON_ZERO"));

    FFIInterface constant ffi = FFIInterface(address(uint160(uint256(keccak256(abi.encode("optimism.ffi"))))));

    function setUp() public virtual override {
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);

        Setup.setUp();
        vm.etch(address(ffi), vm.getDeployedCode("FFIInterface.sol:FFIInterface"));
        vm.label(address(ffi), "FFIInterface");

        // Exclude contracts for the invariant tests
        excludeContract(address(ffi));
        excludeContract(address(deploy));
        excludeContract(address(deploy.cfg()));

        // Make sure the base fee is non zero
        vm.fee(1 gwei);

        // Start @ August 14, 1984 @ block 1M
        vm.warp(461347200);
        vm.roll(1_000_000);

        // Deploy L1
        Setup.L1();
        // Deploy L2
        Setup.L2();
    }

    /// @dev Helper function that wraps `TransactionDeposited` event.
    ///      The magic `0` is the version.
    function emitTransactionDeposited(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        internal
    {
        emit TransactionDeposited(_from, _to, 0, abi.encodePacked(_mint, _value, _gasLimit, _isCreation, _data));
    }

    /// @dev Helper function to propose an output.
    function proposeAnotherOutput() public {
        // Warp forward 1 block
        vm.warp(block.timestamp + 12);
        vm.roll(block.number + 1);

        // Expect the dispute game created event
        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(
            address(0),
            GameTypes.CANNON,
            Claim.wrap(keccak256(abi.encode()))
        );

        disputeGameFactory.create(GameTypes.CANNON, Claim.wrap(keccak256(abi.encode())), abi.encode(block.number));
    }
}
