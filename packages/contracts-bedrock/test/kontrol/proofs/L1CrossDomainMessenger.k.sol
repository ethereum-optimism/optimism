// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import { DeploymentSummary } from "./utils/DeploymentSummary.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import {
    IL1CrossDomainMessenger as L1CrossDomainMessenger,
    ISuperchainConfig as SuperchainConfig
} from "./interfaces/KontrolInterfaces.sol";

contract L1CrossDomainMessengerKontrol is DeploymentSummary, KontrolUtils {
    L1CrossDomainMessenger l1CrossDomainMessenger;
    SuperchainConfig superchainConfig;

    /// @dev Inlined setUp function for faster Kontrol performance
    ///      Tracking issue: https://github.com/runtimeverification/kontrol/issues/282
    function setUpInlined() public {
        l1CrossDomainMessenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxyAddress);
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
    }

    /// TODO: Replace struct parameters and workarounds with the appropriate
    /// types once Kontrol supports symbolic `bytes` and `bytes[]`
    /// Tracking issue: https://github.com/runtimeverification/kontrol/issues/272
    function prove_relayMessage_paused(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gas
    )
        external
    {
        setUpInlined();

        bytes memory _message = freshBigBytes(600);

        // Pause System
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        vm.expectRevert("CrossDomainMessenger: paused");
        l1CrossDomainMessenger.relayMessage(_nonce, _sender, _target, _value, _gas, _message);
    }
}
