// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

import "./lib/Lib_Keccak256.sol";
import "hardhat/console.sol";
//import "./lib/Lib_MerkleTrie.sol";

import { Lib_RLPReader } from "./lib/Lib_RLPReader.sol";
import { Lib_BytesUtils } from "./lib/Lib_BytesUtils.sol";

contract MIPSMemory {
  // TODO: the trie library should read and write from this as appropriate
  mapping(bytes32 => bytes) public trie;

  uint256 constant TREE_RADIX = 16;
  uint256 constant BRANCH_NODE_LENGTH = TREE_RADIX + 1;
  uint256 constant LEAF_OR_EXTENSION_NODE_LENGTH = 2;

  uint8 constant PREFIX_EXTENSION_EVEN = 0;
  uint8 constant PREFIX_EXTENSION_ODD = 1;
  uint8 constant PREFIX_LEAF_EVEN = 2;
  uint8 constant PREFIX_LEAF_ODD = 3;

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

  function fb(bytes memory dat) internal view returns (uint32) {
    require(dat.length == 4, "wrong length value");
    uint32 ret = uint32(uint8(dat[0])) << 24 |
                 uint32(uint8(dat[1])) << 16 |
                 uint32(uint8(dat[2])) << 8 |
                 uint32(uint8(dat[3]));
    return ret;
  }

  /*mapping(bytes32 => mapping(uint32 => bytes)) proofs;

  function AddMerkleProof(bytes32 stateHash, uint32 addr, bytes calldata proof) public {
    // validate proof
    Lib_MerkleTrie.get(tb(addr), proof, stateHash);
    proofs[stateHash][addr] = proof;
  }*/

  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 value) public view returns (bytes32) {
    require(addr & 3 == 0, "write memory must be 32-bit aligned");

    // TODO: this can't delete nodes. modify the client to never delete
    //return Lib_MerkleTrie.update(tb(addr), tb(value), proofs[stateHash][addr], stateHash);
    return stateHash;
  }

  function WriteBytes32(bytes32 stateHash, uint32 addr, bytes32 val) public view returns (bytes32) {
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

    /*bool exists;
    bytes memory value;
    (exists, value) = Lib_MerkleTrie.get(tb(addr), proofs[stateHash][addr], stateHash);

    if (!exists) {
      // this is uninitialized memory
      return 0;
    } else {
      return fb(value);
    }*/
    bytes memory key = Lib_BytesUtils.toNibbles(tb(addr>>2));
    bytes32 cnode = stateHash;
    uint256 idx = 0;

    while (true) {
      Lib_RLPReader.RLPItem[] memory node = Lib_RLPReader.readList(trie[cnode]);
      if (node.length == BRANCH_NODE_LENGTH) {
        //revert("node length bnl");
        uint8 branchKey = uint8(key[idx]);
        if (idx == key.length-1) {
          //if (addr != 0xc0000080) revert("here");
          Lib_RLPReader.RLPItem[] memory lp = Lib_RLPReader.readList(node[branchKey]);
          require(lp.length == 2, "wrong RLP list length");
          return fb(Lib_RLPReader.readBytes(lp[1]));
        } else {
          cnode = Lib_RLPReader.readBytes32(node[branchKey]);
          idx += 1;
        }
      } else if (node.length == LEAF_OR_EXTENSION_NODE_LENGTH) {
        bytes memory path = Lib_BytesUtils.toNibbles(Lib_RLPReader.readBytes(node[0]));
        uint8 prefix = uint8(path[0]);
        if (prefix == PREFIX_LEAF_EVEN || prefix == PREFIX_LEAF_ODD) {
          // TODO: check match
          //return fb(Lib_RLPReader.readList(node[1]));
          // broken
        } else if (prefix == PREFIX_EXTENSION_EVEN || prefix == PREFIX_EXTENSION_ODD) {
          // TODO: check match
          if (prefix == PREFIX_EXTENSION_EVEN) {
            idx += path.length - 2;
          } else {
            idx += path.length - 1;
          }
          cnode = Lib_RLPReader.readBytes32(node[1]);
        }
      } else {
        revert("node in trie broken");
      }
    }
    

  }
}