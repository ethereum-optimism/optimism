// SPDX-License-Identifier: MIT
pragma solidity >=0.5.16 <0.8.0;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import './UniswapV2ERC20.sol';

contract L2StandardERC20 is UniswapV2ERC20, Ownable {
    address public l1Token;

    /**
     * @param _l1Token Address of the corresponding L1 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    constructor(
        address _l1Token,
        string memory _name,
        string memory _symbol
    )
        UniswapV2ERC20(_name, _symbol){
        l1Token = _l1Token;
    }

    function mint(address _to, uint256 _value) public onlyOwner {
        _mint(_to, _value);
    }

    function burn(address _from, uint256 _value) public onlyOwner {
        _burn(_from, _value);
    }
}
