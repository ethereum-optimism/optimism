// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Constants } from "src/libraries/Constants.sol";
import { OptimismPortalInterop as OptimismPortal } from "src/L1/OptimismPortalInterop.sol";
import { GasPayingToken } from "src/libraries/GasPayingToken.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ConfigType } from "src/L2/L1BlockInterop.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";

/// @title SystemConfigInterop
/// @notice The SystemConfig contract is used to manage configuration of an Optimism network.
///         All configuration is stored on L1 and picked up by L2 as part of the derviation of
///         the L2 chain.
contract SystemConfigInterop is SystemConfig {
    /// @custom:semver +interop
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+interop");
    }

    /// @notice Internal setter for the gas paying token address, includes validation.
    ///         The token must not already be set and must be non zero and not the ether address
    ///         to set the token address. This prevents the token address from being changed
    ///         and makes it explicitly opt-in to use custom gas token. Additionally,
    ///         OptimismPortal's address must be non zero, since otherwise the call to set the
    ///         config for the gas paying token to OptimismPortal will fail.
    /// @param _token Address of the gas paying token.
    function _setGasPayingToken(address _token) internal override {
        if (_token != address(0) && _token != Constants.ETHER && !isCustomGasToken()) {
            require(
                ERC20(_token).decimals() == GAS_PAYING_TOKEN_DECIMALS, "SystemConfig: bad decimals of gas paying token"
            );
            bytes32 name = GasPayingToken.sanitize(ERC20(_token).name());
            bytes32 symbol = GasPayingToken.sanitize(ERC20(_token).symbol());

            // Set the gas paying token in storage and in the OptimismPortal.
            GasPayingToken.set({ _token: _token, _decimals: GAS_PAYING_TOKEN_DECIMALS, _name: name, _symbol: symbol });
            OptimismPortal(payable(optimismPortal())).setConfig(
                ConfigType.SET_GAS_PAYING_TOKEN,
                StaticConfig.encodeSetGasPayingToken({
                    _token: _token,
                    _decimals: GAS_PAYING_TOKEN_DECIMALS,
                    _name: name,
                    _symbol: symbol
                })
            );
        }
    }

    /// @notice Adds a chain to the interop dependency set. Can only be called by the owner.
    /// @param _chainId Chain ID of chain to add.
    function addDependency(uint256 _chainId) external onlyOwner {
        OptimismPortal(payable(optimismPortal())).setConfig(
            ConfigType.ADD_DEPENDENCY, StaticConfig.encodeAddDependency(_chainId)
        );
    }

    /// @notice Removes a chain from the interop dependency set. Can only be called by the owner.
    /// @param _chainId Chain ID of the chain to remove.
    function removeDependency(uint256 _chainId) external onlyOwner {
        OptimismPortal(payable(optimismPortal())).setConfig(
            ConfigType.REMOVE_DEPENDENCY, StaticConfig.encodeRemoveDependency(_chainId)
        );
    }
}
