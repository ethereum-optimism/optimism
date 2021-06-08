// SPDX-License-Identifier: MIT
pragma solidity >=0.5.16 <0.8.0;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

import './IL2StandardERC20.sol';

contract L2StandardERC20 is IL2StandardERC20, ERC20, Ownable {
    address public override l1Token;

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
        ERC20(_name, _symbol) {
        l1Token = _l1Token;
    }

    function supportsInterface(bytes4 _interfaceId) public override pure returns (bool) {
        return _interfaceId == 0x01ffc9a7 || _interfaceId == 0x1d1d8b63;
    }

    function mint(address _to, uint256 _amount) public override onlyOwner {
        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    function burn(address _from, uint256 _amount) public override onlyOwner {
        _burn(_from, _amount);

        emit Burn(_from, _amount);
    }
}
