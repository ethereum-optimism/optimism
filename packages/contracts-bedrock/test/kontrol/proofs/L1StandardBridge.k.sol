// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import { DeploymentSummary } from "./utils/DeploymentSummary.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import { Types } from "src/libraries/Types.sol";
import { IL1StandardBridge as L1StandardBridge } from "src/L1/interfaces/IL1StandardBridge.sol";
import { ISuperchainConfig as SuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";
import { ICrossDomainMessenger as CrossDomainMessenger } from "src/universal/interfaces/ICrossDomainMessenger.sol";

contract L1StandardBridgeKontrol is DeploymentSummary, KontrolUtils {
    L1StandardBridge l1standardBridge;
    SuperchainConfig superchainConfig;

    function setUpInlined() public {
        l1standardBridge = L1StandardBridge(payable(l1StandardBridgeProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
    }

    function prove_finalizeBridgeERC20_paused(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        public
    {
        setUpInlined();

        // Pause Standard Bridge
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        vm.mockCall(
            address(l1standardBridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(l1standardBridge.otherBridge()))
        );

        vm.prank(address(l1standardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1standardBridge.finalizeBridgeERC20(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }

    function prove_finalizeBridgeETH_paused(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        public
    {
        setUpInlined();

        // Pause Standard Bridge
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        vm.mockCall(
            address(l1standardBridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(l1standardBridge.otherBridge()))
        );

        vm.prank(address(l1standardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1standardBridge.finalizeBridgeETH(_from, _to, _amount, _extraData);
    }
}
