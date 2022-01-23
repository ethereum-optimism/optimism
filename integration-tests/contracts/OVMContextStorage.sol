// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import {OVMContext} from "./OVMContext.sol";

contract OVMContextStorage is OVMContext {
    mapping(uint256 => uint256) public l1BlockNumbers;
    mapping(uint256 => uint256) public blockNumbers;
    mapping(uint256 => uint256) public timestamps;
    mapping(uint256 => uint256) public difficulty;
    mapping(uint256 => address) public coinbases;
    uint256 public index = 0;

    fallback() external {
        l1BlockNumbers[index] = getCurrentL1BlockNumber();
        blockNumbers[index] = getCurrentBlockNumber();
        timestamps[index] = getCurrentBlockTimestamp();
        difficulty[index] = block.difficulty;
        coinbases[index] = block.coinbase;
        index++;
    }
}
