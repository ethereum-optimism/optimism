//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Testing utilities */
import { DSTest } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";

contract CommonTest is DSTest {
    Vm vm = Vm(HEVM_ADDRESS);
    address immutable ZERO_ADDRESS = address(0);
    address immutable NON_ZERO_ADDRESS = address(1);
    uint256 immutable NON_ZERO_VALUE = 100;
    uint256 immutable ZERO_VALUE = 0;
    uint64 immutable NON_ZERO_GASLIMIT = 50000;
    bytes32 nonZeroHash = keccak256(abi.encode("NON_ZERO"));
    bytes NON_ZERO_DATA = hex"0000111122223333444455556666777788889999aaaabbbbccccddddeeeeffff0000";
}
