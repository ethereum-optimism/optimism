pragma solidity >=0.7.0;
pragma experimental ABIEncoderV2;

struct MessageSteps {
    bytes[] callsToEM;
    bool shouldRevert;
}

struct EMResponse {
    bool success;
    bytes data;
}

contract Helper_CodeContractForCalls {
    function runSteps(
        MessageSteps calldata _stepsToRun
    ) external returns(EMResponse[] memory) {
        uint numSteps = _stepsToRun.callsToEM.length;
        EMResponse[] memory EMResponses = new EMResponse[](numSteps);
        for (uint i = 0; i < numSteps; i++) {
            bytes memory dataToSend = _stepsToRun.callsToEM[i];
            (bool success, bytes memory responseData) = address(msg.sender).call(dataToSend);
            EMResponses[i].success = success;
            EMResponses[i].data = responseData;
        }
        return EMResponses; // TODO: revert with this data in case of !shouldRevert
    }
}
