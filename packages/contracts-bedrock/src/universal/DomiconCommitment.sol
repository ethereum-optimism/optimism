// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { IOptimismMintableERC20, ILegacyMintableERC20 } from "src/universal/IOptimismMintableERC20.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { DomiconNode } from "src/universal/DomiconNode.sol";
import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

/// @custom:upgradeable
/// @title StandardBridge
/// @notice StandardBridge is a base contract for the L1 and L2 standard ERC20 bridges. It handles
///         the core bridging logic, including escrowing tokens that are native to the local chain
///         and minting/burning tokens that are native to the remote chain.
abstract contract DomiconCommitment is Initializable{

    event SendDACommitment(uint256 index,uint256 length,uint256 price,address indexed broadcaster,address indexed user,bytes sign,bytes commitment);

    /// @notice The L2 gas limit set when eth is depoisited using the receive() function.
    uint32 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 200_000;

    /// @notice Corresponding bridge on the other domain. This public getter is deprecated
    ///         and will be removed in the future. Please use `otherBridge` instead.
    ///         This can safely be an immutable because for the L1StandardBridge, it will
    ///         be set to the L2StandardBridge address, which is the same for all OP Stack
    ///         chains. For the L2StandardBridge, there are not multiple proxies using the
    ///         same implementation.
    /// @custom:legacy
    /// @custom:network-specific
    DomiconCommitment public immutable OTHER_COMMITMENT;

    /// @custom:legacy
    /// @custom:spacer messenger
    /// @notice Spacer for backwards compatibility.
    address private spacer_0_2_20;

    /// @custom:legacy
    /// @custom:spacer l2TokenBridge
    /// @notice Spacer for backwards compatibility.
    address private spacer_1_0_20;

    address public DOM = 0x779877A7B0D9E8603169DdbD7836e478b4624789;

    mapping(address => mapping(uint256 => bytes)) public submits;
    mapping(address => uint256) public indices;

    /// @notice Messenger contract on this domain. This public getter is deprecated
    ///         and will be removed in the future. Please use `messenger` instead.
    /// @custom:network-specific
    CrossDomainMessenger public messenger;

    DomiconNode public domiconNode;

    /// @notice Reserve extra slots (to a total of 50) in the storage layout for future upgrades.
    ///         A gap size of 46 was chosen here, so that the first slot used in a child contract
    ///         would be a multiple of 50.
    uint256[46] private __gap;


    event FinalizeSubmitCommitment(
        uint256 index,uint256 length,uint256 price,address indexed broadcaster,address indexed user,bytes sign,bytes commitment
    );

    /// @notice Only allow EOAs to call the functions. Note that this is not safe against contracts
    ///         calling code within their constructors, but also doesn't really matter since we're
    ///         just trying to prevent users accidentally depositing with smart contract wallets.
    modifier onlyEOA() {
        require(!Address.isContract(msg.sender), "DomiconCommitment: function can only be called from an EOA");
        _;
    }

    /// @notice Ensures that the caller is a cross-chain message from the other commitment.
    modifier onlyOtherCommitment() {
        require(
            msg.sender == address(messenger) && messenger.xDomainMessageSender() == address(OTHER_COMMITMENT),
            "DomiconCommitment: function can only be called from the other commitment"
        );
        _;
    }

    modifier onlyBroadcastNode(){
        require(
            domiconNode.IsNodeBroadcast(msg.sender),"DomiconCommitment: broadcast node address error"
        );
        _;
    }

    /// @param _otherCommitment Address of the other DomiconCommitment contract.
    constructor(DomiconCommitment _otherCommitment) {
        OTHER_COMMITMENT = _otherCommitment;
    }

    /// @notice Initializer.
    /// @param _messenger   Address of CrossDomainMessenger on this network.
    // solhint-disable-next-line func-name-mixedcase
    function __DomiconCommitment_init(CrossDomainMessenger _messenger,DomiconNode _domiconNode) internal onlyInitializing {
        messenger = _messenger;
        domiconNode = _domiconNode;
    }

    /// @notice Getter for messenger contract.
    /// @custom:legacy
    /// @return Messenger contract on this domain.
    function MESSENGER() external view returns (CrossDomainMessenger) {
        return messenger;
    }

    /// @notice Getter for the remote domain Commitment contract.
    function otherCommitment() external view returns (DomiconCommitment) {
        return OTHER_COMMITMENT;
    }

    function DOMICON_NODE() external view returns(DomiconNode){
        return domiconNode;
    }

    function _initSubmitCommitment(
        uint32 _minGasLimit,
        uint256 _index,uint256 _length,uint256 _price,address _broadcaster,address _user,bytes calldata _sign,bytes calldata _commitment
    )
    internal
    {
        messenger.sendSubmitMessage(
            address(OTHER_COMMITMENT),
            abi.encodeWithSelector(this.finalizeSubmitCommitment.selector, _index,_length,_price,_broadcaster,_user,_sign,_commitment),
            _minGasLimit
        );
    }

    function finalizeSubmitCommitment(
        uint256 _index,uint256 _length,uint256 _price,address _broadcaster,address _user,bytes calldata _sign,bytes calldata _commitment
    )
    public
    payable
    onlyOtherCommitment
    {
        submits[_user][_index]=_commitment;
        emit FinalizeSubmitCommitment(_index,_length,_price,_broadcaster,_user,_sign,_commitment);
    }

    function SetDom(address addr)external {
        DOM=addr;
    }
}
