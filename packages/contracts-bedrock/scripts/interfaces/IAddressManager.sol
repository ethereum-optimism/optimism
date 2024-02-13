// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title IAddressManager
/// @notice Minimal interface of the Legacy AddressManager.
interface IAddressManager {
    /// @notice Emitted when an address is modified in the registry.
    /// @param name       String name being set in the registry.
    /// @param newAddress Address set for the given name.
    /// @param oldAddress Address that was previously set for the given name.
    event AddressSet(string indexed name, address newAddress, address oldAddress);

    /// @notice Changes the address associated with a particular name.
    /// @param _name    String name to associate an address with.
    /// @param _address Address to associate with the name.
    function setAddress(string memory _name, address _address) external;

    /// @notice Retrieves the address associated with a given name.
    /// @param _name Name to retrieve an address for.
    /// @return Address associated with the given name.
    function getAddress(string memory _name) external view returns (address);
}
