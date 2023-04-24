// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

// https://chenglongma.com/10/simple-keccak/
// https://github.com/firefly/wallet/blob/master/source/libs/ethers/src/keccak256.c

library Lib_Keccak256 {
  struct CTX {
    uint64[25] A;
  }

  function get_round_constant(uint round) internal pure returns (uint64) {
    uint64 result = 0;
    uint8 roundInfo = uint8(0x7421587966164852535d4f3f26350c0e5579211f705e1a01 >> (round*8));
    result |= (uint64(roundInfo) << (63-6)) & (1 << 63);
    result |= (uint64(roundInfo) << (31-5)) & (1 << 31);
    result |= (uint64(roundInfo) << (15-4)) & (1 << 15);
    result |= (uint64(roundInfo) << (7-3)) & (1 << 7);
    result |= (uint64(roundInfo) << (3-2)) & (1 << 3);
    result |= (uint64(roundInfo) << (1-1)) & (1 << 1);
    result |= (uint64(roundInfo) << (0-0)) & (1 << 0);
    return result;
  }

  function keccak_theta_rho_pi(CTX memory c) internal pure {
    uint64 C0 = c.A[0] ^ c.A[5] ^ c.A[10] ^ c.A[15] ^ c.A[20];
    uint64 C1 = c.A[1] ^ c.A[6] ^ c.A[11] ^ c.A[16] ^ c.A[21];
    uint64 C2 = c.A[2] ^ c.A[7] ^ c.A[12] ^ c.A[17] ^ c.A[22];
    uint64 C3 = c.A[3] ^ c.A[8] ^ c.A[13] ^ c.A[18] ^ c.A[23];
    uint64 C4 = c.A[4] ^ c.A[9] ^ c.A[14] ^ c.A[19] ^ c.A[24];
    uint64 D0 = (C1 << 1) ^ (C1 >> 63) ^ C4;
    uint64 D1 = (C2 << 1) ^ (C2 >> 63) ^ C0;
    uint64 D2 = (C3 << 1) ^ (C3 >> 63) ^ C1;
    uint64 D3 = (C4 << 1) ^ (C4 >> 63) ^ C2;
    uint64 D4 = (C0 << 1) ^ (C0 >> 63) ^ C3;
    c.A[0] ^= D0;
    uint64 A1 = ((c.A[1] ^ D1) << 1) ^ ((c.A[1] ^ D1) >> (64-1));
    c.A[1] = ((c.A[6] ^ D1) << 44) ^ ((c.A[6] ^ D1) >> (64-44));
    c.A[6] = ((c.A[9] ^ D4) << 20) ^ ((c.A[9] ^ D4) >> (64-20));
    c.A[9] = ((c.A[22] ^ D2) << 61) ^ ((c.A[22] ^ D2) >> (64-61));
    c.A[22] = ((c.A[14] ^ D4) << 39) ^ ((c.A[14] ^ D4) >> (64-39));
    c.A[14] = ((c.A[20] ^ D0) << 18) ^ ((c.A[20] ^ D0) >> (64-18));
    c.A[20] = ((c.A[2] ^ D2) << 62) ^ ((c.A[2] ^ D2) >> (64-62));
    c.A[2] = ((c.A[12] ^ D2) << 43) ^ ((c.A[12] ^ D2) >> (64-43));
    c.A[12] = ((c.A[13] ^ D3) << 25) ^ ((c.A[13] ^ D3) >> (64-25));
    c.A[13] = ((c.A[19] ^ D4) << 8) ^ ((c.A[19] ^ D4) >> (64-8));
    c.A[19] = ((c.A[23] ^ D3) << 56) ^ ((c.A[23] ^ D3) >> (64-56));
    c.A[23] = ((c.A[15] ^ D0) << 41) ^ ((c.A[15] ^ D0) >> (64-41));
    c.A[15] = ((c.A[4] ^ D4) << 27) ^ ((c.A[4] ^ D4) >> (64-27));
    c.A[4] = ((c.A[24] ^ D4) << 14) ^ ((c.A[24] ^ D4) >> (64-14));
    c.A[24] = ((c.A[21] ^ D1) << 2) ^ ((c.A[21] ^ D1) >> (64-2));
    c.A[21] = ((c.A[8] ^ D3) << 55) ^ ((c.A[8] ^ D3) >> (64-55));
    c.A[8] = ((c.A[16] ^ D1) << 45) ^ ((c.A[16] ^ D1) >> (64-45));
    c.A[16] = ((c.A[5] ^ D0) << 36) ^ ((c.A[5] ^ D0) >> (64-36));
    c.A[5] = ((c.A[3] ^ D3) << 28) ^ ((c.A[3] ^ D3) >> (64-28));
    c.A[3] = ((c.A[18] ^ D3) << 21) ^ ((c.A[18] ^ D3) >> (64-21));
    c.A[18] = ((c.A[17] ^ D2) << 15) ^ ((c.A[17] ^ D2) >> (64-15));
    c.A[17] = ((c.A[11] ^ D1) << 10) ^ ((c.A[11] ^ D1) >> (64-10));
    c.A[11] = ((c.A[7] ^ D2) << 6) ^ ((c.A[7] ^ D2) >> (64-6));
    c.A[7] = ((c.A[10] ^ D0) << 3) ^ ((c.A[10] ^ D0) >> (64-3));
    c.A[10] = A1;
  }

  function keccak_chi(CTX memory c) internal pure {
    uint i;
    uint64 A0;
    uint64 A1;
    uint64 A2;
    uint64 A3;
    uint64 A4;
    for (i = 0; i < 25; i+=5) {
      A0 = c.A[0 + i];
      A1 = c.A[1 + i];
      A2 = c.A[2 + i];
      A3 = c.A[3 + i];
      A4 = c.A[4 + i];
      c.A[0 + i] ^= ~A1 & A2;
      c.A[1 + i] ^= ~A2 & A3;
      c.A[2 + i] ^= ~A3 & A4;
      c.A[3 + i] ^= ~A4 & A0;
      c.A[4 + i] ^= ~A0 & A1;
    }
  }

  function keccak_init(CTX memory c) internal pure {
    // is this needed?
    uint i;
    for (i = 0; i < 25; i++) {
      c.A[i] = 0;
    }
  }

  function sha3_xor_input(CTX memory c, bytes memory dat) internal pure {
    for (uint i = 0; i < 17; i++) {
      uint bo = i*8;
      c.A[i] ^= uint64(uint8(dat[bo+7])) << 56 |
                uint64(uint8(dat[bo+6])) << 48 |
                uint64(uint8(dat[bo+5])) << 40 |
                uint64(uint8(dat[bo+4])) << 32 |
                uint64(uint8(dat[bo+3])) << 24 |
                uint64(uint8(dat[bo+2])) << 16 |
                uint64(uint8(dat[bo+1])) << 8 |
                uint64(uint8(dat[bo+0])) << 0;
    }
  }

  function sha3_permutation(CTX memory c) internal pure {
    uint round;
    for (round = 0; round < 24; round++) {
      keccak_theta_rho_pi(c);
      keccak_chi(c);
      // keccak_iota
      c.A[0] ^= get_round_constant(round);
    }
  }

  // https://stackoverflow.com/questions/2182002/convert-big-endian-to-little-endian-in-c-without-using-provided-func
  function flip(uint64 val) internal pure returns (uint64) {
    val = ((val << 8) & 0xFF00FF00FF00FF00 ) | ((val >> 8) & 0x00FF00FF00FF00FF );
    val = ((val << 16) & 0xFFFF0000FFFF0000 ) | ((val >> 16) & 0x0000FFFF0000FFFF );
    return (val << 32) | (val >> 32);
  }

  function get_hash(CTX memory c) internal pure returns (bytes32) {
    return bytes32((uint256(flip(c.A[0])) << 192) |
                   (uint256(flip(c.A[1])) << 128) |
                   (uint256(flip(c.A[2])) << 64) |
                   (uint256(flip(c.A[3])) << 0));
  }

}