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

    /// @notice Reverts when the caller is not the SystemConfig contract.
    modifier onlySystemConfig() {
        if (msg.sender != address(systemConfig)) revert Unauthorized();
        _;
    }

    /// @notice Constructs the OptimismPortal contract.
    constructor() OptimismPortal() { }

    /// @custom:semver 2.8.0+interop
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+interop");
    }

    /// @notice Sets the gas paying token for the L2 system. This token is used as the
    ///         L2 native asset. Only the SystemConfig contract can call this function.
    /// @param _token    Address of the gas paying token.
    /// @param _decimals Decimals of the gas paying token.
    /// @param _name     Name of the gas paying token.
    /// @param _symbol   Symbol of the gas paying token.
    function setGasPayingToken(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        external
        override
        onlySystemConfig
    {
        _setConfig(ConfigType.GAS_PAYING_TOKEN, abi.encode(_token, _decimals, _name, _symbol));
    }

    /// @notice Adds a chain to the interop dependency set.
    ///         Only the SystemConfig contract can call this function.
    /// @param _chainId Chain ID to add to the dependency set.
    function addDependency(uint256 _chainId) external onlySystemConfig {
        _setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @notice Removes a chain from the interop dependency set.
    ///         Only the SystemConfig contract can call this function.
    /// @param _chainId Chain ID to remove from the dependency set.
    function removeDependency(uint256 _chainId) external onlySystemConfig {
        _setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @notice Sets static configuration options for the L2 system.
    /// @param _type  Type of configuration to set.
    /// @param _value Encoded value of the configuration.
    function _setConfig(ConfigType _type, bytes memory _value) internal {
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
