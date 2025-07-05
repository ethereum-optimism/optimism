// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC20}   from "openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";
import {Ownable} from "openzeppelin-contracts/contracts/access/Ownable.sol";

contract ParamToken is ERC20, Ownable {
    uint256 public constant CAP = 1_000_000_000 * 1e18; // 1 b PARAM hard cap

    constructor() ERC20("PARAM Token", "PARAM") Ownable(msg.sender) {}

    /// @dev one-off mint; owner is the deploy-time multisig
    function mint(address to, uint256 amount) external onlyOwner {
        require(totalSupply() + amount <= CAP, "CAP exceeded");
        _mint(to, amount);
    }
}
