// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

contract Helper_SimpleProxy {
    address private owner;
    address private target;

    constructor() {
        owner = msg.sender;
    }

    fallback()
        external
    {
        makeExternalCall(target, msg.data);
    }

    function setTarget(
        address _target
    )
        public
    {
        if (msg.sender == owner) {
            target = _target;
        } else {
            makeExternalCall(target, msg.data);
        }
    }

    function makeExternalCall(
        address _target,
        bytes memory _calldata
    )
        private
    {
        (bool success, bytes memory returndata) = _target.call(_calldata);

        if (success) {
            assembly {
                return(add(returndata, 0x20), mload(returndata))
            }
        } else {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }
}
