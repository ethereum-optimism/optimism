pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;


contract L2ToL1MessagePasser {
    event L2ToL1Message(
       address ovmSender,
       bytes callData
    );

    address executionManagerAddress;
    constructor(address _executionManagerAddress) public {
        executionManagerAddress = _executionManagerAddress;
    }

    function passMessageToL1(bytes memory messageData) public {
        // for now, to be trustfully relayed by sequencer to L1, so just emit an event for the sequencer to pick up.
        address ovmMsgSender = getCALLER();
        emit L2ToL1Message(
            ovmMsgSender,
            messageData
        );
    }

    function getCALLER() internal returns(address) {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmCALLER()") >> 224;
        address addr = executionManagerAddress;

        address theCaller;
        assembly {
            let callBytes := mload(0x40)
            calldatacopy(callBytes, 0, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)

            // overwrite call params
            let result := mload(0x40)
            let success := call(gas, addr, 0, callBytes, calldatasize, result, 500000)

            if eq(success, 0) {
                revert(0, 0)
            }

            theCaller := mload(result)
        }
        return theCaller;
    }
}