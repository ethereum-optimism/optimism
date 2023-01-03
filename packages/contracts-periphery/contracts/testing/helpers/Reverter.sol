// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Reverter {
    function doRevert() public pure {
        revert("Reverter reverted");
    }
}
