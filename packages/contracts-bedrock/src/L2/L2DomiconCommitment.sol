// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { DomiconCommitment } from "src/universal/DomiconCommitment.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DomiconNode } from "src/universal/DomiconNode.sol";

/// @custom:proxied
/// @title L1StandardBridge
/// @notice The L1StandardBridge is responsible for transfering ETH and ERC20 tokens between L1 and
///         L2. In the case that an ERC20 token is native to L1, it will be escrowed within this
///         contract. If the ERC20 token is native to L2, it will be burnt. Before Bedrock, ETH was
///         stored within this contract. After Bedrock, ETH is instead stored inside the
///         OptimismPortal contract.
///         NOTE: this contract is not intended to support all variations of ERC20 tokens. Examples
///         of some token types that may not be properly supported by this contract include, but are
///         not limited to: tokens with transfer fees, rebasing tokens, and tokens with blocklists.
contract L2DomiconCommitment is DomiconCommitment, ISemver {

    /// @notice Semantic version.
    /// @custom:semver 1.4.1
    string public constant version = "1.4.1";

    /// @notice Constructs the L1DomiconCommitment contract.
    constructor(DomiconCommitment _otherCommitment) DomiconCommitment(_otherCommitment) {
        initialize();
    }

    /// @notice Initializer
    function initialize() public reinitializer(Constants.INITIALIZER) {
        __DomiconCommitment_init({ _messenger: CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER),_domiconNode : DomiconNode(Predeploys.L2_DOMICON_NODE) });
    }
}
