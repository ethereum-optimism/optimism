pragma solidity ^0.8.9;

import { CrossDomainHashing } from "@eth-optimism/contracts-bedrock/contracts/libraries/Lib_CrossDomainHashing.sol";
import { WithdrawalVerifier } from "@eth-optimism/contracts-bedrock/contracts/libraries/Lib_WithdrawalVerifier.sol";

contract MessageEncodingHelper {
    function getVersionedEncoding(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external pure returns (bytes memory) {
        return CrossDomainHashing.getVersionedEncoding(
            _nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
    }

    function getVersionedHash(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external pure returns (bytes32) {
        return CrossDomainHashing.getVersionedHash(
            _nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
    }

    function withdrawalHash(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external pure returns (bytes32) {
        return WithdrawalVerifier.withdrawalHash(
            _nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
    }
}
