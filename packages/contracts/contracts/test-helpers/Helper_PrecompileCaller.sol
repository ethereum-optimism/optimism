// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

import { Helper_SimpleProxy } from "./Helper_SimpleProxy.sol";

contract Helper_PredeployCaller is Helper_SimpleProxy {
    function callPredeploy(
        address _predeploy,
        bytes memory _data
    )
        public
    {
        if (msg.sender == owner) {
            makeExternalCall(_predeploy, _data);
        } else {
            makeExternalCall(target, msg.data);
        }
    }

    function callPredeployAbi(
        address _predeploy,
        bytes memory _data
    )
        public
        returns (
            bytes memory
        )
    {

        bool success;
        bytes memory returndata;
        if (msg.sender == owner) {
            (success, returndata) = _predeploy.call(_data);
        } else {
            (success, returndata) = target.call(msg.data);
        }
        require(success, "Predeploy call reverted");
        return returndata;
    }

    function getL1MessageSender(
        address _predeploy,
        bytes memory _data
    )
        public
        returns (
            address
        )
    {
        callPredeploy(_predeploy, _data);
        return address(0); // unused: silence compiler
    }
}
