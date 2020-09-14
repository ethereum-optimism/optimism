pragma solidity >=0.7.0;
pragma experimental ABIEncoderV2;

// note: this pattern breaks if an action leads to reversion, since the result will not be stored.
// Thus, only the final action in deployedActions can be an ovmREVERT, and the result must be checked via the callee.
contract MockOvmCodeContract {
    
    bytes[] public EMReturnValuesInConstructor;
    bytes[][] public EMReturnValuesInDeployed;
    
    bytes[][] public callDatasForEMInDeployed;
    bytes[] public returnDataForCodeContractInDeployed;

    uint public callIndex = 0;
    
    bytes public constrRet0;
    bytes public constrRet1;
    bytes public constCall0;
    
    constructor(bytes[] memory _callsToEMInConstructor, bytes[][] memory _calldatasToEMInDeployed, bytes[] memory _returnDataForCodeContractInDeployed) {
        require(_calldatasToEMInDeployed.length == _returnDataForCodeContractInDeployed.length, "Invalid behavior requested for mock code contract: mismatch between number of calldata batches and returndata for post-deployment behavior.");
        
        callDatasForEMInDeployed = _calldatasToEMInDeployed;
        returnDataForCodeContractInDeployed = _returnDataForCodeContractInDeployed;
        
        bytes[] memory callsToDoNow = _callsToEMInConstructor;
        bytes[] memory returnVals = doEMCalls(callsToDoNow);
        
        constCall0 = callsToDoNow[0];
        
        constrRet0 = returnVals[0];
        constrRet1 = returnVals[1];
        
        for (uint i = 0; i < returnVals.length; i++) {
            EMReturnValuesInConstructor.push(returnVals[i]);
        }    
    }

    fallback() external {
        bytes[] memory calldatas = callDatasForEMInDeployed[callIndex];
        
        bytes[] memory returndatas = doEMCalls(calldatas);
        
        EMReturnValuesInDeployed.push();
        for (uint i = 0; i < returndatas.length; i++) {
            EMReturnValuesInDeployed[callIndex].push(returndatas[i]);
        }
        
        bytes memory dataToReturn = returnDataForCodeContractInDeployed[callIndex];
        callIndex++;
        uint returnLength = dataToReturn.length;
        assembly {
            return(add(dataToReturn, 0x20), returnLength)
        }
    }
    
    function doEMCalls(bytes[] memory _calldatas) internal returns(bytes[] memory) {
        bytes[] memory calldatas = _calldatas;
        bytes[] memory results = new bytes[](calldatas.length);
        for (uint i = 0; i < calldatas.length; i++) {
            bytes memory data = calldatas[i];
            bytes memory result = callExecutionManager(data);
            results[i] = result;
        }
        return results;
    }

    function callExecutionManager (bytes memory _data) internal returns (bytes memory actionResult) {
        uint dataLength = _data.length;
        uint returnedLength;
        assembly {
            function isContextCREATE() -> isCREATE {
                isCREATE := iszero(extcodesize(address()))
            }
            // Note: this function is the only way that the opcodes REVERT, CALLER, EXTCODESIZE, ADDRESS can appear in a code contract which passes SafetyChecker.isBytecodeSafe().
            // The static analysis enforces that the EXACT functionality below is implemented by comparing to a reference bytestring.
            function doSafeExecutionManagerCall(argOff, argLen, retOffset, retLen) {
                let success := call(gas(), caller(), 0, argOff, argLen, retOffset, retLen)
                if iszero(success) {
                    mstore(0, 0x2a2a7adb00000000000000000000000000000000000000000000000000000000) // ovmREVERT(bytes) methodId
                    returndatacopy(4, 0, 32)
                    let secondsuccess := call(gas(), caller(), 0, 0, 36, 0, 0)
                    if iszero(secondsuccess) {
                        returndatacopy(0, 0, 32)
                        revert(0, 32)
                    }
                    // ovmREVERT will only succeed if we are in a CREATE context, in which case we abort by deploying a single byte contract.
                    mstore(0,0)
                    return(0, 1)
                }
            }
            doSafeExecutionManagerCall(add(_data, 0x20), dataLength, 0, 0)
            returnedLength := returndatasize()
        }
        
        bytes memory returned = new bytes(returnedLength);
        assembly {
            returndatacopy(add(returned, 0x20), 0, returndatasize())
        }
        
        return returned;
    }
}