// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Libraries
import { Storage } from "src/libraries/Storage.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";

/// @notice Base contract for ProxyAdmin-owned contracts. This contract is used to introspect
///         compatible Proxy contracts so that their ProxyAdmin and ProxyAdmin owner addresses can
///         be retrieved onchain. Existing Proxy contracts don't have these getters, so we need a
///         base contract instead.
/// @dev WARNING: This contract is ONLY designed to be used with either the Optimism Proxy
///      implementation or the Optimism ResolvedDelegateProxy implementation. It is not safe to use
///      this contract with any other proxy implementation.
///      WARNING: Multiple OP Stack chains may share the same ProxyAdmin owner address.
abstract contract ProxyAdminOwnedBase {
    /// @notice Thrown when the ProxyAdmin owner of the current contract is not the same as the
    ///         ProxyAdmin owner of the other Proxy address provided.
    error ProxyAdminOwnedBase_NotSharedProxyAdminOwner();

    /// @notice Thrown when the caller is not the ProxyAdmin owner.
    error ProxyAdminOwnedBase_NotProxyAdminOwner();

    /// @notice Thrown when the caller is not the ProxyAdmin.
    error ProxyAdminOwnedBase_NotProxyAdmin();

    /// @notice Thrown when the caller is not the ProxyAdmin owner or the ProxyAdmin.
    error ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner();

    /// @notice Thrown when the ProxyAdmin owner of the current contract is not found.
    error ProxyAdminOwnedBase_ProxyAdminNotFound();

    /// @notice Thrown when the current contract is not a ResolvedDelegateProxy.
    error ProxyAdminOwnedBase_NotResolvedDelegateProxy();

    /// @notice Getter for the owner of the ProxyAdmin.
    function proxyAdminOwner() public view returns (address) {
        return proxyAdmin().owner();
    }

    /// @notice Getter for the ProxyAdmin contract that owns this Proxy contract.
    function proxyAdmin() public view returns (IProxyAdmin) {
        // First check for a non-zero address in the reserved slot.
        address proxyAdminAddress = Storage.getAddress(Constants.PROXY_OWNER_ADDRESS);
        if (proxyAdminAddress != address(0)) {
            return IProxyAdmin(proxyAdminAddress);
        }

        // Otherwise, we'll try to read the AddressManager slot.
        // First we make sure this is almost certainly a ResolvedDelegateProxy. We only have a
        // single ResolvedDelegateProxy and it's for the L1CrossDomainMessenger, so we'll check
        // that the storage slot for the mapping at slot 0 returns the string
        // "OVM_L1CrossDomainMessenger". We need to use Solidity's rules for how strings are stored
        // in storage slots to do this.
        if (
            Storage.getBytes32(keccak256(abi.encode(address(this), uint256(0))))
                != bytes32(
                    uint256(bytes32("OVM_L1CrossDomainMessenger"))
                        | uint256(bytes("OVM_L1CrossDomainMessenger").length * 2)
                )
        ) {
            revert ProxyAdminOwnedBase_NotResolvedDelegateProxy();
        }

        // Ok, now we'll try to read the AddressManager slot.
        address addressManagerAddress = Storage.getAddress(keccak256(abi.encode(address(this), uint256(1))));
        if (addressManagerAddress != address(0)) {
            return IProxyAdmin(IAddressManager(addressManagerAddress).owner());
        }

        // We should revert here, we couldn't find a non-zero owner address.
        revert ProxyAdminOwnedBase_ProxyAdminNotFound();
    }

    /// @notice Reverts if the ProxyAdmin owner of the current contract is not the same as the
    ///         ProxyAdmin owner of the other Proxy address provided. Useful asserting that both
    ///         the current contract and the other Proxy share the same security model.+
    function _assertSharedProxyAdminOwner(address _proxy) internal view {
        if (proxyAdminOwner() != ProxyAdminOwnedBase(_proxy).proxyAdminOwner()) {
            revert ProxyAdminOwnedBase_NotSharedProxyAdminOwner();
        }
    }

    /// @notice Reverts if the caller is not the ProxyAdmin owner.
    function _assertOnlyProxyAdminOwner() internal view {
        if (proxyAdminOwner() != msg.sender) {
            revert ProxyAdminOwnedBase_NotProxyAdminOwner();
        }
    }

    /// @notice Reverts if the caller is not the ProxyAdmin.
    function _assertOnlyProxyAdmin() internal view {
        if (address(proxyAdmin()) != msg.sender) {
            revert ProxyAdminOwnedBase_NotProxyAdmin();
        }
    }

    function _assertOnlyProxyAdminOrProxyAdminOwner() internal view {
        if (address(proxyAdmin()) != msg.sender && proxyAdminOwner() != msg.sender) {
            revert ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner();
        }
    }
}
