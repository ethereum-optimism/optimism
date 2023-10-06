// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Constants } from "src/libraries/Constants.sol";

/// @notice Initializable is a contract that facilitates calling a function wrapped
///         with the initializer() by the ERC-1967 admin. In practice, the admin
///         always calls the initializer() function even in Open Zeppelin's implementation
///         where it is permissionless to call and can only be called once. By allowing
///         the admin to call it, the security model is the same because the admin can
///         already call initialize to set state after changing the implementation.
abstract contract Initializable {
    /// @dev Emitted when the contract is called by an account that is not the owner.
    event Unauthorized();

    /// @dev A modifier that wraps a function which can only be called by
    ///      the owner of the proxy.
    modifier initializer() {
        address owner;
        assembly {
            owner := sload(constants.PROXY_OWNER_ADDRESS)
        }
        if (msg.sender != owner) revert Unauthorized();
        _;
    }
}
