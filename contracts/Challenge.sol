// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;
pragma experimental ABIEncoderV2;

import "./lib/Lib_RLPReader.sol";
import "hardhat/console.sol";

interface IMIPS {
  function Step(bytes32 stateHash) external returns (bytes32);
  function m() external pure returns (IMIPSMemory);
}

interface IMIPSMemory {
  function AddTrieNode(bytes calldata anything) external;
  function ReadMemory(bytes32 stateHash, uint32 addr) external view returns (uint32);
  function ReadBytes32(bytes32 stateHash, uint32 addr) external view returns (bytes32);
  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 val) external returns (bytes32);
  function WriteBytes32(bytes32 stateHash, uint32 addr, bytes32 val) external returns (bytes32);
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
    // TODO: add the block here?
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

  event ChallengeCreate(uint256 challengeId);
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
    emit ChallengeCreate(challengeId);
    return challengeId;
  }

  // helper function to determine what nodes we need
  function CallWithTrieNodes(bytes calldata dat, bytes[] calldata nodes) public {
    for (uint i = 0; i < nodes.length; i++) {
      mem.AddTrieNode(nodes[i]);
    }
    (bool success, bytes memory revertData) = address(this).call(dat);
    // TODO: better way to revert?
    if (!success) {
      uint256 revertDataLength = revertData.length;
      assembly {
          let revertDataStart := add(revertData, 32)
          revert(revertDataStart, revertDataLength)
      }
    }
  }

  function InitiateChallenge(uint blockNumberN, bytes calldata blockHeaderNp1,
        bytes32 assertionRoot, bytes32 finalSystemState, uint256 stepCount) external returns (uint256) {
    bytes32 computedBlockHash = keccak256(blockHeaderNp1);

    // get block hashes, can replace with oracle
    bytes32 blockNumberNHash = blockhash(blockNumberN);
    bytes32 blockNumberNp1Hash = blockhash(blockNumberN+1);

    // TODO: this is only removed for testing. zero security without it
    /*if (blockNumberNHash == bytes32(0) || blockNumberNp1Hash == bytes32(0)) {
      revert("block number too old to challenge");
    }
    require(blockNumberNp1Hash == computedBlockHash, "end block hash wrong");*/

    // decode the blocks
    bytes32 inputHash;
    {
      Lib_RLPReader.RLPItem[] memory blockNp1 = Lib_RLPReader.readList(blockHeaderNp1);
      bytes32 parentHash = Lib_RLPReader.readBytes32(blockNp1[0]);
      //require(blockNumberNHash == parentHash, "parent block hash somehow wrong");

      bytes32 newroot = Lib_RLPReader.readBytes32(blockNp1[3]);
      require(assertionRoot != newroot, "asserting that the real state is correct is not a challenge");

      // load starting info into the input oracle
      // we both agree at the beginning
      bytes32 txhash = Lib_RLPReader.readBytes32(blockNp1[4]);
      bytes32 coinbase = bytes32(uint256(Lib_RLPReader.readAddress(blockNp1[2])));
      bytes32 unclehash = Lib_RLPReader.readBytes32(blockNp1[1]);
      bytes32 gaslimit = bytes32(Lib_RLPReader.readUint256(blockNp1[9]));
      bytes32 time = bytes32(Lib_RLPReader.readUint256(blockNp1[11]));
      inputHash = keccak256(abi.encodePacked(parentHash, txhash, coinbase, unclehash, gaslimit, time));
    }

    bytes32 startState = GlobalStartState;
    startState = mem.WriteBytes32(startState, 0x30000000, inputHash);

    // confirm the finalSystemHash asserts the state you claim and the machine is stopped
    // you must load these trie nodes into MIPSMemory before calling this
    require(mem.ReadMemory(finalSystemState, 0xC0000080) == 0x5EAD0000, "machine is not stopped in final state (PC == 0x5EAD0000)");
    require(mem.ReadMemory(finalSystemState, 0x30000800) == 0x1337f00d, "state is not outputted");
    require(mem.ReadBytes32(finalSystemState, 0x30000804) == assertionRoot, "you are claiming a different state root in machine");

    return newChallengeTrusted(startState, finalSystemState, stepCount);
  }

  // binary search

  function getStepNumber(uint256 challengeId) view public returns (uint256) {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    return (c.L+c.R)/2;
  }

  function getProposedState(uint256 challengeId) view public returns (bytes32) {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    return c.assertedState[stepNumber];
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

  function DenyStateTransition(uint256 challengeId, bytes32 finalRiscState) external {
    Chal storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    require(owner == msg.sender, "must be owner");
    require(c.L + 1 == c.R, "binary search not finished");

    require(finalRiscState != c.assertedState[c.R], "you can't agree with the challenger");

    // it's 0 if you agree with all attacker states except the final one
    // in which case, you get a free pass to submit now
    require(c.defendedState[c.R] == finalRiscState || c.defendedState[c.R] == bytes32(0), "must be consistent with state");
    require(mips.Step(c.defendedState[c.L]) == finalRiscState, "wrong asserted state");

    // consider the challenger mocked
    emit ChallengerLoses(challengeId);
  }
}
