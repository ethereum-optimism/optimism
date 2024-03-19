// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import { DeploymentSummary } from "./utils/DeploymentSummary.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import { Types } from "src/libraries/Types.sol";
import {
    IL1StandardBridge as L1StandardBridge,
    ISuperchainConfig as SuperchainConfig
} from "./interfaces/KontrolInterfaces.sol";

contract L1StandardBridgeKontrol is DeploymentSummary, KontrolUtils {
    L1StandardBridge l1standardBridge;
    SuperchainConfig superchainConfig;

    function setUpInlined() public {
        l1standardBridge = L1StandardBridge(payable(l1StandardBridgeProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
    }

    /// TODO: Replace symbolic workarounds with the appropriate
    /// types once Kontrol supports symbolic `bytes` and `bytes[]`
    /// Tracking issue: https://github.com/runtimeverification/kontrol/issues/272
    function prove_finalizeBridgeERC20_paused(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount
    )
        public
    {
        setUpInlined();

        // Current workaround to be replaced with `vm.mockCall`, once the cheatcode is implemented in Kontrol
        // This overrides the storage slot read by `CrossDomainMessenger::xDomainMessageSender`
        // Tracking issue: https://github.com/runtimeverification/kontrol/issues/285
        vm.store(
            l1CrossDomainMessengerProxyAddress,
            hex"00000000000000000000000000000000000000000000000000000000000000cc",
            bytes32(uint256(uint160(address(l1standardBridge.otherBridge()))))
        );

        // ASSUME: Upper bound on the `_extraData` length, since extra data is optional for
        // for convenience of off-chain tooling, and should not affect execution  This assumption
        // can be removed once Kontrol supports symbolic `bytes`:
        // https://github.com/runtimeverification/kontrol/issues/272
        bytes memory _extraData = freshBigBytes(32);

        // Pause Standard Bridge
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        // Pranking with `vm.prank` instead will result in failure from Kontrol
        // Tracking issue: https://github.com/runtimeverification/kontrol/issues/316
        vm.startPrank(address(l1standardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1standardBridge.finalizeBridgeERC20(_localToken, _remoteToken, _from, _to, _amount, _extraData);
        vm.stopPrank();
    }

    /// TODO: Replace symbolic workarounds with the appropriate
    /// types once Kontrol supports symbolic `bytes` and `bytes[]`
    /// Tracking issue: https://github.com/runtimeverification/kontrol/issues/272
    function prove_finalizeBridgeETH_paused(address _from, address _to, uint256 _amount) public {
        setUpInlined();

        // Current workaround to be replaced with `vm.mockCall`, once the cheatcode is implemented in Kontrol
        // This overrides the storage slot read by `CrossDomainMessenger::xDomainMessageSender`
        // Tracking issue: https://github.com/runtimeverification/kontrol/issues/285
        vm.store(
            l1CrossDomainMessengerProxyAddress,
            hex"00000000000000000000000000000000000000000000000000000000000000cc",
            bytes32(uint256(uint160(address(l1standardBridge.otherBridge()))))
        );

        // ASSUME: Upper bound on the `_extraData` length, since extra data is optional for
        // for convenience of off-chain tooling, and should not affect execution  This assumption
        // can be removed once Kontrol supports symbolic `bytes`:
        // https://github.com/runtimeverification/kontrol/issues/272
        bytes memory _extraData = freshBigBytes(32);

        // Pause Standard Bridge
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        // Pranking with `vm.prank` instead will result in failure from Kontrol
        // Tracking issue: https://github.com/runtimeverification/kontrol/issues/316
        vm.startPrank(address(l1standardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1standardBridge.finalizeBridgeETH(_from, _to, _amount, _extraData);
        vm.stopPrank();
    }
}
