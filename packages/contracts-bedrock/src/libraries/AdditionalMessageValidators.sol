// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Storage } from "src/libraries/Storage.sol";
import { Constants } from "src/libraries/Constants.sol";
import { LibString } from "@solady/utils/LibString.sol";

/// @title IAdditionalMessageValidators
/// @notice Implemented by contracts that are aware of custom message validation
///         rules added to L1/L2 briding contracts (if they exist).
interface IAdditionalMessageValidators {
    /// @notice Getter for the L1MessageValidator address called from the OptimismPortal.
    ///         The zero address represents no message validation and should not be called.
    function l1MessageValidator() external view returns (address);
    /// @notice Getter for the L2MessageValidator address called from the L2CrossDomainMessenger.
    ///         The zero address represents no custom message validation.
    function l2MessageValidator() external view returns (address);
    /// @notice Returns true if the network has additional message validation turned on.
    function isAdditionalMessageValidating() external view returns (bool);
}

/// @title AdditionalMessageValidators
/// @notice Handles reading and writing additional message validation settings to storage.
///         To be used in any place where message validation information is read or
///         written to state. If multiple contracts use this library, the
///         values in storage should be kept in sync between them.
library AdditionalMessageValidators {
    /// @notice The storage slot that contains the address of the L1MessageValidator
    bytes32 internal constant L1_MESSAGE_VALIDATOR_SLOT = bytes32(uint256(keccak256("opstack.l1messagevalidator")) - 1);

    /// @notice The storage slot that contains the address of the L2MessageValidator
    bytes32 internal constant L2_MESSAGE_VALIDATOR_SLOT = bytes32(uint256(keccak256("opstack.l2messagevalidator")) - 1);

    /// @notice Reads the L1_MESSAGE_VALIDATOR_SLOT from the magic storage slot.
    function getL1MessageValidator() internal view returns (address addr_) {
        addr_ = Storage.getAddress(L1_MESSAGE_VALIDATOR_SLOT);
    }

    /// @notice Reads the L2_MESSAGE_VALIDATOR_SLOT from the magic storage slot.
    function getL2MessageValidator() internal view returns (address addr_) {
        addr_ = Storage.getAddress(L2_MESSAGE_VALIDATOR_SLOT);
    }

    /// @notice Writes the gas paying token, its decimals, name and symbol to the magic storage slot.
    function set(address _l1MessageValidator, address _l2MessageValidator) internal {
        Storage.setAddress(L1_MESSAGE_VALIDATOR_SLOT, _l1MessageValidator);
        Storage.setAddress(L2_MESSAGE_VALIDATOR_SLOT, _l2MessageValidator);
    }
}
