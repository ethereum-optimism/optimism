pragma solidity ^0.5.0;

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

    uint nonce;


    /*
     * Public Functions
     */

    function passTransactionToL2(
        address _ovmEntrypoint,
        bytes memory _ovmCalldata
    ) public {
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