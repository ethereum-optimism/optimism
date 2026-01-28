// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title MockAddressManager
/// @dev Mock implementation of an AddressManager.
contract MockAddressManager {
    /// @notice Mapping of names to addresses.
    mapping(string => address) public addresses;

    /// @notice Sets the address for a given name.
    function setAddress(string memory _name, address _addr) external {
        addresses[_name] = _addr;
    }

    /// @notice Returns the address for a given name.
    function getAddress(string memory _name) external view returns (address) {
        return addresses[_name];
    }
}
