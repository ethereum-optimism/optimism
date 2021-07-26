// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_RLPReader } from "../rlp/Lib_RLPReader.sol";
import { Lib_RLPWriter } from "../rlp/Lib_RLPWriter.sol";

/**
 * @title Lib_EIP155Tx
 * @dev A simple library for dealing with the transaction type defined by EIP155:
 *      https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
 */
library Lib_EIP155Tx {

    /***********
     * Structs *
     ***********/

    // Struct representing an EIP155 transaction. See EIP link above for more information.
    struct EIP155Tx {
        // These fields correspond to the actual RLP-encoded fields specified by EIP155.
        uint256 nonce;
        uint256 gasPrice;
        uint256 gasLimit;
        address to;
        uint256 value;
        bytes data;
        uint8 v;
        bytes32 r;
        bytes32 s;

        // Chain ID to associate this transaction with. Used all over the place, seemed easier to
        // set this once when we create the transaction rather than providing it as an input to
        // each function. I don't see a strong need to have a transaction with a mutable chain ID.
        uint256 chainId;

        // The ECDSA "recovery parameter," should always be 0 or 1. EIP155 specifies that:
        // `v = {0,1} + CHAIN_ID * 2 + 35`
        // Where `{0,1}` is a stand in for our `recovery_parameter`. Now computing our formula for
        // the recovery parameter:
        // 1. `v = {0,1} + CHAIN_ID * 2 + 35`
        // 2. `v = recovery_parameter + CHAIN_ID * 2 + 35`
        // 3. `v - CHAIN_ID * 2 - 35 = recovery_parameter`
        // So we're left with the final formula:
        // `recovery_parameter = v - CHAIN_ID * 2 - 35`
        // NOTE: This variable is a uint8 because `v` is inherently limited to a uint8. If we
        // didn't use a uint8, then recovery_parameter would always be a negative number for chain
        // IDs greater than 110 (`255 - 110 * 2 - 35 = 0`). So we need to wrap around to support
        // anything larger.
        uint8 recoveryParam;

        // Whether or not the transaction is a creation. Necessary because we can't make an address
        // "nil". Using the zero address creates a potential conflict if the user did actually
        // intend to send a transaction to the zero address.
        bool isCreate;
    }

    // Lets us use nicer syntax.
    using Lib_EIP155Tx for EIP155Tx;


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Decodes an EIP155 transaction and attaches a given Chain ID.
     * Transaction *must* be RLP-encoded.
     * @param _encoded RLP-encoded EIP155 transaction.
     * @param _chainId Chain ID to assocaite with this transaction.
     * @return Parsed transaction.
     */
    function decode(
        bytes memory _encoded,
        uint256 _chainId
    )
        internal
        pure
        returns (
            EIP155Tx memory
        )
    {
        Lib_RLPReader.RLPItem[] memory decoded = Lib_RLPReader.readList(_encoded);

        // Note formula above about how recoveryParam is computed.
        uint8 v = uint8(Lib_RLPReader.readUint256(decoded[6]));
        uint8 recoveryParam = uint8(v - 2 * _chainId - 35);

        // Recovery param being anything other than 0 or 1 indicates that we have the wrong chain
        // ID.
        require(
            recoveryParam < 2,
            "Lib_EIP155Tx: Transaction signed with wrong chain ID"
        );

        // Creations can be detected by looking at the byte length here.
        bool isCreate = Lib_RLPReader.readBytes(decoded[3]).length == 0;

        return EIP155Tx({
            nonce: Lib_RLPReader.readUint256(decoded[0]),
            gasPrice: Lib_RLPReader.readUint256(decoded[1]),
            gasLimit: Lib_RLPReader.readUint256(decoded[2]),
            to: Lib_RLPReader.readAddress(decoded[3]),
            value: Lib_RLPReader.readUint256(decoded[4]),
            data: Lib_RLPReader.readBytes(decoded[5]),
            v: v,
            r: Lib_RLPReader.readBytes32(decoded[7]),
            s: Lib_RLPReader.readBytes32(decoded[8]),
            chainId: _chainId,
            recoveryParam: recoveryParam,
            isCreate: isCreate
        });
    }

    /**
     * Encodes an EIP155 transaction into RLP.
     * @param _transaction EIP155 transaction to encode.
     * @param _includeSignature Whether or not to encode the signature.
     * @return RLP-encoded transaction.
     */
    function encode(
        EIP155Tx memory _transaction,
        bool _includeSignature
    )
        internal
        pure
        returns (
            bytes memory
        )
    {
        bytes[] memory raw = new bytes[](9);

        raw[0] = Lib_RLPWriter.writeUint(_transaction.nonce);
        raw[1] = Lib_RLPWriter.writeUint(_transaction.gasPrice);
        raw[2] = Lib_RLPWriter.writeUint(_transaction.gasLimit);

        // We write the encoding of empty bytes when the transaction is a creation, *not* the zero
        // address as one might assume.
        if (_transaction.isCreate) {
            raw[3] = Lib_RLPWriter.writeBytes("");
        } else {
            raw[3] = Lib_RLPWriter.writeAddress(_transaction.to);
        }

        raw[4] = Lib_RLPWriter.writeUint(_transaction.value);
        raw[5] = Lib_RLPWriter.writeBytes(_transaction.data);

        if (_includeSignature) {
            raw[6] = Lib_RLPWriter.writeUint(_transaction.v);
            raw[7] = Lib_RLPWriter.writeBytes32(_transaction.r);
            raw[8] = Lib_RLPWriter.writeBytes32(_transaction.s);
        } else {
            // Chain ID *is* included in the unsigned transaction.
            raw[6] = Lib_RLPWriter.writeUint(_transaction.chainId);
            raw[7] = Lib_RLPWriter.writeBytes("");
            raw[8] = Lib_RLPWriter.writeBytes("");
        }

        return Lib_RLPWriter.writeList(raw);
    }

    /**
     * Computes the hash of an EIP155 transaction. Assumes that you don't want to include the
     * signature in this hash because that's a very uncommon usecase. If you really want to include
     * the signature, just encode with the signature and take the hash yourself.
     */
    function hash(
        EIP155Tx memory _transaction
    )
        internal
        pure
        returns (
            bytes32
        )
    {
        return keccak256(
            _transaction.encode(false)
        );
    }

    /**
     * Computes the sender of an EIP155 transaction.
     * @param _transaction EIP155 transaction to get a sender for.
     * @return Address corresponding to the private key that signed this transaction.
     */
    function sender(
        EIP155Tx memory _transaction
    )
        internal
        pure
        returns (
            address
        )
    {
        return ecrecover(
            _transaction.hash(),
            _transaction.recoveryParam + 27,
            _transaction.r,
            _transaction.s
        );
    }
}
