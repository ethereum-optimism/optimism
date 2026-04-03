// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Features } from "src/libraries/Features.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L1Block } from "src/L2/L1Block.sol";

// Interfaces
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1BlockCGT
/// @notice The L1BlockCGT predeploy gives users access to information about the last known L1 block.
///         Values within this contract are updated once per epoch (every L1 block) and can only be
///         set by the "depositor" account, a special system address. Depositor account transactions
///         are created by the protocol whenever we move to a new epoch.
contract L1BlockCGT is L1Block {
    /// @custom:semver +custom-gas-token.1
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+custom-gas-token.1");
    }

    /// @notice Returns whether the gas paying token is custom.
    function isCustomGasToken() public view override returns (bool isCustom_) {
        isCustom_ = isFeatureEnabled[Features.CUSTOM_GAS_TOKEN];
    }

    /// @notice Returns the gas paying token, its decimals, name and symbol.
    function gasPayingToken() public pure override returns (address, uint8) {
        revert("L1BlockCGT: deprecated");
    }

    /// @notice Returns the gas paying token name.
    ///         If nothing is set in state, then it means ether is used.
    ///         This function cannot be removed because WETH depends on it.
    function gasPayingTokenName() public view override returns (string memory name_) {
        name_ =
            isCustomGasToken() ? ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).gasPayingTokenName() : "Ether";
    }

    /// @notice Returns the gas paying token symbol.
    ///         If nothing is set in state, then it means ether is used.
    ///         This function cannot be removed because WETH depends on it.
    function gasPayingTokenSymbol() public view override returns (string memory symbol_) {
        symbol_ =
            isCustomGasToken() ? ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).gasPayingTokenSymbol() : "ETH";
    }
}
