// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract FakeL2StandardERC20 {

    address public immutable l1Token;
    address public immutable l2Bridge;

    constructor(address _l1Token, address _l2Bridge) {
        l1Token = _l1Token;
        l2Bridge = _l2Bridge;
    }

    // Burn will be called by the L2 Bridge to burn the tokens we are bridging to L1
    function burn(address, uint256) external {}
}
