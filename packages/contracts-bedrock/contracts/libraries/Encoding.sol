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
     * @notice RLP encodes the L2 transaction that would be generated when a given deposit is sent
     *         to the L2 system. Useful for searching for a deposit in the L2 system.
     *
     * @param _from        Address of the sender of the deposit.
     * @param _to          Address of the receiver of the deposit.
     * @param _value       ETH value to send to the receiver.
     * @param _mint        ETH value to mint on L2.
     * @param _gasLimit    Gas limit to use for the transaction.
     * @param _isCreation  Whether or not the transaction is a contract creation.
     * @param _data        Data to send with the transaction.
     * @param _l1BlockHash Hash of the L1 block where the deposit was included.
     * @param _logIndex    Index of the deposit event in the L1 block.
     *
     * @return RLP encoded L2 deposit transaction.
     */
    function encodeDepositTransaction(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data,
        bytes32 _l1BlockHash,
        uint256 _logIndex
    ) internal pure returns (bytes memory) {
        bytes32 source = Hashing.hashDepositSource(_l1BlockHash, _logIndex);
        bytes[] memory raw = new bytes[](7);
        raw[0] = RLPWriter.writeBytes(abi.encodePacked(source));
        raw[1] = RLPWriter.writeAddress(_from);
        raw[2] = _isCreation ? RLPWriter.writeBytes("") : RLPWriter.writeAddress(_to);
        raw[3] = RLPWriter.writeUint(_mint);
        raw[4] = RLPWriter.writeUint(_value);
        raw[5] = RLPWriter.writeUint(uint256(_gasLimit));
        raw[6] = RLPWriter.writeBytes(_data);
        return abi.encodePacked(uint8(0x7e), uint8(0x0), RLPWriter.writeList(raw));
    }

    /**
     * @notice Encodes the cross domain message based on the version that is encoded into the
     *         message nonce.
     *
     * @param _nonce    Message nonce with version encoded into the first two bytes.
     * @param _sender   Address of the sender of the message.
     * @param _target   Address of the target of the message.
     * @param _value    ETH value to send to the target.
     * @param _gasLimit Gas limit to use for the message.
     * @param _data     Data to send with the message.
     *
     * @return Encoded cross domain message.
     */
    function encodeCrossDomainMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes memory) {
        (, uint16 version) = decodeVersionedNonce(_nonce);
        if (version == 0) {
            return encodeCrossDomainMessageV0(_target, _sender, _data, _nonce);
        } else if (version == 1) {
            return encodeCrossDomainMessageV1(_nonce, _sender, _target, _value, _gasLimit, _data);
        } else {
            revert("Encoding: unknown cross domain message version");
        }
    }

    /**
     * @notice Encodes a cross domain message based on the V0 (legacy) encoding.
     *
     * @param _target Address of the target of the message.
     * @param _sender Address of the sender of the message.
     * @param _data   Data to send with the message.
     * @param _nonce  Message nonce.
     *
     * @return Encoded cross domain message.
     */
    function encodeCrossDomainMessageV0(
        address _target,
        address _sender,
        bytes memory _data,
        uint256 _nonce
    ) internal pure returns (bytes memory) {
        return
            abi.encodeWithSignature(
                "relayMessage(address,address,bytes,uint256)",
                _target,
                _sender,
                _data,
                _nonce
            );
    }

    /**
     * @notice Encodes a cross domain message based on the V1 (current) encoding.
     *
     * @param _nonce    Message nonce.
     * @param _sender   Address of the sender of the message.
     * @param _target   Address of the target of the message.
     * @param _value    ETH value to send to the target.
     * @param _gasLimit Gas limit to use for the message.
     * @param _data     Data to send with the message.
     *
     * @return Encoded cross domain message.
     */
    function encodeCrossDomainMessageV1(
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
     * @notice Adds a version number into the first two bytes of a message nonce.
     *
     * @param _nonce   Message nonce to encode into.
     * @param _version Version number to encode into the message nonce.
     *
     * @return Message nonce with version encoded into the first two bytes.
     */
    function encodeVersionedNonce(uint240 _nonce, uint16 _version) internal pure returns (uint256) {
        uint256 nonce;
        assembly {
            nonce := or(shl(240, _version), _nonce)
        }
        return nonce;
    }

    /**
     * @notice Pulls the version out of a version-encoded nonce.
     *
     * @param _nonce Message nonce with version encoded into the first two bytes.
     *
     * @return Nonce without encoded version.
     * @return Version of the message.
     */
    function decodeVersionedNonce(uint256 _nonce) internal pure returns (uint240, uint16) {
        uint240 nonce;
        uint16 version;
        assembly {
            nonce := and(_nonce, 0x0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff)
            version := shr(240, _nonce)
        }
        return (nonce, version);
    }
}
