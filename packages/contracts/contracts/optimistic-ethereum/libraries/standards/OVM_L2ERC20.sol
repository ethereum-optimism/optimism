// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

// Based on OpenZeppelin's ERC20PresetMinterPauser, with modifications
// https://github.com/OpenZeppelin/openzeppelin-contracts/blob/release-v3.3/contracts/presets/ERC20PresetMinterPauser.sol

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { ERC20Burnable } from "@openzeppelin/contracts/token/ERC20/ERC20Burnable.sol";

 /**
 * @title OVM_L2ERC20
 * @dev An L2 Deposited Token is an L2 representation of funds which were deposited from L1.
 * The TokenGateway contract is the owner, and mints or burns tokens as required when transfers
 * are made to or from L1.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_L2ERC20 is Ownable, ERC20Burnable {

    /**
     * @dev Sets owner to the deployer.
     */
    constructor(
        string memory _name,
        string memory _symbol
    )
        ERC20(_name, _symbol)
        Ownable()
    {}

    /**
     * @dev Creates `amount` new tokens for `to`.
     * @param _to The account to burn from.
     * @param _amount The amount to burn.
     */
    function mint(
        address _to,
        uint256 _amount
    )
        public
        onlyOwner
    {
        _mint(_to, _amount);
    }

    /**
     * @dev Burn tokens.
     * @param _from The account to burn from.
     * @param _amount The amount to burn.
     */
    function burn(
        address _from,
        uint256 _amount
    )
        public
        onlyOwner
    {
        _burn(_from, _amount);
    }
}
