// SPDX-License-Identifier: MIT

pragma solidity 0.6.12;

import "../SushiMakerKashi.sol";

contract SushiMakerKashiExploitMock {
    SushiMakerKashi public immutable sushiMaker;
    
    constructor(address _sushiMaker) public {
        sushiMaker = SushiMakerKashi(_sushiMaker);
    } 
  
    function convert(IKashiWithdrawFee kashiPair) external {
        sushiMaker.convert(kashiPair);
    }
}
