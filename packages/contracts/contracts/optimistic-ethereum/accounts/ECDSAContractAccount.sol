pragma solidity ^0.5.0;

/* Library Imports */
import { ECDSAUtils } from "../utils/libraries/ECDSAUtils.sol";
import { ExecutionManagerWrapper } from "../utils/libraries/ExecutionManagerWrapper.sol";
import { RLPReader } from "../utils/libraries/RLPReader.sol";

/* Contract Imports */
import { ExecutionManager } from "../ovm/ExecutionManager.sol";

import { console } from "@nomiclabs/buidler/console.sol";

/**
 * @title ECDSAContractAccount
 * @dev NOTE: This contract must be made upgradeable!
 */
contract ECDSAContractAccount {
    /*
     * Data Structures
     */

    struct EOATransaction {
        address target;
        uint256 nonce;
        uint256 gasLimit;
        uint256 gasPrice;
        bytes data;
    }


    /*
     * Constructor
     */
    
    constructor()
        public
    {
        // TODO: Pay the Sequencer a fee in the ETH ERC-20 token.
    }


    /*
     * Public Functions
     */

    /**
     * Executes a signed transaction.
     * @param _transaction Signed EOA transaction.
     * @param _isEthSignedMessage Whether or not the user used the `Ethereum Signed Message` prefix.
     * @param _v Signature `v` parameter.
     * @param _r Signature `r` parameter.
     * @param _s Signature `s` parameter.
     * @return Result of executing the transaction.
     */
    function execute(
        bytes memory _transaction,
        bool _isEthSignedMessage,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        public
        returns (
            bytes memory _ret
        )
    {
        ExecutionManager executionManager = ExecutionManager(msg.sender);

        require(
            ECDSAUtils.recover(
                _transaction,
                _isEthSignedMessage,
                _v,
                _r,
                _s,
                ExecutionManagerWrapper.ovmCHAINID(address(executionManager))
            ) == ExecutionManagerWrapper.ovmADDRESS(address(executionManager)),
            "Provided signature is invalid."
        );

        EOATransaction memory decodedTx = _decodeTransaction(_transaction, _isEthSignedMessage);

        uint256 expectedNonce = executionManager.ovmGETNONCE() + 1;
        require(
            decodedTx.nonce == expectedNonce,
            "Nonce must match expected nonce."
        );
        executionManager.ovmSETNONCE(expectedNonce);

        if (decodedTx.target == address(0)) {
            _ret = abi.encode(
                ExecutionManagerWrapper.ovmCREATE(
                    address(executionManager),
                    decodedTx.data,
                    decodedTx.gasLimit
                )
            );
        } else {
            _ret = ExecutionManagerWrapper.ovmCALL(
                address(executionManager),
                decodedTx.target,
                decodedTx.data,
                decodedTx.gasLimit
            );
        }

        return _ret;
    }


    /*
     * Internal Functions
     */

    /**
     * Decodes an ABI encoded EOA transaction.
     * @param _transaction Encoded transaction.
     * @param _isEthSignedMessage Whether or not this was signed with the Ethereum message prefix.
     * @return Decoded transaction as a struct.
     */
    function _decodeTransaction(
        bytes memory _transaction,
        bool _isEthSignedMessage
    )
        internal
        pure
        returns (
            EOATransaction memory _decoded
        )
    {
        if (_isEthSignedMessage) {
            (
                uint256 nonce,
                uint256 gasLimit,
                uint256 gasPrice,
                address target,
                bytes memory data
            ) = abi.decode(_transaction, (uint256, uint256, uint256, address, bytes));

            return EOATransaction({
                target: target,
                nonce: nonce,
                gasLimit: gasLimit,
                gasPrice: gasPrice,
                data: data
            });
        } else {
            RLPReader.RLPItem[] memory decoded = RLPReader.toList(RLPReader.toRlpItem(_transaction));

            return EOATransaction({
                target: RLPReader.toAddress(decoded[3]),
                nonce: RLPReader.toUint(decoded[0]),
                gasLimit: RLPReader.toUint(decoded[2]),
                gasPrice: RLPReader.toUint(decoded[1]),
                data: RLPReader.toBytes(decoded[5])
            });
        }
    }
}
