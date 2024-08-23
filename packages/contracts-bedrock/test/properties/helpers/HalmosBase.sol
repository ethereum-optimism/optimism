// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { Test } from "forge-std/Test.sol";

contract HalmosBase is Test {
    uint256 internal constant CURRENT_CHAIN_ID = 1;
    uint256 internal constant ZERO_AMOUNT = 0;

    address internal remoteToken = address(bytes20(keccak256("remoteToken")));
    string internal name = "SuperchainERC20";
    string internal symbol = "SUPER";
    uint8 internal decimals = 18;

    function eqStrings(string memory a, string memory b) internal pure returns (bool) {
        return keccak256(abi.encode(a)) == keccak256(abi.encode(b));
    }
}
