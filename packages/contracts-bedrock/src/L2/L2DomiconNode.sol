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

    /// @notice Constructs the L1StandardBridge contract.
    constructor() DomiconNode(DomiconNode(payable(Predeploys.L2_DOMICON_COMMITMENT))) {
        initialize({ _messenger: CrossDomainMessenger(address(0)) });
    }

    /// @notice Initializer
    function initialize(CrossDomainMessenger _messenger) public reinitializer(Constants.INITIALIZER) {
        __DomiconNode_init({ _messenger: _messenger });
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
