// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

contract TestScript {
    struct Input {
        uint8 id;
        bytes data;
        uint32[] slice;
        uint256[3] array;
    }

    struct Output {
        uint8 id;
        bytes data;
        uint32[] slice;
        uint256[3] array;
    }

    function _run(Input memory _input) public pure returns (Output memory) {
        return Output({ id: 0x02, data: _input.data, slice: _input.slice, array: _input.array });
    }

    function runWithBytes(bytes memory _input) public pure returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = _run(input);
        return abi.encode(output);
    }
}
