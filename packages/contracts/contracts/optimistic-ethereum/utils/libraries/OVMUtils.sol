pragma solidity ^0.5.0;

/**
 * @title OVMUtils
 */
library OVMUtils {
    function ovmCALLER(
        address _executionManagerAddress
    )
        internal
        returns (address)
    {
        bytes memory callbytes = abi.encodePacked(
            bytes4(keccak256("ovmCALLER()"))
        );

        address ret;
        assembly {
            let result := mload(0x40)
            mstore(0x40, add(result, 0x20))

            let success := call(gas, _executionManagerAddress, 0, add(callbytes, 0x20), mload(callbytes), result, 0x20)

            if eq(success, 0) {
                revert(result, returndatasize)
            }

            ret := mload(result)
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

        address ret;
        assembly {
            let result := mload(0x40)
            mstore(0x40, add(result, 0x20))

            let success := call(gas, _executionManagerAddress, 0, add(callbytes, 0x20), mload(callbytes), result, 0x20)

            if eq(success, 0) {
                revert(result, returndatasize)
            }

            ret := mload(result)
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

        assembly {
            let success := call(gas, _executionManagerAddress, 0, add(callbytes, 0x20), mload(callbytes), 0, 0)

            let size := returndatasize()
            let result := mload(0x40)
            mstore(result, size)
            returndatacopy(add(result, 0x20), 0, size)

            if eq(success, 0) {
                revert(result, size)
            }

            mstore(0x40, add(result, add(size, 0x20)))
            return(result, size)
        }
    }
}