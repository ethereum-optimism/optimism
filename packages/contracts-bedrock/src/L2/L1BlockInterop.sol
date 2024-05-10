// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L1Block, ConfigType } from "src/L2/L1Block.sol";

/// @notice Thrown when a non-depositor account attempts to set L1 block values.
error NotDepositor();

/// @notice Thrown when dependencySetSize does not match the length of the dependency set.
error DependencySetSizeMismatch();

/// @notice Error when a chain ID is not in the interop dependency set.
error NotDependency();

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1BlockInterop
/// @notice Interop extenstions of L1Block.
contract L1BlockInterop is L1Block {
    /// @notice Event emitted when a new dependency is added to the interop dependency set.
    event DependencyAdded(uint256 indexed chainId);

    /// @notice Event emitted when a dependency is removed from the interop dependency set.
    event DependencyRemoved(uint256 indexed chainId);

    /// @notice The chain IDs of the interop dependency set.
    uint256[] public dependencySet;

    /// @custom:semver 1.5.0+interop
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+interop");
    }

    /// @notice Returns true if a chain ID is in the interop dependency set and false otherwise.
    ///         Every chain ID is in the interop dependency set of itself.
    /// @param _chainId The chain ID to check.
    /// @return True if the chain ID to check is in the interop dependency set. False otherwise.
    function isInDependencySet(uint256 _chainId) public view returns (bool) {
        // Every chain ID is in the interop dependency set of itself.
        if (_chainId == block.chainid) {
            return true;
        }

        uint256 length = dependencySet.length;
        for (uint256 i = 0; i < length;) {
            if (dependencySet[i] == _chainId) {
                return true;
            }
            unchecked {
                i++;
            }
        }

        return false;
    }

    /// @notice Returns the size of the interop dependency set.
    /// @return The size of the interop dependency set.
    function dependencySetSize() external view returns (uint8) {
        return uint8(dependencySet.length);
    }

    /// @notice Internal function to set configuration options for the L2 system.
    /// @param _type  The type of configuration to set.
    /// @param _value The encoded value with which to set the configuration.
    function _setConfig(ConfigType _type, bytes calldata _value) internal override {
        super._setConfig(_type, _value);

        // For ADD_DEPENDENCY config type
        if (_type == ConfigType.ADD_DEPENDENCY) {
            uint256 chainId = abi.decode(_value, (uint256));

            dependencySet.push(chainId);

            emit DependencyAdded(chainId);
            return;
        }

        // For REMOVE_DEPENDENCY config type
        if (_type == ConfigType.REMOVE_DEPENDENCY) {
            uint256 chainId = abi.decode(_value, (uint256));

            uint256 length = dependencySet.length;
            for (uint256 i = 0; i < length;) {
                if (dependencySet[i] == chainId) {
                    dependencySet[i] = dependencySet[length - 1];
                    dependencySet.pop();

                    emit DependencyRemoved(chainId);
                    return;
                }
                unchecked {
                    i++;
                }
            }

            revert NotDependency();
        }
    }
}
