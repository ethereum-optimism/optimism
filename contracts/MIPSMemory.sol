// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

contract MIPSMemory {
  // This state is global
  mapping(bytes32 => mapping (uint32 => uint64)) public state;
  mapping(bytes32 => bytes) public preimage;

  function AddPreimage(bytes calldata anything) public {
    preimage[keccak256(anything)] = anything;
  }

  function AddMerkleState(bytes32 stateHash, uint32 addr, uint32 value, string calldata proof) public {
    // TODO: check proof
    state[stateHash][addr] = (1 << 32) | value;
  }

  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 val) public pure returns (bytes32) {
    // TODO: does the stateHash mutation
    require(addr & 3 == 0, "write memory must be 32-bit aligned");
  }

  // needed for preimage oracle
  function ReadBytes32(bytes32 stateHash, uint32 addr) public view returns (bytes32) {
    uint256 ret = 0;
    for (uint32 i = 0; i < 32; i += 4) {
      ret <<= 32;
      ret |= uint256(ReadMemory(stateHash, addr+i));
    }
    return bytes32(ret);
  }

  function ReadMemory(bytes32 stateHash, uint32 addr) public view returns (uint32) {
    require(addr & 3 == 0, "read memory must be 32-bit aligned");

    // zero register is always 0
    if (addr == 0xc0000000) {
      return 0;
    }

    // MMIO preimage oracle
    if (addr >= 0x31000000 && addr < 0x32000000) {
      bytes32 pihash = ReadBytes32(stateHash, 0x30001000);
      if (addr == 0x31000000) {
        return uint32(preimage[pihash].length);
      }
      uint offset = addr-0x31000004;
      uint8 a0 = uint8(preimage[pihash][offset]);
      uint8 a1 = uint8(preimage[pihash][offset+1]);
      uint8 a2 = uint8(preimage[pihash][offset+2]);
      uint8 a3 = uint8(preimage[pihash][offset+3]);
      return (uint32(a0) << 24) |
             (uint32(a1) << 16) |
             (uint32(a2) << 8) |
             (uint32(a3) << 0);
    }

    uint64 ret = state[stateHash][addr];
    require((ret >> 32) == 1, "memory was not initialized");
    return uint32(ret);
  }
}