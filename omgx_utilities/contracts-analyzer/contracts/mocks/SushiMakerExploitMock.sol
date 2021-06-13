// SPDX-License-Identifier: MIT

pragma solidity 0.6.12;

import "../SushiMaker.sol";

contract SushiMakerExploitMock {
    SushiMaker public immutable sushiMaker;
    constructor (address _sushiMaker) public{
        sushiMaker = SushiMaker(_sushiMaker);
    } 
    function convert(address token0, address token1) external {
        sushiMaker.convert(token0, token1);
    }
}