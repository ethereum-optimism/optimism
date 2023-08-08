// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "../libraries/Predeploys.sol";
import { OptimismPortal } from "./OptimismPortal.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { Semver } from "../universal/Semver.sol";

/// @custom:proxied
/// @title L1CrossDomainMessenger
/// @notice The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
///         for sending and receiving data on the L1 side. Users are encouraged to use this
///         interface instead of interacting with lower-level contracts directly.
contract L1CrossDomainMessenger is CrossDomainMessenger, Semver {
    /// @notice Address of the OptimismPortal. The public getter for this
    ///         is legacy and will be removed in the future. Use `portal()` instead.
    /// @custom:network-specific
    /// @custom:legacy
    OptimismPortal public PORTAL;

    /// @custom:semver 1.5.0
    /// @notice Constructs the L1CrossDomainMessenger contract.
    constructor() Semver(1, 5, 0) CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER) {
        initialize({ _portal: OptimismPortal(payable(0)) });
    }

    /// @notice Initializes the contract.
    /// @param _portal Address of the OptimismPortal contract on this network.
    function initialize(OptimismPortal _portal) public reinitializer(2) {
        PORTAL = _portal;
        __CrossDomainMessenger_init();
    }

    /// @notice Getter for the OptimismPortal address.
    function portal() external view returns (address) {
        return address(PORTAL);
    }

    /// @inheritdoc CrossDomainMessenger
    function _sendMessage(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        bytes memory _data
    ) internal override {
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
}
