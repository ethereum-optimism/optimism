// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

contract Helper_ModifiableStorage {
    mapping (address => address) private target;

    constructor(
        address _target
    )
    {
        target[address(this)] = _target;
    }

    fallback()
        external
    {
        (bool success, bytes memory returndata) = target[address(this)].delegatecall(msg.data);

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

    function __setStorageSlot(
        bytes32 _key,
        bytes32 _value
    )
        public
    {
        assembly {
            sstore(_key, _value)
        }
    }

    function __getStorageSlot(
        bytes32 _key
    )
        public
        view
        returns (
            bytes32 _value
        )
    {
        bytes32 value;
        assembly {
            value := sload(_key)
        }
        return value;
    }
}
