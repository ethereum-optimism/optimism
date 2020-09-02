pragma solidity ^0.5.0;

import { console } from '@nomiclabs/buidler/console.sol';

/**
 * @title ExecutionManagerWrapper
 * @dev Wraps ExecutionManager calls to be ABI encoded.
 */
library ExecutionManagerWrapper {
    function ovmCALLER(
        address _executionManagerAddress
    )
        internal
        returns (address)
    {
        bytes memory callbytes = abi.encodePacked(
            bytes4(keccak256("ovmCALLER()"))
        );

        bytes memory returndata = _ovmcall(
            _executionManagerAddress,
            callbytes
        );

        address ret;
        assembly {
            ret := mload(add(returndata, 0x20))
        }

        return ret;
    }

    function ovmADDRESS(
        address _executionManagerAddress
    )
        internal
        returns (address)
    {
        bytes memory callbytes = abi.encodePacked(
            bytes4(keccak256("ovmADDRESS()"))
        );

        bytes memory returndata = _ovmcall(
            _executionManagerAddress,
            callbytes
        );

        address ret;
        assembly {
            ret := mload(add(returndata, 0x20))
        }

        return ret;
    }

    function ovmCALL(
        address _executionManagerAddress,
        address _target,
        bytes memory _calldata
    )
        internal
        returns (
            bytes memory
        )
    {
        bytes memory callbytes = abi.encodePacked(
            bytes4(keccak256("ovmCALL()")),
            abi.encode(_target),
            _calldata
        );

        return _ovmcall(
            _executionManagerAddress,
            callbytes
        );
    }

    function ovmCREATE(
        address _executionManagerAddress,
        bytes memory _bytecode
    )
        internal
        returns (
            address
        )
    {
        bytes memory callbytes = abi.encodePacked(
            bytes4(keccak256("ovmCREATE()")),
            _bytecode
        );

        bytes memory returndata = _ovmcall(
            _executionManagerAddress,
            callbytes
        );

        address ret;
        assembly {
            ret := mload(add(returndata, 0x20))
        }

        return ret;
    }

    function _ovmcall(
        address _executionManagerAddress,
        bytes memory _callbytes
    )
        private
        returns (
            bytes memory
        )
    {
        bool success;
        uint256 size;
        bytes memory result;
        assembly {
            success := call(gas, _executionManagerAddress, 0, add(_callbytes, 0x20), mload(_callbytes), 0, 0)

            size := returndatasize()
            result := mload(0x40)
            mstore(0x40, add(result, add(size, 0x20)))
            mstore(result, size)
            returndatacopy(add(result, 0x20), 0, size)
        }

        if (success == false) {
            revert(string(result));
        } else {
            return(result);
        }
    }
}
