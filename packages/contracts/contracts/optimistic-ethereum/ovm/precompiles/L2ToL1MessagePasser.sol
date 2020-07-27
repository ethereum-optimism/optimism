pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title L2ToL1MessagePasser
 */
contract L2ToL1MessagePasser {
    /*
     * Events
     */

    event L2ToL1Message(
       uint _nonce,
       address _ovmSender,
       bytes _callData
    );


    /*
     * Contract Variables
     */

    uint private nonce;
    address private executionManagerAddress;


    /*
     * Constructor
     */

    /**
     * @param _executionManagerAddress Address of the ExecutionManager contract.
     */
    constructor(
        address _executionManagerAddress
    )
        public
    {
        executionManagerAddress = _executionManagerAddress;
    }


    /*
     * Public Functions
     */

    /**
     * Passes a message to L1.
     * @param _messageData Message to pass to L1.
     */
    function passMessageToL1(
        bytes memory _messageData
    )
        public
    {
        // For now, to be trustfully relayed by sequencer to L1, so just emit
        // an event for the sequencer to pick up.

        emit L2ToL1Message(
            nonce++,
            getCALLER(),
            _messageData
        );
    }


    /*
     * Internal Functions
     */

    /**
     * Retrieves the OVM message caller.
     * @return Address of the message caller.
     */
    function getCALLER()
        internal
        returns (address)
    {
        bytes32 methodId = keccak256("ovmCALLER()");
        address addr = executionManagerAddress;

        address theCaller;
        assembly {
            // store methodId at free memory
            let callBytes := mload(0x40)
            mstore(callBytes, methodId)

            // we overwrite the call args here because why not!
            let result := callBytes
            let success := call(gas, addr, 0, callBytes, calldatasize, result, 500000)

            if eq(success, 0) {
                revert(result, returndatasize)
            }

            theCaller := mload(result)
        }

        return theCaller;
    }
}