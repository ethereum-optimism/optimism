pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;


contract L2ToL1MessagePasser {
    event L2ToL1Message(
       uint _nonce,
       address _ovmSender,
       bytes _callData
    );

    uint nonce;
    address executionManagerAddress;
    constructor(address _executionManagerAddress) public {
        executionManagerAddress = _executionManagerAddress;
    }

    function passMessageToL1(bytes memory messageData) public {
        // for now, to be trustfully relayed by sequencer to L1, so just emit an event for the sequencer to pick up.
        address ovmMsgSender = getCALLER();
        emit L2ToL1Message(
            nonce++,
            ovmMsgSender,
            messageData
        );
    }

    function getCALLER() internal returns(address) {
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
