// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Test.sol";

import { ClonesWithImmutableArgs } from "@cwia/ClonesWithImmutableArgs.sol";

import { Clone } from "../libraries/Clone.sol";

contract ExampleClone is Clone {
    uint256 argOffset;

    constructor(uint256 _argOffset) {
        argOffset = _argOffset;
    }

    function addressArg() public view returns (address) {
        return _getArgAddress(argOffset);
    }

    function uintArg() public view returns (uint256) {
        return _getArgUint256(argOffset);
    }

    function fixedBytesArg() public view returns (bytes32) {
        return _getArgFixedBytes(argOffset);
    }

    function uintArrayArg(uint64 arrLen) public view returns (uint256[] memory) {
        return _getArgUint256Array(argOffset, arrLen);
    }

    function dynBytesArg(uint64 arrLen) public view returns (bytes memory) {
        return _getArgDynBytes(argOffset, arrLen);
    }

    function uint64Arg() public view returns (uint64) {
        return _getArgUint64(argOffset);
    }

    function uint8Arg() public view returns (uint8) {
        return _getArgUint8(argOffset);
    }
}

contract ExampleCloneFactory {
    using ClonesWithImmutableArgs for address;

    ExampleClone public implementation;

    constructor(ExampleClone implementation_) {
        implementation = implementation_;
    }

    function createAddressClone(address arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createUintClone(uint256 arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createFixedBytesClone(bytes32 arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createUintArrayClone(uint256[] memory arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createDynBytesClone(bytes memory arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createUint64Clone(uint64 arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createUint8Clone(uint8 arg) external returns (ExampleClone clone) {
        bytes memory data = abi.encodePacked(arg);
        clone = ExampleClone(address(implementation).clone(data));
    }

    function createClone(bytes memory randomCalldata) external returns (ExampleClone clone) {
        clone = ExampleClone(address(implementation).clone(randomCalldata));
    }
}

contract Clones_Test is Test {
    function testFuzz_clone_addressArg_succeeds(uint256 argOffset, address param) public {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createAddressClone(param);
        address fetched = clone.addressArg();
        assertEq(fetched, param);
    }

    function testFuzz_clone_uintArg_succeeds(uint256 argOffset, uint256 param) public {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createUintClone(param);
        uint256 fetched = clone.uintArg();
        assertEq(fetched, param);
    }

    function testFuzz_clone_fixedBytesArg_succeeds(uint256 argOffset, bytes32 param) public {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createFixedBytesClone(param);
        bytes32 fetched = clone.fixedBytesArg();
        assertEq(fetched, param);
    }

    function testFuzz_clone_uintArrayArg_succeeds(uint256 argOffset, uint256[] memory param)
        public
    {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createUintArrayClone(param);
        uint256[] memory fetched = clone.uintArrayArg(uint64(param.length));
        assertEq(fetched, param);
    }

    function testFuzz_clone_dynBytesArg_succeeds(uint256 argOffset, bytes memory param) public {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createDynBytesClone(param);
        bytes memory fetched = clone.dynBytesArg(uint64(param.length));
        assertEq(fetched, param);
    }

    function testFuzz_clone_uint64Arg_succeeds(uint256 argOffset, uint64 param) public {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createUint64Clone(param);
        uint64 fetched = clone.uint64Arg();
        assertEq(fetched, param);
    }

    function testFuzz_clone_uint8Arg_succeeds(uint256 argOffset, uint8 param) public {
        ExampleClone implementation = new ExampleClone(argOffset);
        ExampleCloneFactory factory = new ExampleCloneFactory(implementation);
        ExampleClone clone = factory.createUint8Clone(param);
        uint8 fetched = clone.uint8Arg();
        assertEq(fetched, param);
    }
}
