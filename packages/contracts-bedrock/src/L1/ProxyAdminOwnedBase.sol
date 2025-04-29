// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Libraries
import { Storage } from "src/libraries/Storage.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @notice Base contract for ProxyAdmin-owned contracts. This contract is used to introspect
///         compatible Proxy contracts so that their ProxyAdmin and ProxyAdmin owner addresses can
///         be retrieved onchain. Existing Proxy contracts don't have these getters, so we need a
///         base contract instead.
abstract contract ProxyAdminOwnedBase {
    /// @notice Getter for the owner of the ProxyAdmin.
    function proxyAdminOwner() public view returns (address) {
        return proxyAdmin().owner();
    }

    /// @notice Getter for the ProxyAdmin contract that owns this Proxy contract.
    function proxyAdmin() public view returns (IProxyAdmin) {
        // Get the proxy admin address reading for the reserved slot it has on the Proxy contract.
        return IProxyAdmin(Storage.getAddress(Constants.PROXY_OWNER_ADDRESS));
    }

    /// @notice Checks if the ProxyAdmin owner of the current contract is the same as the
    ///         ProxyAdmin owner of the given proxy.
    /// @param _proxy The address of the proxy to check.
    function _sameProxyAdminOwner(address _proxy) internal view returns (bool) {
        return proxyAdminOwner() == ProxyAdminOwnedBase(_proxy).proxyAdminOwner();
    }

    /// @notice Modifier that reverts if the caller is not the ProxyAdmin owner.
    modifier onlyProxyAdminOwner() {
        if (proxyAdminOwner() != msg.sender) {
            revert("ProxyAdminOwnedBase: only the ProxyAdmin owner can call this function");
        }
        _;
    }

    /// @notice Modifier that reverts if the caller is not the ProxyAdmin.
    modifier onlyProxyAdmin() {
        if (address(proxyAdmin()) != msg.sender) {
            revert("ProxyAdminOwnedBase: only the ProxyAdmin can call this function");
        }
        _;
    }
}
