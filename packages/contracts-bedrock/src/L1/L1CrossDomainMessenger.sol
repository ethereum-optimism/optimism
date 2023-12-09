// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

/// @custom:proxied
/// @title L1CrossDomainMessenger
/// @notice The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
///         for sending and receiving data on the L1 side. Users are encouraged to use this
///         interface instead of interacting with lower-level contracts directly.
contract L1CrossDomainMessenger is CrossDomainMessenger, ISemver {
    /// @notice Address of the OptimismPortal. This will be removed in the
    ///         future, use `portal` instead.
    /// @custom:legacy
    OptimismPortal public immutable PORTAL;

    /// @notice Address of the SuperchainConfig contract.
    SuperchainConfig public superchainConfig;

    /// @notice Semantic version.
    /// @custom:semver 2.1.1
    string public constant version = "2.1.1";

    /// @notice Constructs the L1CrossDomainMessenger contract.
    /// @param _portal Address of the OptimismPortal contract on this network.
    constructor(OptimismPortal _portal) CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER) {
        PORTAL = _portal;
        initialize({ _superchainConfig: SuperchainConfig(address(0)) });
    }

    /// @notice Initializes the contract.
    /// @param _superchainConfig Address of the SuperchainConfig contract on this network.
    function initialize(SuperchainConfig _superchainConfig) public initializer {
        superchainConfig = _superchainConfig;
        __CrossDomainMessenger_init();
    }

    /// @notice Getter for the OptimismPortal address.
    function portal() external view returns (address) {
        return address(PORTAL);
    }

    /// @inheritdoc CrossDomainMessenger
    function _sendMessage(address _to, uint64 _gasLimit, uint256 _value, bytes memory _data) internal override {
        PORTAL.depositTransaction{ value: _value }(_to, _value, _gasLimit, false, _data);
    }

    /// @inheritdoc CrossDomainMessenger
    function _isOtherMessenger() internal view override returns (bool) {
        return msg.sender == address(PORTAL) && PORTAL.l2Sender() == OTHER_MESSENGER;
    }

    /// @inheritdoc CrossDomainMessenger
    function _isUnsafeTarget(address _target) internal view override returns (bool) {
        return _target == address(this) || _target == address(PORTAL);
    }

    /// @inheritdoc CrossDomainMessenger
    function paused() public view override returns (bool) {
        return superchainConfig.paused();
    }
}
