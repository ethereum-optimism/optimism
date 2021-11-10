// SPDX-License-Identifier: MIT
pragma solidity ^0.8.8;

/* Library Imports */
import { AddressAliasHelper } from "../../standards/AddressAliasHelper.sol";

/**
 * @title TestLib_AddressAliasHelper
 */
contract TestLib_AddressAliasHelper {
    function applyL1ToL2Alias(address _address) public pure returns (address) {
        return AddressAliasHelper.applyL1ToL2Alias(_address);
    }

    function undoL1ToL2Alias(address _address) public pure returns (address) {
        return AddressAliasHelper.undoL1ToL2Alias(_address);
    }
}
