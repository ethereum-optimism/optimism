// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

interface IMIPS {
  function Step(bytes32 stateHash) external view returns (bytes32);
}

contract Challenge {
  address payable immutable owner;
  IMIPS immutable mips;

  struct Chal {
    uint256 L;
    uint256 R;
    mapping(uint256 => bytes32) assertedState;
    mapping(uint256 => bytes32) defendedState;
    address payable challenger;
  }

  Chal[] challenges;

  constructor(IMIPS imips) {
    owner = msg.sender;
    mips = imips;
  }

  // allow getting money
  fallback() external payable {}
  receive() external payable {}
  function withdraw() external {
    require(msg.sender == owner);
    owner.transfer(address(this).balance);
  }

  function InitiateChallenge(uint blockNumberN,
        bytes calldata blockHeaderN, bytes calldata blockHeaderNp1,
        bytes32 assertionHash, bytes32 finalSystemHash, uint256 stepCount) external {
    // is this new?
    Chal storage c = challenges[challenges.length];
    c.challenger = msg.sender;


    // TODO: this is the function with the complexity
  }

  function getStepNumber(uint256 challengeId) view public returns (uint256) {
    Chal storage c = challenges[challengeId];
    return (c.L+c.R)/2;
  }

  function ProposeState(uint256 challengeId, bytes32 riscState) external {
    Chal storage c = challenges[challengeId];
    require(c.challenger == msg.sender, "must be challenger");

    uint256 stepNumber = getStepNumber(challengeId);
    require(c.assertedState[stepNumber] == bytes32(0), "state already proposed");
    c.assertedState[stepNumber] = riscState;
  }

  function RespondState(uint256 challengeId, bytes32 riscState) external {
    Chal storage c = challenges[challengeId];
    require(msg.sender == owner, "must be owner");

    uint256 stepNumber = getStepNumber(challengeId);
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

  function ConfirmStateTransition(uint256 challengeId) external {
    Chal storage c = challenges[challengeId];
    require(c.challenger == msg.sender, "must be challenger");

    require(c.L + 1 == c.R, "binary search not finished");
    bytes32 newState = mips.Step(c.assertedState[c.L]);
    require(newState == c.assertedState[c.R], "wrong asserted state");

    // pay out bounty!!
    msg.sender.transfer(address(this).balance);
  }
}
