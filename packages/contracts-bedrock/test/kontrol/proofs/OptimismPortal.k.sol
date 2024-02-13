// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import { DeploymentSummary } from "./utils/DeploymentSummary.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import { Types } from "src/libraries/Types.sol";
import {
    IOptimismPortal as OptimismPortal,
    ISuperchainConfig as SuperchainConfig
} from "./interfaces/KontrolInterfaces.sol";

contract OptimismPortalKontrol is DeploymentSummary, KontrolUtils {
    OptimismPortal optimismPortal;
    SuperchainConfig superchainConfig;

    /// @dev Inlined setUp function for faster Kontrol performance
    ///      Tracking issue: https://github.com/runtimeverification/kontrol/issues/282
    function setUpInlined() public {
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
    }

    /// @custom:kontrol-length-equals _withdrawalProof: 10,
    /// @custom:kontrol-length-equals _withdrawalProof[]: 600,
    /// @custom:kontrol-length-equals data: 1000,
    function prove_proveWithdrawalTransaction_paused(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        setUpInlined();

        // Pause Optimism Portal
        vm.prank(optimismPortal.guardian());
        superchainConfig.pause("identifier");

        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.proveWithdrawalTransaction(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    function prove_finalizeWithdrawalTransaction_paused(Types.WithdrawalTransaction calldata _tx) external {
        setUpInlined();

        // Pause Optimism Portal
        vm.prank(optimismPortal.guardian());
        superchainConfig.pause("identifier");

        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.finalizeWithdrawalTransaction(_tx);
    }
}
