//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title Burner
 * @dev This contract is used to remove ETH from
 * the L2 circulating supply as it is withdrawn.
 */
contract Burner {
    constructor() payable {
        selfdestruct(payable(address(this)));
    }
}
