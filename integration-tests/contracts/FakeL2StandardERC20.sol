// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract FakeL2StandardERC20 {

    address public l1Token;
    constructor(address _l1Token){
        l1Token = _l1Token;
    }

    // Burn will be called by the L2 Bridge to burn the tokens we are bridging to L1
    function burn(address from, uint256 amount) external {
        from; amount;
    }
}
