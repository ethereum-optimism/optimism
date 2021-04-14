// @unsupported: ovm
pragma solidity ^0.7.0;

import { console } from "hardhat/console.sol";

contract ChugSplashProxy {
    bytes13 constant public DEPLOY_CODE_PREFIX = 0x600D380380600D6000396000f3;
    bytes32 constant public IMPLEMENTATION_KEY = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF;

    fallback()
        external
    {
        assembly {
            let target := sload(IMPLEMENTATION_KEY)
            calldatacopy(0x0, 0x0, calldatasize())
            let result := delegatecall(gas(), target, 0x0, calldatasize(), 0x0, 0x0)
            returndatacopy(0x0, 0x0, returndatasize())
            switch result
            case 0x0 {
                revert(0x0, returndatasize())
            }
            default {
                return (0x0, returndatasize())
            }
        }
    }

    function implementation()
        public
        returns (
            address
        )
    {
        assembly {
            return(sload(IMPLEMENTATION_KEY), 0x20)
        }
    }

    function setCode(
        bytes memory _code
    )
        public
    {
        // Exit early if we aren't going to have any effect.
        if (keccak256(getCode()) == keccak256(_code)) {
            return;
        }

        bytes memory deploycode = abi.encodePacked(
            DEPLOY_CODE_PREFIX,
            _code
        );

        assembly {
            let created := create(0x0, add(deploycode, 0x20), mload(deploycode))
            sstore(IMPLEMENTATION_KEY, created)
        }
    }

    function getCode()
        public
        view
        returns (
            bytes memory
        )
    {
        bytes memory code;
        assembly {
            let target := sload(IMPLEMENTATION_KEY)
            let size := extcodesize(target)
            code := mload(0x40)
            mstore(0x40, add(code, and(add(add(size, 0x20), 0x1f), not(0x1f))))
            mstore(code, size)
            extcodecopy(target, add(code, 0x20), 0, size)
        }
        return code;
    }

    function setStorage(
        bytes32 _key,
        bytes32 _value
    )
        public
    {
        // Exit early if we aren't going to have any effect.
        if (getStorage(_key) == _value) {
            return;
        }

        assembly {
            sstore(_key, _value)
        }
    }

    function getStorage(
        bytes32 _key
    )
        public
        returns (
            bytes32
        )
    {
        bytes32 val;
        assembly {
            val := sload(_key)
        }

        return val;
    }
}
