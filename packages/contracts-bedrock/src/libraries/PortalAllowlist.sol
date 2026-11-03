// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title PortalAllowlist
/// @notice Selects the addresses allowed to deposit and to prove or finalize withdrawals through
///         the OptimismPortal by L1 chain ID. A chain without a list admits any address, so
///         devnets and test suites keep working.
library PortalAllowlist {
    /// @notice Returns the addresses allowed to use the portal on the current L1 chain.
    /// @return allowed_ The allowed addresses, empty when the chain is unrestricted.
    function allowedAddresses() internal view returns (address[] memory allowed_) {
        if (block.chainid == 11155111) {
            allowed_ = new address[](4);
            allowed_[0] = 0xE5327a59e33CAB54AcFFFBEba6bb167AaEDCD953;
            allowed_[1] = 0x2EA11DcDAEE45D82aAe2Ad6e7255f8d5bb70AbFF;
            allowed_[2] = 0x7d9F14FCd14fdf7A47BC2657949e087aA24f62bA;
            allowed_[3] = 0xcE38EA73Aac620E9aF3d45b53088B9E7294C5a2C;
        }
        // TODO: add the mainnet addresses. Until then mainnet admits any address.
    }

    /// @notice Checks if an address is allowed to use the portal.
    /// @param _address The address to check.
    /// @return bool True if the address is allowed.
    function isAllowed(address _address) internal view returns (bool) {
        address[] memory allowed = allowedAddresses();
        if (allowed.length == 0) return true;
        for (uint256 i = 0; i < allowed.length; i++) {
            if (allowed[i] == _address) return true;
        }
        return false;
    }
}
