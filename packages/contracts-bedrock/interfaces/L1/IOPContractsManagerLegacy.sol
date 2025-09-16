// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Claim } from "src/dispute/lib/Types.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

interface IOPContractsManagerLegacyStandardValidator {
    struct ValidationInput {
        IProxyAdmin proxyAdmin;
        ISystemConfig sysCfg;
        bytes32 absolutePrestate;
        uint256 l2ChainID;
    }

    function validate(ValidationInput memory _input, bool _allowFailure) external view returns (string memory);
}

interface IOPContractsManagerLegacyUpgrade {
    /// @notice The input required to identify a chain for upgrading to U16a.
    struct OpChainConfig {
        ISystemConfig systemConfigProxy;
        IProxyAdmin proxyAdmin;
        Claim absolutePrestate;
    }

    /// @notice U16a upgrade method
    /// @param _opChainConfigs The chains to upgrade
    function upgrade(OpChainConfig[] memory _opChainConfigs) external;
}
