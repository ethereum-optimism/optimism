// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

contract TestHelpers_BasicReturnContract {
    fallback()
        external
    {}

    function empty()
        public
    {}

    function getBoolean()
        public
        returns (
            bool _out1
        )
    {}

    function getUint256()
        public
        returns (
            uint256 _out1
        )
    {}

    function getBytes32()
        public
        returns (
            bytes32 _out1
        )
    {}

    function getBytes()
        public
        returns (
            bytes memory _out1
        )
    {}

    function getString()
        public
        returns (
            string memory _out1
        )
    {}

    function getInputtedBoolean(
        bool _in1
    )
        public
        returns (
            bool _out1
        )
    {}

    function getInputtedUint256(
        uint256 _in1
    )
        public
        returns (
            uint256 _out1
        )
    {}

    function getInputtedBytes32(
        bytes32 _in1
    )
        public
        returns (
            bytes32 _out1
        )
    {}

    struct StructFixedSize {
        bool valBoolean;
        uint256 valUint256;
        bytes32 valBytes32;
    }

    function getStructFixedSize()
        public
        returns (
            StructFixedSize memory _out1
        )
    {}

    struct StructDynamicSize {
        bytes valBytes;
        string valString;
    }

    function getStructDynamicSize()
        public
        returns (
            StructDynamicSize memory _out1
        )
    {}

    struct StructMixedSize {
        bool valBoolean;
        uint256 valUint256;
        bytes32 valBytes32;
        bytes valBytes;
        string valString;
    }

    function getStructMixedSize()
        public
        returns (
            StructMixedSize memory _out1
        )
    {}

    struct StructNested {
        StructFixedSize valStructFixedSize;
        StructDynamicSize valStructDynamicSize;
    }

    function getStructNested()
        public
        returns (
            StructNested memory _out1
        )
    {}

    function getArrayUint256()
        public
        returns (
            uint256[] memory _out
        )
    {}

    function overloadedFunction(
        uint256 _paramA,
        uint256 _paramB
    )
        public
        returns (
            uint256
        )
    {}

    function overloadedFunction(
        uint256
    )
        public
        returns (
            uint256
        )
    {}
}
