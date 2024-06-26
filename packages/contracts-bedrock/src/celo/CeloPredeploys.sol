// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";

/// @title CeloPredeploys
/// @notice Contains constant addresses for protocol contracts that are pre-deployed to the L2 system.
library CeloPredeploys {
    address internal constant CELO_REGISTRY = 0x000000000000000000000000000000000000ce10;
    address internal constant GOLD_TOKEN = 0x471EcE3750Da237f93B8E339c536989b8978a438;
    address internal constant FEE_HANDLER = 0xcD437749E43A154C07F3553504c68fBfD56B8778;
    address internal constant FEE_CURRENCY_WHITELIST = 0xBB024E9cdCB2f9E34d893630D19611B8A5381b3c;
    address internal constant MENTO_FEE_HANDLER_SELLER = 0x4eFa274B7e33476C961065000D58ee09F7921A74;
    address internal constant UNISWAP_FEE_HANDLER_SELLER = 0xD3aeE28548Dbb65DF03981f0dC0713BfCBd10a97;
    address internal constant SORTED_ORACLES = 0xefB84935239dAcdecF7c5bA76d8dE40b077B7b33;
    address internal constant ADDRESS_SORTED_LINKED_LIST_WITH_MEDIAN = 0xED477A99035d0c1e11369F1D7A4e587893cc002B;
    address internal constant FEE_CURRENCY = 0x4200000000000000000000000000000000001022;
    address internal constant BRIDGED_ETH = 0x4200000000000000000000000000000000001023;
    address internal constant FEE_CURRENCY_DIRECTORY = 0x71FFbD48E34bdD5a87c3c683E866dc63b8B2a685;
    address internal constant cUSD = 0x765DE816845861e75A25fCA122bb6898B8B1282a;

    /// @notice Returns the name of the predeploy at the given address.
    function getName(address _addr) internal pure returns (string memory out_) {
        // require(isPredeployNamespace(_addr), "Predeploys: address must be a predeploy");

        if (_addr == CELO_REGISTRY) return "CeloRegistry";
        if (_addr == GOLD_TOKEN) return "GoldToken";
        if (_addr == FEE_HANDLER) return "FeeHandler";
        if (_addr == FEE_CURRENCY_WHITELIST) return "FeeCurrencyWhitelist";
        if (_addr == MENTO_FEE_HANDLER_SELLER) return "MentoFeeHandlerSeller";
        if (_addr == UNISWAP_FEE_HANDLER_SELLER) return "UniswapFeeHandlerSeller";
        if (_addr == SORTED_ORACLES) return "SortedOracles";
        if (_addr == ADDRESS_SORTED_LINKED_LIST_WITH_MEDIAN) return "AddressSortedLinkedListWithMedian";
        if (_addr == FEE_CURRENCY) return "FeeCurrency";
        if (_addr == BRIDGED_ETH) return "BridgedEth";
        if (_addr == FEE_CURRENCY_DIRECTORY) return "FeeCurrencyDirectory";
        if (_addr == cUSD) return "cUSD";

        revert("Predeploys: unnamed predeploy");
    }
}
