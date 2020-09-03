pragma solidity ^0.5.0;

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
            callbytes,
            gasleft()
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
            callbytes,
            gasleft()
        );

        address ret;
        assembly {
            ret := mload(add(returndata, 0x20))
        }

        return ret;
    }

    function ovmCHAINID(
        address _executionManagerAddress
    )
        internal
        returns (uint256)
    {
        bytes memory callbytes = abi.encodePacked(
            bytes4(keccak256("ovmCHAINID()"))
        );

        bytes memory returndata = _ovmcall(
            _executionManagerAddress,
            callbytes,
            gasleft()
        );

        uint256 ret;
        assembly {
            ret := mload(add(returndata, 0x20))
        }

        return ret;
    }

    function ovmCALL(
        address _executionManagerAddress,
        address _target,
        bytes memory _calldata,
        uint256 _gasLimit
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
            callbytes,
            _gasLimit
        );
    }

    function ovmCREATE(
        address _executionManagerAddress,
        bytes memory _bytecode,
        uint256 _gasLimit
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
            callbytes,
            _gasLimit
        );

        address ret;
        assembly {
            ret := mload(add(returndata, 0x20))
        }

        return ret;
    }

    function _ovmcall(
        address _executionManagerAddress,
        bytes memory _callbytes,
        uint256 _gasLimit
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
            success := call(_gasLimit, _executionManagerAddress, 0, add(_callbytes, 0x20), mload(_callbytes), 0, 0)

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
