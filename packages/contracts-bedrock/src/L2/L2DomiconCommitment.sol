// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { DomiconCommitment } from "src/universal/DomiconCommitment.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { Constants } from "src/libraries/Constants.sol";

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

    mapping(address => mapping(uint256 => DAInfo)) public submits;
    mapping(address => uint256) public indices;

    /// @notice Constructs the L1DomiconCommitment contract.
    constructor(DomiconCommitment _otherCommitment) DomiconCommitment(_otherCommitment) {
        initialize();
    }

    /// @notice Initializer
    function initialize() public reinitializer(Constants.INITIALIZER) {
        __DomiconCommitment_init({ _messenger: CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER) });
    }

    function SubmitCommitment(uint256 _index,uint256 _length,uint256 _price,address _user,bytes calldata _sign,bytes calldata _commitment) external onlyEOA {
        require(checkSign(_user,_sign),"L1DomiconCommitment:invalid Signature");
        require(indices[_user]==_index,"L1DomiconCommitment:index Error");
        submits[_user][_index]=DAInfo({index:_index,length:_length,price:_price,user:_user,broadcaster:msg.sender,sign:_sign,commitment:_commitment});
        indices[_user]++;
        emit SendDACommitment(_index,_length,_price,msg.sender,_user,_sign,_commitment);

        _initSubmitCommitment(RECEIVE_DEFAULT_GAS_LIMIT,_index,_length,_price,msg.sender,_user,_sign,_commitment);
    }

    function checkSign(address user,bytes calldata sign) internal returns (bool){
        return true;
    }

    function checkIndex(address user,uint256 index) internal returns (bool){
        return true;
    }

    function getGas(uint256 length) internal returns(uint256){
        return length;
    }
}
