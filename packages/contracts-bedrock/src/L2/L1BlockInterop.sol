// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L1Block } from "src/L2/L1Block.sol";
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";
import { GasPayingToken } from "src/libraries/GasPayingToken.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";
import "src/libraries/L1BlockErrors.sol";

/// @notice Enum representing different types of configurations that can be set on L1BlockInterop.
/// @custom:value SET_GAS_PAYING_TOKEN  Represents the config type for setting the gas paying token.
/// @custom:value ADD_DEPENDENCY        Represents the config type for adding a chain to the interop dependency set.
/// @custom:value REMOVE_DEPENDENCY     Represents the config type for removing a chain from the interop dependency set.
enum ConfigType {
    SET_GAS_PAYING_TOKEN,
    ADD_DEPENDENCY,
    REMOVE_DEPENDENCY
}

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1BlockInterop
/// @notice Interop extenstions of L1Block.
contract L1BlockInterop is L1Block {
    using EnumerableSet for EnumerableSet.UintSet;

    /// @notice Event emitted when a new dependency is added to the interop dependency set.
    event DependencyAdded(uint256 indexed chainId);

    /// @notice Event emitted when a dependency is removed from the interop dependency set.
    event DependencyRemoved(uint256 indexed chainId);

    /// @notice The interop dependency set, containing the chain IDs in it.
    EnumerableSet.UintSet dependencySet;

    /// @custom:semver +interop
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+interop");
    }

    /// @notice Returns true if a chain ID is in the interop dependency set and false otherwise.
    ///         The chain's chain ID is always considered to be in the dependency set.
    /// @param _chainId The chain ID to check.
    /// @return True if the chain ID to check is in the interop dependency set. False otherwise.
    function isInDependencySet(uint256 _chainId) public view returns (bool) {
        return _chainId == block.chainid || dependencySet.contains(_chainId);
    }

    /// @notice Returns the size of the interop dependency set.
    /// @return The size of the interop dependency set.
    function dependencySetSize() external view returns (uint8) {
        return uint8(dependencySet.length());
    }

    /// @notice Sets static configuration options for the L2 system. Can only be called by the special
    ///         depositor account.
    /// @param _type  The type of configuration to set.
    /// @param _value The encoded value with which to set the configuration.
    function setConfig(ConfigType _type, bytes calldata _value) external {
        if (msg.sender != DEPOSITOR_ACCOUNT()) revert NotDepositor();

        if (_type == ConfigType.SET_GAS_PAYING_TOKEN) {
            _setGasPayingToken(_value);
        } else if (_type == ConfigType.ADD_DEPENDENCY) {
            _addDependency(_value);
        } else if (_type == ConfigType.REMOVE_DEPENDENCY) {
            _removeDependency(_value);
        }
    }

    /// @notice Internal method to set the gas paying token.
    /// @param _value The encoded value with which to set the gas paying token.
    function _setGasPayingToken(bytes calldata _value) internal {
        (address token, uint8 decimals, bytes32 name, bytes32 symbol) = StaticConfig.decodeSetGasPayingToken(_value);

        GasPayingToken.set({ _token: token, _decimals: decimals, _name: name, _symbol: symbol });

        emit GasPayingTokenSet({ token: token, decimals: decimals, name: name, symbol: symbol });
    }

    /// @notice Internal method to add a dependency to the interop dependency set.
    /// @param _value The encoded value with which to add the dependency.
    function _addDependency(bytes calldata _value) internal {
        uint256 chainId = StaticConfig.decodeAddDependency(_value);

        if (dependencySet.length() == type(uint8).max) revert DependencySetSizeTooLarge();

        if (chainId == block.chainid || !dependencySet.add(chainId)) revert AlreadyDependency();

        emit DependencyAdded(chainId);
    }

    /// @notice Internal method to remove a dependency from the interop dependency set.
    /// @param _value The encoded value with which to remove the dependency.
    function _removeDependency(bytes calldata _value) internal {
        uint256 chainId = StaticConfig.decodeRemoveDependency(_value);

        if (chainId == block.chainid) revert CantRemovedDependency();

        if (!dependencySet.remove(chainId)) revert NotDependency();

        emit DependencyRemoved(chainId);
    }
}
