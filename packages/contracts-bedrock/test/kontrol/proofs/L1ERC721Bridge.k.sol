// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import { DeploymentSummary } from "./utils/DeploymentSummary.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import { Types } from "src/libraries/Types.sol";
import {
    IL1ERC721Bridge as L1ERC721Bridge,
    ISuperchainConfig as SuperchainConfig
} from "./interfaces/KontrolInterfaces.sol";

contract L1ERC721BridgeKontrol is DeploymentSummary, KontrolUtils {
    L1ERC721Bridge l1ERC721Bridge;
    SuperchainConfig superchainConfig;

    function setUpInlined() public {
        l1ERC721Bridge = L1ERC721Bridge(l1ERC721BridgeProxyAddress);
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
    }

    /// TODO: Replace symbolic workarounds with the appropriate
    /// types once Kontrol supports symbolic `bytes` and `bytes[]`
    /// Tracking issue: https://github.com/runtimeverification/kontrol/issues/272
    function prove_finalizeBridgeERC721_paused(
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
            bytes32(uint256(uint160(address(l1ERC721Bridge.otherBridge()))))
        );

        // ASSUME: Conservative upper bound on the `_extraData` length, since extra data is optional
        // for convenience of off-chain tooling. This assumption can be removed once Kontrol
        // supports symbolic `bytes`: https://github.com/runtimeverification/kontrol/issues/272
        bytes memory _extraData = freshBigBytes(64);

        // Pause Standard Bridge
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        // Pranking with `vm.prank` instead will result in failure from Kontrol
        // Tracking issue: https://github.com/runtimeverification/kontrol/issues/316
        vm.startPrank(address(l1ERC721Bridge.messenger()));
        vm.expectRevert("L1ERC721Bridge: paused");
        l1ERC721Bridge.finalizeBridgeERC721(_localToken, _remoteToken, _from, _to, _amount, _extraData);
        vm.stopPrank();
    }
}
