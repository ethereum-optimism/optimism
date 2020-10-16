// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

import { Helper_SimpleProxy } from "./Helper_SimpleProxy.sol";

contract Helper_PrecompileCaller is Helper_SimpleProxy {
    function callPrecompile(
        address _precompile,
        bytes memory _data
    )
        public
    {
        if (msg.sender == owner) {
            makeExternalCall(_precompile, _data);
        } else {
            makeExternalCall(target, msg.data);
        }
    }

    function getL1MessageSender(
        address _precompile,
        bytes memory _data
    )
        public
        returns (
            address
        )
    {
        callPrecompile(_precompile, _data);
    }
}
