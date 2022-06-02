//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Lib_CrossDomainUtils } from "./Lib_CrossDomainUtils.sol";
import { Lib_RLPWriter } from "./rlp/Lib_RLPWriter.sol";

/**
 * @title CrossDomainHashing
 * This library is responsible for holding cross domain utility
 * functions.
 * TODO(tynes): merge with Lib_CrossDomainUtils
 * TODO(tynes): fill out more devdocs
 */
library CrossDomainHashing {
    /**
     * @notice Compute the L2 transaction hash given
     * data about an L1 deposit transaction. This is useful for
     * environments that do not have access to arbitrary
     * RLP encoding functionality but have access to the
     * standard web3 API
     * TODO: rearrange args in a sane way
     * @param _l1BlockHash The L1 block hash corresponding to the block
     * the deposit was included in
     * @param _logIndex The log index of the event that the deposit was
     * created from. This can be found on the transaction receipt
     * @param _from The sender of the deposit
     * @param _to The L2 contract to be called by the deposit transaction
     * @param _isCreate Indicates if the deposit creates a contract
     * @param _mint The amount of ETH being minted by the transaction
     * @param _value The amount of ETH send in the L2 call
     * @param _gas The gas limit for the L2 call
     */
    function L2TransactionHash(
        bytes32 _l1BlockHash,
        uint256 _logIndex,
        address _from,
        address _to,
        bool _isCreate,
        uint256 _mint,
        uint256 _value,
        uint256 _gas,
        bytes memory _data
    ) internal pure returns (bytes32) {
        bytes memory raw = L2Transaction(
            _l1BlockHash,
            _logIndex,
            _from,
            _to,
            _isCreate,
            _mint,
            _value,
            _gas,
            _data
        );

        return keccak256(raw);
    }

    /**
     * @notice Compute the deposit transaction source hash.
     * This value ensures that the L2 transaction hash is unique
     * and deterministic based on L1 execution
     * @param l1BlockHash The L1 blockhash corresponding to the block including
     * the deposit
     * @param logIndex The index of the log that created the deposit transaction
     */
    function sourceHash(bytes32 l1BlockHash, uint256 logIndex) internal pure returns (bytes32) {
        bytes32 depositId = keccak256(abi.encode(l1BlockHash, logIndex));
        return keccak256(abi.encode(bytes32(0), depositId));
    }

    /**
     * @notice RLP encode a deposit transaction
     * This only works for user deposits, not system deposits
     * TODO: better name + rearrange the input param ordering?
     */
    function L2Transaction(
        bytes32 _l1BlockHash,
        uint256 _logIndex,
        address _from,
        address _to,
        bool _isCreate,
        uint256 _mint,
        uint256 _value,
        uint256 _gas,
        bytes memory _data
    ) internal pure returns (bytes memory) {
        bytes32 source = sourceHash(_l1BlockHash, _logIndex);

        bytes[] memory raw = new bytes[](7);

        raw[0] = Lib_RLPWriter.writeBytes(bytes32ToBytes(source));
        raw[1] = Lib_RLPWriter.writeAddress(_from);

        if (_isCreate == true) {
            require(_to == address(0));
            raw[2] = Lib_RLPWriter.writeBytes("");
        } else {
            raw[2] = Lib_RLPWriter.writeAddress(_to);
        }

        raw[3] = Lib_RLPWriter.writeUint(_mint);
        raw[4] = Lib_RLPWriter.writeUint(_value);
        raw[5] = Lib_RLPWriter.writeUint(_gas);
        raw[6] = Lib_RLPWriter.writeBytes(_data);

        bytes memory encoded = Lib_RLPWriter.writeList(raw);
        return abi.encodePacked(uint8(0x7e), encoded);
    }

    /**
     * @notice Helper function to turn bytes32 into bytes
     */
    function bytes32ToBytes(bytes32 input) internal pure returns (bytes memory) {
        bytes memory b = new bytes(32);
        assembly {
            mstore(add(b, 32), input) // set the bytes data
        }
        return b;
    }

    /**
     * @notice Adds the version to the nonce
     */
    function addVersionToNonce(uint256 _nonce, uint16 _version)
        internal
        pure
        returns (uint256 nonce)
    {
        assembly {
            nonce := or(shl(240, _version), _nonce)
        }
    }

    /**
     * @notice Gets the version out of the nonce
     */
    function getVersionFromNonce(uint256 _nonce) internal pure returns (uint16 version) {
        assembly {
            version := shr(240, _nonce)
        }
    }

    /**
     * @notice Encodes the cross domain message based on the version that
     * is encoded in the nonce
     */
    function getVersionedEncoding(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes memory) {
        uint16 version = getVersionFromNonce(_nonce);
        if (version == 0) {
            return getEncodingV0(_target, _sender, _data, _nonce);
        } else if (version == 1) {
            return getEncodingV1(_nonce, _sender, _target, _value, _gasLimit, _data);
        }

        revert("Unknown version.");
    }

    /**
     * @notice Compute the cross domain hash based on the versioned nonce
     */
    function getVersionedHash(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes32) {
        uint16 version = getVersionFromNonce(_nonce);
        if (version == 0) {
            return getHashV0(_target, _sender, _data, _nonce);
        } else if (version == 1) {
            return getHashV1(_nonce, _sender, _target, _value, _gasLimit, _data);
        }

        revert("Unknown version.");
    }

    /**
     * @notice Compute the legacy cross domain serialization
     */
    function getEncodingV0(
        address _target,
        address _sender,
        bytes memory _data,
        uint256 _nonce
    ) internal pure returns (bytes memory) {
        return Lib_CrossDomainUtils.encodeXDomainCalldata(_target, _sender, _data, _nonce);
    }

    /**
     * @notice Compute the V1 cross domain serialization
     */
    function getEncodingV1(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes memory) {
        return
            abi.encodeWithSignature(
                "relayMessage(uint256,address,address,uint256,uint256,bytes)",
                _nonce,
                _sender,
                _target,
                _value,
                _gasLimit,
                _data
            );
    }

    /**
     * @notice Compute the legacy hash of a cross domain message
     */
    function getHashV0(
        address _target,
        address _sender,
        bytes memory _data,
        uint256 _nonce
    ) internal pure returns (bytes32) {
        return keccak256(getEncodingV0(_target, _sender, _data, _nonce));
    }

    /**
     * @notice Compute the V1 hash of a cross domain message
     */
    function getHashV1(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes32) {
        return keccak256(getEncodingV1(_nonce, _sender, _target, _value, _gasLimit, _data));
    }
}
