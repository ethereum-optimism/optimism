// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

import "./lib/Lib_Keccak256.sol";
import "./lib/Lib_MerkleTrie.sol";

contract MIPSMemory {
  // TODO: the trie library should read and write from this as appropriate
  mapping(bytes32 => bytes) public trie;

  function AddTrieNode(bytes calldata anything) public {
    trie[keccak256(anything)] = anything;
  }

  // TODO: replace with mapping(bytes32 => mapping(uint, bytes4))
  // to only save the part we care about
  mapping(bytes32 => bytes) public preimage;

  function AddPreimage(bytes calldata anything) public {
    preimage[keccak256(anything)] = anything;
  }

  // one per owner (at a time)
  mapping(address => uint64[25]) public largePreimage;
  // TODO: also track the offset into the largePreimage to know what to store

  function AddLargePreimageInit() public {
    Lib_Keccak256.CTX memory c;
    Lib_Keccak256.keccak_init(c);
    largePreimage[msg.sender] = c.A;
  }

  // TODO: input 136 bytes, as many times as you'd like
  // Uses about 1M gas, 7352 gas/byte
  function AddLargePreimageUpdate(uint64[17] calldata data) public {
    // sha3_process_block
    Lib_Keccak256.CTX memory c;
    c.A = largePreimage[msg.sender];
    for (uint i = 0; i < 17; i++) {
      c.A[i] ^= data[i];
    }
    Lib_Keccak256.sha3_permutation(c);
    largePreimage[msg.sender] = c.A;
  }

  // TODO: input <136 bytes and do the end of hash | 0x01 / | 0x80
  function AddLargePreimageFinal() public view returns (bytes32) {
    Lib_Keccak256.CTX memory c;
    c.A = largePreimage[msg.sender];
    // TODO: do this properly and save the hash
    // when this is updated, it won't be "view"
    return Lib_Keccak256.get_hash(c);
  }

  function tb(uint32 dat) internal pure returns (bytes memory) {
    bytes memory ret = new bytes(4);
    ret[0] = bytes1(uint8(dat >> 24));
    ret[1] = bytes1(uint8(dat >> 16));
    ret[2] = bytes1(uint8(dat >> 8));
    ret[3] = bytes1(uint8(dat >> 0));
    return ret;
  }

  function fb(bytes memory dat) internal pure returns (uint32) {
    require(dat.length == 4, "wrong length value");
    uint32 ret = uint32(uint8(dat[0])) << 24 |
                 uint32(uint8(dat[1])) << 16 |
                 uint32(uint8(dat[2])) << 8 |
                 uint32(uint8(dat[3]));
    return ret;
  }

  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 value) public returns (bytes32) {
    require(addr & 3 == 0, "write memory must be 32-bit aligned");
    return Lib_MerkleTrie.update(tb(addr>>2), tb(value), trie, stateHash);
  }

  event DidStep(bytes32 stateHash);
  function WriteMemoryWithReceipt(bytes32 stateHash, uint32 addr, uint32 value) public {
    bytes32 newStateHash = WriteMemory(stateHash, addr, value);
    emit DidStep(newStateHash);
  }

  function WriteBytes32(bytes32 stateHash, uint32 addr, bytes32 val) public returns (bytes32) {
    for (uint32 i = 0; i < 32; i += 4) {
      uint256 tv = uint256(val>>(224-(i*8)));
      stateHash = WriteMemory(stateHash, addr+i, uint32(tv));
    }
    return stateHash;
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
    if (addr >= 0xB1000000 && addr < 0xB2000000) {
      bytes32 pihash = ReadBytes32(stateHash, 0xB0001000);
      if (addr == 0xB1000000) {
        return uint32(preimage[pihash].length);
      }
      uint offset = addr-0xB1000004;
      uint8 a0 = uint8(preimage[pihash][offset]);
      uint8 a1 = uint8(preimage[pihash][offset+1]);
      uint8 a2 = uint8(preimage[pihash][offset+2]);
      uint8 a3 = uint8(preimage[pihash][offset+3]);
      return (uint32(a0) << 24) |
             (uint32(a1) << 16) |
             (uint32(a2) << 8) |
             (uint32(a3) << 0);
    }

    bool exists;
    bytes memory value;
    (exists, value) = Lib_MerkleTrie.get(tb(addr>>2), trie, stateHash);

    if (!exists) {
      // this is uninitialized memory
      return 0;
    } else {
      return fb(value);
    }
  }

}