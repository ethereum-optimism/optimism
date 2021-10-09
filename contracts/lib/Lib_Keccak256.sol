// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

// https://chenglongma.com/10/simple-keccak/
// https://github.com/firefly/wallet/blob/master/source/libs/ethers/src/keccak256.c

library Lib_Keccak256 {
  struct CTX {
    uint64[25] A;
  }

  bytes public constant round_constant = hex"011a5e701f2179550e0c35263f4f5d535248166679582174";
  bytes public constant pi_transform = hex"010609160e14020c0d13170f0418150810050312110b070a";
  bytes public constant rho_transform = hex"013e1c1b242c063714030a2b1927292d0f150812023d380e";

  function ROTL64(uint64 qword, uint64 n) internal pure returns (uint64) {
    return ((qword) << (n) ^ ((qword) >> (64 - (n))));
  }

  function get_round_constant(uint round) internal pure returns (uint64) {
    uint64 result = 0;
    uint8 roundInfo = uint8(round_constant[round]);
    // TODO: write this without control flow
    if (roundInfo & (1 << 6) != 0) { result |= (1 << 63); }
    if (roundInfo & (1 << 5) != 0) { result |= (1 << 31); }
    if (roundInfo & (1 << 4) != 0) { result |= (1 << 15); }
    if (roundInfo & (1 << 3) != 0) { result |= (1 << 7); }
    if (roundInfo & (1 << 2) != 0) { result |= (1 << 3); }
    if (roundInfo & (1 << 1) != 0) { result |= (1 << 1); }
    if (roundInfo & (1 << 0) != 0) { result |= (1 << 0); }
    return result;
  }

  function keccak_theta(CTX memory c) internal pure {
    uint64[5] memory C;
    uint64[5] memory D;
    uint i;
    uint j;
    for (i = 0; i < 5; i++) {
      C[i] = c.A[i];
      for (j = 5; j < 25; j += 5) { C[i] ^= c.A[i + j]; }
    }
    for (i = 0; i < 5; i++) {
      D[i] = ROTL64(C[(i + 1) % 5], 1) ^ C[(i + 4) % 5];
    }
    for (i = 0; i < 5; i++) {
      for (j = 0; j < 25; j += 5) { c.A[i + j] ^= D[i]; }
    }
  }

  function keccak_rho(CTX memory c) internal pure {
    uint i;
    for (i = 1; i < 25; i++) {
      // TODO: unroll this?
      c.A[i] = ROTL64(c.A[i], uint8(rho_transform[i-1]));
    }
  }

  function keccak_pi(CTX memory c) internal pure {
    uint64 A1 = c.A[1];
    uint i;
    for (i = 1; i < 24; i++) {
      // TODO: unroll this?
      c.A[uint8(pi_transform[i-1])] = c.A[uint8(pi_transform[i])];
    }
    c.A[10] = A1;
  }

  function keccak_chi(CTX memory c) internal pure {
    uint i;
    uint64 A0;
    uint64 A1;
    for (i = 0; i < 25; i+=5) {
      A0 = c.A[0 + i];
      A1 = c.A[1 + i];
      c.A[0 + i] ^= ~A1 & c.A[2 + i];
      c.A[1 + i] ^= ~c.A[2 + i] & c.A[3 + i];
      c.A[2 + i] ^= ~c.A[3 + i] & c.A[4 + i];
      c.A[3 + i] ^= ~c.A[4 + i] & A0;
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

  function sha3_permutation(CTX memory c) internal pure {
    uint round;
    for (round = 0; round < 24; round++) {
      keccak_theta(c);
      keccak_rho(c);
      keccak_pi(c);
      keccak_chi(c);
      // keccak_iota
      c.A[0] ^= get_round_constant(round);
    }
  }

}