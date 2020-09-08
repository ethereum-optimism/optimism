pragma solidity ^0.5.0;

/* Library Imports */
import { DataTypes } from "../utils/libraries/DataTypes.sol";
import { TransactionParser } from "../utils/libraries/TransactionParser.sol";
import { ECDSAUtils } from "../utils/libraries/ECDSAUtils.sol";
import { ExecutionManagerWrapper } from "../utils/libraries/ExecutionManagerWrapper.sol";
import { RLPReader } from "../utils/libraries/RLPReader.sol";

/* Contract Imports */
import { ExecutionManager } from "../ovm/ExecutionManager.sol";

/**
 * @title ECDSAContractAccount
 * @dev NOTE: This contract must be made upgradeable!
 */
contract ECDSAContractAccount {
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
        DataTypes.EOATransaction memory decodedTx = TransactionParser.decodeEOATransaction(
            _transaction
        );
        bytes memory encodedTx = TransactionParser.encodeEOATransaction(
            decodedTx,
            _isEthSignedMessage
        );

        ExecutionManager executionManager = ExecutionManager(msg.sender);

        require(
            ECDSAUtils.recover(
                encodedTx,
                _isEthSignedMessage,
                _v,
                _r,
                _s,
                ExecutionManagerWrapper.ovmCHAINID(address(executionManager))
            ) == ExecutionManagerWrapper.ovmADDRESS(address(executionManager)),
            "Provided signature is invalid."
        );

        uint256 expectedNonce = executionManager.ovmGETNONCE() + 1;
        require(
            decodedTx.nonce == expectedNonce,
            "Nonce must match expected nonce."
        );
        executionManager.ovmSETNONCE(expectedNonce);

        if (decodedTx.to == address(0)) {
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
                decodedTx.to,
                decodedTx.data,
                decodedTx.gasLimit
            );
        }

        return _ret;
    }
}
