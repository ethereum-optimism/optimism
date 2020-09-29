// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

contract Helper_SimpleProxy {
    address private target;

    constructor(
        address _target
    ) {
        target = _target;
    }

    fallback()
        external
    {
        (bool success, bytes memory returndata) = target.call(msg.data);

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
