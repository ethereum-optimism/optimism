// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { L1BlockInterop, ConfigType } from "src/L2/L1BlockInterop.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @custom:proxied
/// @title OptimismPortalInterop
/// @notice The OptimismPortal is a low-level contract responsible for passing messages between L1
///         and L2. Messages sent directly to the OptimismPortal have no form of replayability.
///         Users are encouraged to use the L1CrossDomainMessenger for a higher-level interface.
contract OptimismPortalInterop is OptimismPortal {
    /// @notice Thrown when a non-depositor account attempts update static configuration.
    error Unauthorized();

    /// @custom:semver +interop
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+interop");
    }

    /// @notice Sets static configuration options for the L2 system.
    /// @param _type  Type of configuration to set.
    /// @param _value Encoded value of the configuration.
    function setConfig(ConfigType _type, bytes memory _value) external {
        if (msg.sender != address(systemConfig)) revert Unauthorized();

        // Set L2 deposit gas as used without paying burning gas. Ensures that deposits cannot use too much L2 gas.
        // This value must be large enough to cover the cost of calling `L1Block.setConfig`.
        useGas(SYSTEM_DEPOSIT_GAS_LIMIT);

        // Emit the special deposit transaction directly that sets the config in the L1Block predeploy contract.
        emit TransactionDeposited(
            Constants.DEPOSITOR_ACCOUNT,
            Predeploys.L1_BLOCK_ATTRIBUTES,
            DEPOSIT_VERSION,
            abi.encodePacked(
                uint256(0), // mint
                uint256(0), // value
                uint64(SYSTEM_DEPOSIT_GAS_LIMIT), // gasLimit
                false, // isCreation,
                abi.encodeCall(L1BlockInterop.setConfig, (_type, _value))
            )
        );
    }
}
