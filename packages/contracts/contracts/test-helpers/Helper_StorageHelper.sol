// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

contract Helper_StorageHelper {
    function setStorage(
        bytes32 _key,
        bytes32 _val
    )
        public
    {
        assembly {
            sstore(_key, _val)
        }
    }

    struct BasicStruct {
        uint256 _structUint256;
        address _structAddress;
        bytes32 _structBytes32;
    }

    uint8 public _uint8;     // slot 0
    bytes32 _spacer1;        // to avoid slot packing, unused

    uint64 public _uint64;   // slot 2
    bytes32 _spacer2;        // to avoid slot packing, unused

    uint256 public _uint256; // slot 4
    bytes32 _spacer3;        // to avoid slot packing, unused

    bytes1 public _bytes1;   // slot 6
    bytes32 _spacer4;        // to avoid slot packing, unused

    bytes8 public _bytes8;   // slot 8
    bytes32 _spacer5;        // to avoid slot packing, unused

    bytes32 public _bytes32; // slot 10
    bytes32 _spacer6;        // to avoid slot packing, unused

    bool public _bool;       // slot 12
    bytes32 _spacer7;        // to avoid slot packing, unused

    address public _address; // slot 14
    bytes32 _spacer8;        // to avoid slot packing, unused

    bytes public _bytes;     // slot 16
    string public _string;   // slot 17

    BasicStruct public _struct; // slot 18,19,20

    // Pack into (bytes11,bool,address)
    address public _packedAddress; // slot 21
    bool public _packedBool;       // slot 21
    bytes11 public _packedBytes11; // slot 21

    // Pack into (address,bool,bytes11)
    bytes11 public _otherPackedBytes11; // slot 22
    bool public _otherPackedBool;       // slot 22
    address public _otherPackedAddress; // slot 22

    // Unsupported types.
    mapping (uint256 => uint256) _uint256ToUint256Map;
    uint256[] _uint256Array;
}
