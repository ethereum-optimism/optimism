// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { DomiconNode } from "src/universal/DomiconNode.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { Constants } from "src/libraries/Constants.sol";

contract L2DomiconNode is DomiconNode, ISemver {


    /// @notice Semantic version.
    /// @custom:semver 1.4.1
    string public constant version = "1.4.1";

    event BroadcastNode(address indexed add,string rpc,string name,uint256 stakedTokens);

    /// @notice Constructs the L1DomiconNode contract.
    constructor(DomiconNode _otherNode) DomiconNode(_otherNode) {
        initialize();
    }

    /// @notice Initializer
    function initialize() public reinitializer(Constants.INITIALIZER) {
        __DomiconNode_init({ _messenger: CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER) });
    }

    function finalizeBroadcastNode(
        address node,NodeInfo calldata nodeInfo
    )
    public
    payable
    override
    onlyOtherDomiconNode
    {
        emit FinalizeBroadcastNode(nodeInfo);
        super.finalizeBroadcastNode(node,nodeInfo);
    }
}
