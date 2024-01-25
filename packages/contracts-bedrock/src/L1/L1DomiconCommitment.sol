// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DomiconCommitment } from "src/universal/DomiconCommitment.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { DomiconNode } from "src/universal/DomiconNode.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Hashing } from "src/libraries/Hashing.sol";

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
contract L1DomiconCommitment is DomiconCommitment, ISemver {
    using SafeERC20 for IERC20;

    /// @notice Semantic version.
    /// @custom:semver 1.4.1
    string public constant version = "1.4.1";

    /// @notice Constructs the L1StandardBridge contract.
    constructor() DomiconCommitment(DomiconCommitment(payable(Predeploys.L2_DOMICON_COMMITMENT))) {
        initialize({ _messenger: CrossDomainMessenger(address(0)),_node: DomiconNode(address(0)) });
    }

    /// @notice Initializer
    function initialize(CrossDomainMessenger _messenger,DomiconNode _node) public reinitializer(Constants.INITIALIZER) {
        __DomiconCommitment_init({ _messenger: _messenger,_domiconNode:_node });
    }

    function SubmitCommitment(uint64 _index,uint64 _length,uint64 _price,address _user,bytes calldata _sign,bytes calldata _commitment) external onlyEOA onlyBroadcastNode {
        require(checkSign(_user,_price,_index,_length,_sign,_commitment),"L1DomiconCommitment:invalid Signature");
//        require(indices[_user]==_index,"L1DomiconCommitment:index Error");

        IERC20(DOM).safeTransferFrom(_user, address(this), 200);

        submits[_user][_index]=_commitment;
        indices[_user]++;
        emit SendDACommitment(_index,_length,_price,msg.sender,_user,_sign,_commitment);

        _initSubmitCommitment(RECEIVE_DEFAULT_GAS_LIMIT,_index,_length,_price,msg.sender,_user,_sign,_commitment);
    }

    function checkSign(address _user,uint64 _price,uint64 _index,uint64 _length,bytes calldata _sign,bytes calldata _commitment) internal view returns (bool){
        bytes32 hash = Hashing.getDataHash(_user,msg.sender,_price,_index,_length,_commitment);
        return Hashing.verifySignature(hash,_sign,_user);
    }

    function getGas(uint256 length) internal pure returns(uint256){
        return length;
    }
}
