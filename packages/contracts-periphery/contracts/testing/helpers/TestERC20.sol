// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ERC20 } from "@rari-capital/solmate/src/tokens/ERC20.sol";

contract TestERC20 is ERC20 {
    constructor() ERC20("TEST", "TST", 18) {}

    function mint(address to, uint256 value) public {
        _mint(to, value);
    }
}
