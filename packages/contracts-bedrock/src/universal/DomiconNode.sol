// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { IOptimismMintableERC20, ILegacyMintableERC20 } from "src/universal/IOptimismMintableERC20.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

abstract contract DomiconNode is Initializable{

    event FinalizeBroadcastNode(NodeInfo nodeInfo);
    DomiconNode public immutable OTHER_DOMICON_NODE;

    /// @notice The L2 gas limit set when eth is depoisited using the receive() function.
    uint32 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 200_000;


    uint256 internal index =0;

    /// @custom:legacy
    /// @custom:spacer messenger
    /// @notice Spacer for backwards compatibility.
    address private spacer_0_2_20;

    /// @custom:legacy
    /// @custom:spacer l2TokenBridge
    /// @notice Spacer for backwards compatibility.
    address private spacer_1_0_20;

    struct NodeInfo{
        address add;
        string rpc;
        string name;
        uint256 stakedTokens;
        uint256 index;
    }

    mapping(address => NodeInfo) public broadcastingNodes;
    address[] public broadcastNodeList;
    mapping(address => NodeInfo)  public storageNodes;
    address[] public storageNodeList;


    /// @notice Messenger contract on this domain. This public getter is deprecated
    ///         and will be removed in the future. Please use `messenger` instead.
    /// @custom:network-specific
    CrossDomainMessenger public messenger;

    modifier onlyEOA() {
        require(!Address.isContract(msg.sender), "DomiconNode: function can only be called from an EOA");
        _;
    }

    modifier onlyOtherDomiconNode() {
        require(
            msg.sender == address(messenger) && messenger.xDomainMessageSender() == address(OTHER_DOMICON_NODE),
            "DomiconNode: function can only be called from the other domicon node"
        );
        _;
    }

    /// @param _otherDomiconNode Address of the other DomiconNode contract.
    constructor(DomiconNode _otherDomiconNode) {
        OTHER_DOMICON_NODE = _otherDomiconNode;
    }

    /// @notice Initializer.
    /// @param _messenger   Address of CrossDomainMessenger on this network.
    // solhint-disable-next-line func-name-mixedcase
    function __DomiconNode_init(CrossDomainMessenger _messenger) internal onlyInitializing {
        messenger = _messenger;
    }

    /// @notice Getter for messenger contract.
    /// @custom:legacy
    /// @return Messenger contract on this domain.
    function MESSENGER() external view returns (CrossDomainMessenger) {
        return messenger;
    }

    /// @notice Getter for the remote domain Commitment contract.
    function otherDomiconNode() external view returns (DomiconNode) {
        return OTHER_DOMICON_NODE;
    }

    function IsNodeBroadcast(address addr) external view returns (bool){
        if (broadcastingNodes[addr].stakedTokens!=0){
            return true;
        }
        return false;
    }

    function registerBroadcastNode(address node,NodeInfo memory nodeInfo) internal  {
        broadcastingNodes[node] = nodeInfo;

        messenger.sendMessage(
            address(OTHER_DOMICON_NODE),
            abi.encodeWithSelector(this.finalizeBroadcastNode.selector, node,nodeInfo),
            RECEIVE_DEFAULT_GAS_LIMIT
        );
    }

    function registerStorageNode(address node,NodeInfo memory nodeInfo) internal  {
        storageNodes[node] = nodeInfo;
    }

    function finalizeBroadcastNode(
        address node,NodeInfo calldata nodeInfo
    )
    public
    payable
    virtual
    onlyOtherDomiconNode
    {
        emit FinalizeBroadcastNode(nodeInfo);
        broadcastNodeList.push(node);
        broadcastingNodes[node] = nodeInfo;
    }

    function BROADCAST_NODES() external view returns(address[] memory){
        return broadcastNodeList;
    }



    function STORAGE_NODES() external view returns(address[] memory){
        return storageNodeList;
    }
}
