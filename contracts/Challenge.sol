// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

import "./lib/Lib_RLPReader.sol";

interface IMIPS {
  function Step(bytes32 stateHash) external view returns (bytes32);
  function m() external pure returns (IMIPSMemory);
}

interface IMIPSMemory {
  function ReadMemory(bytes32 stateHash, uint32 addr) external view returns (uint32);
  function ReadBytes32(bytes32 stateHash, uint32 addr) external view returns (bytes32);
  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 val) external pure returns (bytes32);
  function WriteBytes32(bytes32 stateHash, uint32 addr, bytes32 val) external pure returns (bytes32);
}

contract Challenge {
  address payable immutable owner;

  // the mips machine state transition function
  IMIPS immutable mips;
  IMIPSMemory immutable mem;

  // the program start state
  bytes32 immutable GlobalStartState;

  struct Chal {
    uint256 L;
    uint256 R;
    mapping(uint256 => bytes32) assertedState;
    mapping(uint256 => bytes32) defendedState;
    address payable challenger;
  }
  mapping(uint256 => Chal) challenges;

  constructor(IMIPS imips, bytes32 globalStartState) {
    owner = msg.sender;
    mips = imips;
    mem = imips.m();
    GlobalStartState = globalStartState;
  }

  // allow getting money (and withdrawing the bounty, honor system)
  fallback() external payable {}
  receive() external payable {}
  function withdraw() external {
    require(msg.sender == owner);
    owner.transfer(address(this).balance);
  }

  // create challenge
  uint256 public lastChallengeId = 0;

  function newChallengeTrusted(bytes32 startState, bytes32 finalSystemState, uint256 stepCount) internal returns (uint256) {
    uint256 challengeId = lastChallengeId;
    Chal storage c = challenges[challengeId];
    lastChallengeId += 1;

    // the challenger arrives
    c.challenger = msg.sender;

    // the state is set 
    c.assertedState[0] = startState;
    c.defendedState[0] = startState;
    c.assertedState[stepCount] = finalSystemState;

    // init the binary search
    c.L = 0;
    c.R = stepCount;

    // find me later
    return challengeId;
  }

  function InitiateChallenge(uint blockNumberN, bytes calldata blockHeaderNp1,
        bytes32 assertionRoot, bytes32 finalSystemState, uint256 stepCount) external returns (uint256) {
    require(blockhash(blockNumberN+1) == keccak256(blockHeaderNp1), "end block hash wrong");

    // decode the blocks
    Lib_RLPReader.RLPItem[] memory blockNp1 = Lib_RLPReader.readList(blockHeaderNp1);
    bytes32 parentHash = Lib_RLPReader.readBytes32(blockNp1[0]);
    require(blockhash(blockNumberN) == parentHash, "parent block hash somehow wrong");

    bytes32 newroot = Lib_RLPReader.readBytes32(blockNp1[3]);
    require(assertionRoot != newroot, "asserting that the real state is correct is not a challenge");

    // load starting info into the input oracle
    // we both agree at the beginning
    // the first instruction executed in MIPS should be an access of startState
    // parentblockhash, txhash, coinbase, unclehash, gaslimit, time
    bytes32 startState = GlobalStartState;
    startState = mem.WriteBytes32(startState, 0xB0000000, parentHash);
    startState = mem.WriteBytes32(startState, 0xB0000020, Lib_RLPReader.readBytes32(blockNp1[4]));
    startState = mem.WriteBytes32(startState, 0xB0000040, bytes32(uint256(Lib_RLPReader.readAddress(blockNp1[2]))));
    startState = mem.WriteBytes32(startState, 0xB0000060, Lib_RLPReader.readBytes32(blockNp1[1]));
    startState = mem.WriteBytes32(startState, 0xB0000080, bytes32(Lib_RLPReader.readUint256(blockNp1[9])));
    startState = mem.WriteBytes32(startState, 0xB00000a0, bytes32(Lib_RLPReader.readUint256(blockNp1[11])));

    // confirm the finalSystemHash asserts the state you claim (in $t0-$t7) and the machine is stopped
    // you must load these proofs into MIPS before calling this
    // we disagree at the end
    require(mem.ReadMemory(finalSystemState, 0xC0000080) == 0x5EAD0000, "machine is not stopped in final state (PC == 0x5EAD0000)");
    require(mem.ReadMemory(finalSystemState, 0xB0000800) == 0x1337f00d, "state is not outputted");
    require(mem.ReadBytes32(finalSystemState, 0xB0000804) == assertionRoot, "you are claiming a different state root in machine");

    return newChallengeTrusted(startState, finalSystemState, stepCount);
  }

  // binary search

  function getStepNumber(uint256 challengeId) view public returns (uint256) {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    return (c.L+c.R)/2;
  }

  function ProposeState(uint256 challengeId, bytes32 riscState) external {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    require(c.challenger == msg.sender, "must be challenger");

    uint256 stepNumber = getStepNumber(challengeId);
    require(c.assertedState[stepNumber] == bytes32(0), "state already proposed");
    c.assertedState[stepNumber] = riscState;
  }

  function RespondState(uint256 challengeId, bytes32 riscState) external {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    require(owner == msg.sender, "must be owner");

    uint256 stepNumber = getStepNumber(challengeId);
    require(c.assertedState[stepNumber] != bytes32(0), "challenger state not proposed");
    require(c.defendedState[stepNumber] == bytes32(0), "state already proposed");
    // technically, we don't have to save these states
    // but if we want to prove us right and not just the attacker wrong, we do
    c.defendedState[stepNumber] = riscState;
    if (c.assertedState[stepNumber] == c.defendedState[stepNumber]) {
      // agree
      c.L = stepNumber;
    } else {
      // disagree
      c.R = stepNumber;
    }
  }

  // final payout

  event ChallengerWins(uint256 challengeId);
  event ChallengerLoses(uint256 challengeId);

  function ConfirmStateTransition(uint256 challengeId) external {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    require(c.challenger == msg.sender, "must be challenger");
    require(c.L + 1 == c.R, "binary search not finished");

    require(mips.Step(c.assertedState[c.L]) == c.assertedState[c.R], "wrong asserted state");

    // pay out bounty!!
    c.challenger.transfer(address(this).balance);
    
    emit ChallengerWins(challengeId);
  }
}
