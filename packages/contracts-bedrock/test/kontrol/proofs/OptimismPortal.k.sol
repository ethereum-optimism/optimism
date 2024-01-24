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

    /// TODO: Replace struct parameters and workarounds with the appropriate
    /// types once Kontrol supports symbolic `bytes` and `bytes[]`
    /// Tracking issue: https://github.com/runtimeverification/kontrol/issues/272
    function prove_proveWithdrawalTransaction_paused(
        // WithdrawalTransaction args
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        // bytes   memory _data,
        uint256 _l2OutputIndex,
        // OutputRootProof args
        bytes32 _outputRootProof0,
        bytes32 _outputRootProof1,
        bytes32 _outputRootProof2,
        bytes32 _outputRootProof3
    )
        external
    {
        setUpInlined();
        bytes memory _data = freshBigBytes(320);

        bytes[] memory _withdrawalProof = freshWithdrawalProof();

        Types.WithdrawalTransaction memory _tx =
            Types.WithdrawalTransaction(_nonce, _sender, _target, _value, _gasLimit, _data);
        Types.OutputRootProof memory _outputRootProof =
            Types.OutputRootProof(_outputRootProof0, _outputRootProof1, _outputRootProof2, _outputRootProof3);

        // Pause Optimism Portal
        vm.prank(optimismPortal.GUARDIAN());
        superchainConfig.pause("identifier");

        // No one can call proveWithdrawalTransaction
        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.proveWithdrawalTransaction(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// TODO: Replace struct parameters and workarounds with the appropriate
    /// types once Kontrol supports symbolic `bytes` and `bytes[]`
    /// Tracking issue: https://github.com/runtimeverification/kontrol/issues/272
    function prove_finalizeWithdrawalTransaction_paused(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit
    )
        external
    {
        setUpInlined();
        bytes memory _data = freshBigBytes(320);

        Types.WithdrawalTransaction memory _tx =
            Types.WithdrawalTransaction(_nonce, _sender, _target, _value, _gasLimit, _data);

        // Pause Optimism Portal
        vm.prank(optimismPortal.GUARDIAN());
        superchainConfig.pause("identifier");

        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.finalizeWithdrawalTransaction(_tx);
    }
}
