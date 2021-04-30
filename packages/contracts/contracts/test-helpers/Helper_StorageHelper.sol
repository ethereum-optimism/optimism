// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

contract Helper_StorageHelper {
    struct BasicStruct {
        uint256 _structUint256;
        address _structAddress;
        bytes32 _structBytes32;
    }

    uint8 _uint8;     // slot 0
    bytes32 _spacer1; // to avoid slot packing, unused

    uint64 _uint64;   // slot 2
    bytes32 _spacer2; // to avoid slot packing, unused

    uint256 _uint256; // slot 4
    bytes32 _spacer3; // to avoid slot packing, unused

    bytes1 _bytes1;   // slot 6
    bytes32 _spacer4; // to avoid slot packing, unused

    bytes8 _bytes8;   // slot 8
    bytes32 _spacer5; // to avoid slot packing, unused

    bytes32 _bytes32; // slot 10
    bytes32 _spacer6; // to avoid slot packing, unused

    bool _bool;       // slot 12
    bytes32 _spacer7; // to avoid slot packing, unused

    address _address; // slot 14
    bytes32 _spacer8; // to avoid slot packing, unused

    bytes _bytes;     // slot 16
    string _string;   // slot 17

    BasicStruct _struct; // slot 18,19,20
}
