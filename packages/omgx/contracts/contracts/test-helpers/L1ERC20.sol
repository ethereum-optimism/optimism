// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;

import '@openzeppelin/contracts/token/ERC20/ERC20.sol';

/**
 * @title ERC20
 * @dev A super simple ERC20 implementation!
 */
contract L1ERC20 is ERC20 {
    constructor(
        uint256 _initialSupply,
        string memory _tokenName,
        string memory _tokenSymbol
    )
        public
        ERC20(
            _tokenName,
            _tokenSymbol
        )
    {
        _mint(msg.sender, _initialSupply);
    }
}
