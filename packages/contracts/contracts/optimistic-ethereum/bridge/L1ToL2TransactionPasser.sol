pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title L1ToL2TransactionPasser
 */
contract L1ToL2TransactionPasser {
    /*
     * Events
     */

    event L1ToL2Transaction(
        uint _nonce,
        address _sender,
        address _target,
        bytes _callData
    );


    /*
     * Contract Variables
     */

    uint private nonce;


    /*
     * Public Functions
     */

    /**
     * Pass an L1 transaction to the L2 rollup chain.
     * @param _ovmEntrypoint Target address for the transaction.
     * @param _ovmCalldata Calldata for the transaction.
     */
    function passTransactionToL2(
        address _ovmEntrypoint,
        bytes memory _ovmCalldata
    )
        public
    {
        // TODO: Actually create/enqueue a rollup block with this message.
        // We are simply mocking this functionality for now.

        emit L1ToL2Transaction(
            nonce++,
            msg.sender,
            _ovmEntrypoint,
            _ovmCalldata
        );
    }
}