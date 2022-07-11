// SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Hashing } from "./Hashing.sol";
import { RLPWriter } from "./rlp/RLPWriter.sol";

/**
 * @title Encoding
 * @notice Encoding handles Optimism's various different encoding schemes.
 */
library Encoding {
    /**
     * Generates the correct cross domain calldata for a message.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return ABI encoded cross domain calldata.
     */
    function encodeXDomainCalldata(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) internal pure returns (bytes memory) {
        return
            abi.encodeWithSignature(
                "relayMessage(address,address,bytes,uint256)",
                _target,
                _sender,
                _message,
                _messageNonce
            );
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
        bytes32 source = Hashing.sourceHash(_l1BlockHash, _logIndex);

        bytes[] memory raw = new bytes[](7);

        raw[0] = RLPWriter.writeBytes(abi.encodePacked(source));
        raw[1] = RLPWriter.writeAddress(_from);

        if (_isCreate == true) {
            require(_to == address(0));
            raw[2] = RLPWriter.writeBytes("");
        } else {
            raw[2] = RLPWriter.writeAddress(_to);
        }

        raw[3] = RLPWriter.writeUint(_mint);
        raw[4] = RLPWriter.writeUint(_value);
        raw[5] = RLPWriter.writeUint(_gas);
        raw[6] = RLPWriter.writeBytes(_data);

        bytes memory encoded = RLPWriter.writeList(raw);
        return abi.encodePacked(uint8(0x7e), uint8(0x0), encoded);
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
     * @notice Compute the legacy cross domain serialization
     */
    function getEncodingV0(
        address _target,
        address _sender,
        bytes memory _data,
        uint256 _nonce
    ) internal pure returns (bytes memory) {
        return encodeXDomainCalldata(_target, _sender, _data, _nonce);
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
}
