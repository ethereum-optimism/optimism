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
    /// @notice Contract of the SuperchainConfig contract.
    SuperchainConfig public superchainConfig;

    /// @notice Contract of the OptimismPortal.
    /// @custom:network-specific
    OptimismPortal public portal;

    /// @notice Semantic version.
    /// @custom:semver 2.2.0
    string public constant version = "2.2.0";

    /// @notice Constructs the L1CrossDomainMessenger contract.
    constructor() CrossDomainMessenger() {
        initialize({
            _superchainConfig: SuperchainConfig(address(0)),
            _portal: OptimismPortal(payable(address(0))),
            _otherMessenger: Predeploys.L2_CROSS_DOMAIN_MESSENGER
        });
    }

    /// @notice Initializes the contract.
    /// @param _superchainConfig Contract of the SuperchainConfig contract on this network.
    /// @param _portal Contract of the OptimismPortal contract on this network.
    /// @param _otherMessenger Address of the L2CrossDomainMessenger contract on the other network.
    function initialize(
        SuperchainConfig _superchainConfig,
        OptimismPortal _portal,
        address _otherMessenger
    )
        public
        initializer
    {
        superchainConfig = _superchainConfig;
        portal = _portal;
        __CrossDomainMessenger_init({ _otherMessenger: _otherMessenger });
    }

    /// @notice Getter function for the address of the OptimismPortal on this chain.
    /// @return Address of the OptimismPortal on this chain.
    /// @custom:legacy
    function PORTAL() external view returns (address) {
        return address(portal);
    }

    /// @inheritdoc CrossDomainMessenger
    function _sendMessage(address _to, uint64 _gasLimit, uint256 _value, bytes memory _data) internal override {
        portal.depositTransaction{ value: _value }(_to, _value, _gasLimit, false, _data);
    }

    /// @inheritdoc CrossDomainMessenger
    function _isOtherMessenger() internal view override returns (bool) {
        return msg.sender == address(portal) && portal.l2Sender() == otherMessenger;
    }

    /// @inheritdoc CrossDomainMessenger
    function _isUnsafeTarget(address _target) internal view override returns (bool) {
        return _target == address(this) || _target == address(portal);
    }

    /// @inheritdoc CrossDomainMessenger
    function paused() public view override returns (bool) {
        return superchainConfig.paused();
    }
}
