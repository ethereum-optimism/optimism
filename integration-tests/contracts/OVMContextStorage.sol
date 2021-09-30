// SPDX-License-Identifier: MIT

pragma solidity ^0.8.9;

// Can't do this until the package is published.
//import { iOVM_L1BlockNumber } from "@eth-optimism/contracts/iOVM_L1BlockNumber";

interface iOVM_L1BlockNumber {
    function getL1BlockNumber() external view returns (uint256);
}

contract OVMContextStorage {
    mapping (uint256 => uint256) public l1BlockNumbers;
    mapping (uint256 => uint256) public blockNumbers;
    mapping (uint256 => uint256) public timestamps;
    mapping (uint256 => uint256) public difficulty;
    mapping (uint256 => address) public coinbases;
    uint256 public index = 0;

    fallback() external {
        l1BlockNumbers[index] = iOVM_L1BlockNumber(
            0x4200000000000000000000000000000000000013
        ).getL1BlockNumber();
        blockNumbers[index] = block.number;
        timestamps[index] = block.timestamp;
        difficulty[index] = block.difficulty;
        coinbases[index] = block.coinbase;
        index++;
    }
}
