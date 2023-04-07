// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import { IInitializable } from "src/interfaces/IInitializable.sol";

/// @title Initializable
/// @author clabby <https://github.com/clabby>
/// @notice Enables a contract to have an initializer function that may only be ran once.
abstract contract Initializable is IInitializable {
    /// @notice Flag that designates whether or not the contract has been initialized.
    bool public initialized;

    /// @notice Emitted upon initialization of the contract.
    event Initialized();

    /// @notice Thrown when the contract has already been initialized.
    error AlreadyInitialized();

    /// @notice Only allows this contract to be initialized once.
    modifier initializer() {
        assembly {
            if sload(initialized.slot) {
                // Store error selector for "AlreadyInitialized()" in memory
                mstore(0x00, 0x0dc149f0)
                revert(0x1c, 0x04)
            }
        }
        _;
        // If the contract was initialized, flag it as such and emit the
        // `Initialized()` event.
        assembly {
            sstore(initialized.slot, 0x01)
        }
        emit Initialized();
    }
}
