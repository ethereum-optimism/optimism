// SPDX-License-Identifier: MIT

pragma solidity >=0.7.0;

contract OVMContextStorage {
    mapping (uint256 => uint256) public blockNumbers;
    mapping (uint256 => uint256) public timestamps;
    mapping (uint256 => uint256) public difficulty;
    mapping (uint256 => address) public coinbases;
    uint256 public index = 0;

    fallback() external {
        blockNumbers[index] = block.number;
        timestamps[index] = block.timestamp;
        difficulty[index] = block.difficulty;
        coinbases[index] = block.coinbase;
        index++;
    }
}
