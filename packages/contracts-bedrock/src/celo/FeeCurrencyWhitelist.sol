// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";

import "./interfaces/IFeeCurrencyWhitelist.sol";

import "./common/Initializable.sol";

import "./common/interfaces/ICeloVersionedContract.sol";

/**
 * @title Holds a whitelist of the ERC20+ tokens that can be used to pay for gas
 * Not including the native Celo token
 */
contract FeeCurrencyWhitelist is IFeeCurrencyWhitelist, Ownable, Initializable, ICeloVersionedContract {
    // Array of all the tokens enabled
    address[] public whitelist;

    event FeeCurrencyWhitelisted(address token);

    event FeeCurrencyWhitelistRemoved(address token);

    /**
     * @notice Sets initialized == true on implementation contracts
     * @param test Set to true to skip implementation initialization
     */
    constructor(bool test) Initializable(test) { }

    /**
     * @notice Used in place of the constructor to allow the contract to be upgradable via proxy.
     */
    function initialize() external initializer {
        _transferOwnership(msg.sender);
    }

    /**
     * @notice Returns the storage, major, minor, and patch version of the contract.
     * @return Storage version of the contract.
     * @return Major version of the contract.
     * @return Minor version of the contract.
     * @return Patch version of the contract.
     */
    function getVersionNumber() external pure returns (uint256, uint256, uint256, uint256) {
        return (1, 1, 1, 0);
    }

    /**
     * @notice Removes a Mento token as enabled fee token. Tokens added with addToken should be
     * removed with this function.
     * @param tokenAddress The address of the token to remove.
     * @param index The index of the token in the whitelist array.
     */
    function removeToken(address tokenAddress, uint256 index) public onlyOwner {
        require(whitelist[index] == tokenAddress, "Index does not match");
        uint256 length = whitelist.length;
        whitelist[index] = whitelist[length - 1];
        whitelist.pop();
        emit FeeCurrencyWhitelistRemoved(tokenAddress);
    }

    /**
     * @dev Add a token to the whitelist
     * @param tokenAddress The address of the token to add.
     */
    function addToken(address tokenAddress) external onlyOwner {
        whitelist.push(tokenAddress);
        emit FeeCurrencyWhitelisted(tokenAddress);
    }

    /**
     * @return a list of all tokens enabled as gas fee currency.
     */
    function getWhitelist() external view returns (address[] memory) {
        return whitelist;
    }
}
