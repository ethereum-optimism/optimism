pragma solidity ^0.5.0;

/* Testing Imports */
import { console } from "@nomiclabs/buidler/console.sol";

contract SimpleStorage {
    mapping(bytes32 => bytes32) public builtInStorage;

    function setStorage(bytes32 key, bytes32 value) public {
        bytes memory EMcalldata = abi.encodeWithSelector(bytes4(keccak256(bytes("ovmSSTORE()"))), key, value);

        // // #if FLAG_IS_DEBUG
        // console.log("Generated the following calldata for the EM (ovmSSTORE op):");
        // console.logBytes(EMcalldata);
        // // #endif

        (bool success,) = msg.sender.call(EMcalldata);

        // // #if FLAG_IS_DEBUG
        // console.log("call to ovmSSTORE was:");
        // console.log(success);
        // // #endif
    }

    function getStorage(bytes32 key) public returns (bytes32) {
        bytes memory EMcalldata = abi.encodeWithSelector(bytes4(keccak256(bytes("ovmSLOAD()"))), key);

        // // #if FLAG_IS_DEBUG
        // console.log("Generated the following calldata for the EM (ovmSLOAD op):");
        // console.logBytes(EMcalldata);
        // // #endif

        (bool success, bytes memory response) = msg.sender.call(EMcalldata);

        // // #if FLAG_IS_DEBUG
        // console.log("Got the following response from the EM (ovmSLOAD op):");
        // console.log(success);
        // console.logBytes(response);
        // console.log("which converts to the following bytes32:");
        // console.logBytes32(bytesToBytes32(response));
        // // #endif

        return bytesToBytes32(response);
    }

    function setSequentialSlots(uint startKey, bytes32 value) public {
        for (uint i = 0; i < 20; i++) {
            setStorage(bytes32(startKey + i), value);
        }
    }

    function setSameSlotRepeated(bytes32 key, bytes32 value) public {
        for (uint i = 0; i < 20; i++) {
            setStorage(key, value);
        }
    }

    function getStorages(bytes32 key) public {
        for (uint i = 0; i < 20; i++) {
            getStorage(key);
        }
    }

    function bytesToBytes32(bytes memory source) private pure returns (bytes32 result) {
        if (source.length == 0) {
            return 0x0;
        }
        assembly {
            result := mload(add(source, 32))
        }
    }
}
