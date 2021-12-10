// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

import "./lib/Lib_Keccak256.sol";
import "./lib/Lib_MerkleTrie.sol";
import { Lib_BytesUtils } from "./lib/Lib_BytesUtils.sol";

contract MIPSMemory {
  function AddTrieNode(bytes calldata anything) public {
    Lib_MerkleTrie.GetTrie()[keccak256(anything)] = anything;
  }

  struct Preimage {
    uint64 length;
    mapping(uint => uint64) data;
  }

  mapping(bytes32 => Preimage) public preimage;

  function MissingPreimageRevert(bytes32 outhash, uint offset) internal pure {
    Lib_BytesUtils.revertWithHex(abi.encodePacked(outhash, offset));
  }

  function GetPreimageLength(bytes32 outhash) public view returns (uint32) {
    uint64 data = preimage[outhash].length;
    if (data == 0) {
      MissingPreimageRevert(outhash, 0);
    }
    return uint32(data);
  }

  function GetPreimage(bytes32 outhash, uint offset) public view returns (uint32) {
    uint64 data = preimage[outhash].data[offset];
    if (data == 0) {
      MissingPreimageRevert(outhash, offset);
    }
    return uint32(data);
  }

  function AddPreimage(bytes calldata anything, uint offset) public {
    require(offset & 3 == 0, "offset must be 32-bit aligned");
    uint len = anything.length;
    require(offset < len, "offset can't be longer than input");
    Preimage storage p = preimage[keccak256(anything)];
    require(p.length == 0 || uint32(p.length) == len, "length is somehow wrong");
    p.length = (1 << 32) | uint64(uint32(len));
    p.data[offset] = (1 << 32) |
                     ((len <= (offset+0) ? 0 : uint32(uint8(anything[offset+0]))) << 24) |
                     ((len <= (offset+1) ? 0 : uint32(uint8(anything[offset+1]))) << 16) |
                     ((len <= (offset+2) ? 0 : uint32(uint8(anything[offset+2]))) << 8) |
                     ((len <= (offset+3) ? 0 : uint32(uint8(anything[offset+3]))) << 0);
  }

  // one per owner (at a time)

  struct LargePreimage {
    uint offset;
    uint len;
    uint32 data;
  }
  mapping(address => LargePreimage) public largePreimage;
  // sadly due to soldiity limitations this can't be in the LargePreimage struct
  mapping(address => uint64[25]) public largePreimageState;

  function AddLargePreimageInit(uint offset) public {
    require(offset & 3 == 0, "offset must be 32-bit aligned");
    Lib_Keccak256.CTX memory c;
    Lib_Keccak256.keccak_init(c);
    largePreimageState[msg.sender] = c.A;
    largePreimage[msg.sender].offset = offset;
    largePreimage[msg.sender].len = 0;
  }

  // input 136 bytes, as many times as you'd like
  // Uses about 500k gas, 3435 gas/byte
  function AddLargePreimageUpdate(bytes calldata dat) public {
    require(dat.length == 136, "update must be in multiples of 136");
    // sha3_process_block
    Lib_Keccak256.CTX memory c;
    c.A = largePreimageState[msg.sender];

    int offset = int(largePreimage[msg.sender].offset) - int(largePreimage[msg.sender].len);
    if (offset >= 0 && offset < 136) {
      largePreimage[msg.sender].data = fbo(dat, uint(offset));
    }
    Lib_Keccak256.sha3_xor_input(c, dat);
    Lib_Keccak256.sha3_permutation(c);
    largePreimageState[msg.sender] = c.A;
    largePreimage[msg.sender].len += 136;
  }

  function AddLargePreimageFinal(bytes calldata idat) public view returns (bytes32, uint32, uint32) {
    require(idat.length < 136, "final must be less than 136");
    int offset = int(largePreimage[msg.sender].offset) - int(largePreimage[msg.sender].len);
    require(offset < int(idat.length), "offset must be less than length");
    Lib_Keccak256.CTX memory c;
    c.A = largePreimageState[msg.sender];

    bytes memory dat = new bytes(136);
    for (uint i = 0; i < idat.length; i++) {
      dat[i] = idat[i];
    }
    uint len = largePreimage[msg.sender].len + idat.length;
    uint32 data = largePreimage[msg.sender].data;
    if (offset >= 0) {
      data = fbo(dat, uint(offset));
    }
    dat[135] = bytes1(uint8(0x80));
    dat[idat.length] |= bytes1(uint8(0x1));

    Lib_Keccak256.sha3_xor_input(c, dat);
    Lib_Keccak256.sha3_permutation(c);

    bytes32 outhash = Lib_Keccak256.get_hash(c);
    require(len < 0x10000000, "max length is 32-bit");
    return (outhash, uint32(len), data);
  }

  function AddLargePreimageFinalSaved(bytes calldata idat) public {
    bytes32 outhash;
    uint32 len;
    uint32 data;
    (outhash, len, data) = AddLargePreimageFinal(idat);

    Preimage storage p = preimage[outhash];
    require(p.length == 0 || uint32(p.length) == len, "length is somehow wrong");
    require(largePreimage[msg.sender].offset < len, "offset is somehow beyond length");
    p.length = (1 << 32) | uint64(len);
    p.data[largePreimage[msg.sender].offset] = (1 << 32) | data;
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

  function fbo(bytes memory dat, uint offset) internal pure returns (uint32) {
    uint32 ret = uint32(uint8(dat[offset+0])) << 24 |
                 uint32(uint8(dat[offset+1])) << 16 |
                 uint32(uint8(dat[offset+2])) << 8 |
                 uint32(uint8(dat[offset+3]));
    return ret;
  }

  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 value) public returns (bytes32) {
    require(addr & 3 == 0, "write memory must be 32-bit aligned");
    return Lib_MerkleTrie.update(tb(addr>>2), tb(value), stateHash);
  }

  function WriteBytes32(bytes32 stateHash, uint32 addr, bytes32 val) public returns (bytes32) {
    for (uint32 i = 0; i < 32; i += 4) {
      uint256 tv = uint256(val>>(224-(i*8)));
      stateHash = WriteMemory(stateHash, addr+i, uint32(tv));
    }
    return stateHash;
  }

  // TODO: refactor writeMemory function to not need these
  event DidStep(bytes32 stateHash);
  function WriteMemoryWithReceipt(bytes32 stateHash, uint32 addr, uint32 value) public {
    bytes32 newStateHash = WriteMemory(stateHash, addr, value);
    emit DidStep(newStateHash);
  }

  function WriteBytes32WithReceipt(bytes32 stateHash, uint32 addr, bytes32 value) public {
    bytes32 newStateHash = WriteBytes32(stateHash, addr, value);
    emit DidStep(newStateHash);
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
      if (pihash == keccak256("")) {
        // both the length and any data are 0
        return 0;
      }
      if (addr == 0x31000000) {
        return uint32(GetPreimageLength(pihash));
      }
      return GetPreimage(pihash, addr-0x31000004);
    }

    bool exists;
    bytes memory value;
    (exists, value) = Lib_MerkleTrie.get(tb(addr>>2), stateHash);

    if (!exists) {
      // this is uninitialized memory
      return 0;
    } else {
      return fb(value);
    }
  }

}