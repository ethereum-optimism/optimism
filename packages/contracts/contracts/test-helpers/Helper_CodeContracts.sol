pragma solidity >=0.7.0;
pragma experimental ABIEncoderV2;

import {console} from "@nomiclabs/buidler/console.sol";

interface Helper_CodeContractDataTypes {
    struct CALLResponse {
        bool success;
        bytes data;
    }
}


contract Helper_CodeContractForCalls is Helper_CodeContractDataTypes {
    bytes constant sampleCREATEData = abi.encodeWithSignature("ovmCREATE(bytes)", hex"");
    bytes constant sampleCREATE2Data = abi.encodeWithSignature("ovmCREATE2(bytes,bytes32)", hex"", 0x4242424242424242424242424242424242424242424242424242424242424242);

    function runSteps(
        bytes[] memory callsToEM,
        bool _shouldRevert,
        address _createEMResponsesStorer
    ) public returns(CALLResponse[] memory) {
        console.log("in runSteps()");
        uint numSteps = callsToEM.length;
        CALLResponse[] memory EMResponses = new CALLResponse[](numSteps);
        for (uint i = 0; i < numSteps; i++) {
            bytes memory dataToSend = callsToEM[i];
            console.log("calling EM with data:");
            console.logBytes(dataToSend);
            (bool success, bytes memory responseData) = address(msg.sender).call(dataToSend);
            console.log("step to EM had result:");
            console.logBool(success);
            EMResponses[i].success = success;
            if (_isOVMCreateCall(dataToSend)) {
                console.log("step to EM returned data:");
                console.logBytes(responseData);
                EMResponses[i].data = abi.encode(
                    responseData,
                    _getStoredEMREsponsesInCreate(_createEMResponsesStorer)
                );
                console.log("since this is create step, stored concatenation:");
                console.logBytes(EMResponses[i].data);
            } else {
                console.log("step to EM returned data:");
                console.logBytes(responseData);
                EMResponses[i].data = responseData;
            }
        }
        return EMResponses;
    }

    function _getStoredEMREsponsesInCreate(address _createEMResponsesStorer) internal returns(bytes memory) {
        (bool success, bytes memory data) = _createEMResponsesStorer.call(abi.encodeWithSignature("getLastResponses()"));
        return data;
    }

    function _isOVMCreateCall(bytes memory _calldata) public returns(bool) {
        return (
            _doMethodIdsMatch(_calldata, sampleCREATEData) || _doMethodIdsMatch(_calldata, sampleCREATE2Data)
        );
    }

    function _doMethodIdsMatch(bytes memory _calldata1, bytes memory _calldata2) internal returns(bool) {
        return (
            _calldata1[0] == _calldata2[0] &&
            _calldata1[1] == _calldata2[1] &&
            _calldata1[2] == _calldata2[2] &&
            _calldata1[3] == _calldata2[3]            
        );
    }
}

contract Helper_CodeContractForCreates is Helper_CodeContractForCalls {
    constructor(
        bytes[] memory callsToEM,
        bool _shouldRevert,
        bytes memory _codeToDeploy,
        address _createEMResponsesStorer
    ) {  
        console.log("In CREATE helper (deployment)");
        CALLResponse[] memory responses = runSteps(callsToEM, _shouldRevert, _createEMResponsesStorer);
        Helper_CreateEMResponsesStorer(_createEMResponsesStorer).store(responses);
        uint lengthToDeploy = _codeToDeploy.length;
        // todo  revert if _shouldrevert
        assembly {
            return(add(_codeToDeploy, 0x20), lengthToDeploy)
        }
    }
}

contract Helper_CreateEMResponsesStorer is Helper_CodeContractDataTypes {
    CALLResponse[] responses;

    function store(
        CALLResponse[] memory _responses
    ) public {
        console.log("create storer helper is storing responses...");
        for (uint i = 0; i < _responses.length; i++) {
            responses.push();
            responses[i] = _responses[i];
        }
        console.log("helper successfully stored this many responses:");
        console.logUint(responses.length);
    }

    function getLastResponses() public returns(CALLResponse[] memory) {
        console.log("helper is retreiving last stored responses.  It has this many responses stored:");
        console.logUint(responses.length);

        CALLResponse[] memory toReturn = responses;
        delete responses;
        return toReturn;
    }
}

contract Helper_CodeContractForReverts {
    function doRevert(
        bytes memory _revertdata
    ) public {
        uint revertLength = _revertdata.length;
        assembly {
            revert(add(_revertdata, 0x20), revertLength)
        }
    }
}

// note: behavior of this contract covers all 3 cases of EVM message exceptions:
// - out of gas
// - INVALID opcode
// - invalid JUMPDEST
contract Helper_CodeContractForInvalid {
    function doInvalid() public {
        assembly {
            invalid()
        }
    }
}

// note: behavior of this contract covers all 3 cases of EVM message exceptions:
// - out of gas
// - INVALID opcode
// - invalid JUMPDEST
contract Helper_CodeContractForInvalidInCreation {
    constructor() public {
        assembly {
            invalid()
        }
    }
}
