// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;
pragma experimental ABIEncoderV2;

import "./lib/Lib_RLPReader.sol";

interface IMIPS {
  // Given a MIPS state hash (includes code & registers), execute the next instruction and returns
  // the update state hash.
  function Step(bytes32 stateHash) external returns (bytes32);

  // Returns the associated MIPS memory contract.
  function m() external pure returns (IMIPSMemory);
}

interface IMIPSMemory {
  // Adds a `(hash(anything) => anything)` entry to the mapping that underpins all the Merkle tries
  // that this contract deals with (where "state hash" = Merkle root of such a trie).
  // Here, `anything` is supposed to be node data in such a trie.
  function AddTrieNode(bytes calldata anything) external;

  function ReadMemory(bytes32 stateHash, uint32 addr) external view returns (uint32);
  function ReadBytes32(bytes32 stateHash, uint32 addr) external view returns (bytes32);

  // Write 32 bits at the given address and returns the updated state hash.
  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 val) external returns (bytes32);

  // Write 32 bytes at the given address and returns the updated state hash.
  function WriteBytes32(bytes32 stateHash, uint32 addr, bytes32 val) external returns (bytes32);
}

contract Challenge {
  address payable immutable owner;

  IMIPS immutable mips;
  IMIPSMemory immutable mem;

  // State hash of the fault proof program's initial MIPS state.
  bytes32 immutable GlobalStartState;

  constructor(IMIPS imips, bytes32 globalStartState) {
    owner = msg.sender;
    mips = imips;
    mem = imips.m();
    GlobalStartState = globalStartState;
  }

  struct ChallengeData {
    // Left bound of the binary search: challenger & defender agree on all steps <= L.
    uint256 L;
    // Right bound of the binary search: challenger & defender disagree on all steps >= R.
    uint256 R;
    // Maps step numbers to asserted state hashes for the challenger.
    mapping(uint256 => bytes32) assertedState;
    // Maps step numbers to asserted state hashes for the defender.
    mapping(uint256 => bytes32) defendedState;
    // Address of the challenger.
    address payable challenger;
    // Block number preceding the challenged block.
    uint256 blockNumberN;
  }

  mapping(uint256 => ChallengeData) challenges;

  // Allow sending money to the contract (without calldata).
  receive() external payable {}

  // Allows the owner to withdraw funds from the contract.
  function withdraw() external {
    require(msg.sender == owner, "not owner");
    (bool sent, ) = owner.call{value: address(this).balance}("");
    require(sent, "Failed to send Ether");
  }

  // ID if the last created challenged, incremented for new challenge IDs.
  uint256 public lastChallengeId = 0;

  // Emitted when a new challenge is created.
  event ChallengeCreated(uint256 challengeId);

  // helper function to determine what nodes we need
  function CallWithTrieNodes(address target, bytes calldata dat, bytes[] calldata nodes) public {
    for (uint i = 0; i < nodes.length; i++) {
      mem.AddTrieNode(nodes[i]);
    }
    (bool success, bytes memory revertData) = target.call(dat);
    // TODO: better way to revert?
    if (!success) {
      uint256 revertDataLength = revertData.length;
      assembly {
          let revertDataStart := add(revertData, 32)
          revert(revertDataStart, revertDataLength)
      }
    }
  }

  /// @notice Challenges the transition from block `blockNumberN` to the next block (N+1), which is
  ///         the block being challenged.
  ///         Before calling this, it is necessary to have loaded all the trie node necessary to
  ///         write the input hash in the Merkleized initial MIPS state, and to read the output hash
  ///         and machine state from the Merkleized final MIPS state (i.e. `finalSystemState`). Use
  ///         `MIPSMemory.AddTrieNode` for this purpose.
  /// @param blockNumberN The number N of the parent of the block being challenged
  /// @param blockHeaderNp1 The RLP-encoded header of the block being challenged (N+1)
  /// @param assertionRoot The state root that the challenger claims is the correct one for the
  ///        given the transactions included in block N+1.
  /// @param finalSystemState The state hash of the fault proof program's final MIPS state.
  /// @param stepCount The number of steps (MIPS instructions) taken to execute the fault proof
  ///        program.
  /// @return The challenge identifier
  function InitiateChallenge(
      uint blockNumberN, bytes calldata blockHeaderNp1, bytes32 assertionRoot,
      bytes32 finalSystemState, uint256 stepCount)
    external
    returns (uint256)
  {
    bytes32 computedBlockHash = keccak256(blockHeaderNp1);

    // get block hashes, can replace with oracle
    bytes32 blockNumberNHash = blockhash(blockNumberN);
    bytes32 blockNumberNp1Hash = blockhash(blockNumberN+1);

    if (blockNumberNHash == bytes32(0) || blockNumberNp1Hash == bytes32(0)) {
      revert("block number too old to challenge");
    }
    require(blockNumberNp1Hash == computedBlockHash, "incorrect header supplied for block N+1");

    // Decode the N+1 block header to construct the fault proof program's input hash.
    // Because the input hash is constructed from data proven against on-chain block hashes,
    // it is provably correct, and we can consider that both parties agree on it.
    bytes32 inputHash;
    {
      Lib_RLPReader.RLPItem[] memory decodedHeader = Lib_RLPReader.readList(blockHeaderNp1);

      bytes32 parentHash = Lib_RLPReader.readBytes32(decodedHeader[0]);
      // This should never happen, as we validated the hashes beforehand.
      require(blockNumberNHash == parentHash, "parent block hash somehow wrong");

      bytes32 newroot = Lib_RLPReader.readBytes32(decodedHeader[3]);
      require(assertionRoot != newroot,
          "asserting that the real state is correct is not a challenge");

      bytes32 txhash    = Lib_RLPReader.readBytes32(decodedHeader[4]);
      bytes32 coinbase  = bytes32(uint256(uint160(Lib_RLPReader.readAddress(decodedHeader[2]))));
      bytes32 unclehash = Lib_RLPReader.readBytes32(decodedHeader[1]);
      bytes32 gaslimit  = Lib_RLPReader.readBytes32(decodedHeader[9]);
      bytes32 time      = Lib_RLPReader.readBytes32(decodedHeader[11]);

      inputHash = keccak256(abi.encodePacked(parentHash, txhash, coinbase, unclehash, gaslimit, time));
    }

    // Write input hash at predefined memory address.
    bytes32 startState = GlobalStartState;
    startState = mem.WriteBytes32(startState, 0x30000000, inputHash);

    // Confirm that `finalSystemState` asserts the state you claim and that the machine is stopped.
    require(mem.ReadMemory(finalSystemState, 0xC0000080) == 0x5EAD0000,
        "the final MIPS machine state is not stopped (PC != 0x5EAD0000)");
    require(mem.ReadMemory(finalSystemState, 0x30000800) == 0x1337f00d,
        "the final state root has not been written a the predefined MIPS memory location");
    require(mem.ReadBytes32(finalSystemState, 0x30000804) == assertionRoot,
        "the final MIPS machine state asserts a different state root than your challenge");

    uint256 challengeId = lastChallengeId++;
    ChallengeData storage c = challenges[challengeId];

    // A NEW CHALLENGER APPEARS
    c.challenger = msg.sender;
    c.blockNumberN = blockNumberN;
    c.assertedState[0] = startState;
    c.defendedState[0] = startState;
    c.assertedState[stepCount] = finalSystemState;
    c.L = 0;
    c.R = stepCount;

    emit ChallengeCreated(challengeId);
    return challengeId;
  }

  // binary search

  function isSearching(uint256 challengeId) view public returns (bool) {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    return c.L + 1 != c.R;
  }

  function getStepNumber(uint256 challengeId) view public returns (uint256) {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    return (c.L+c.R)/2;
  }

  function getProposedState(uint256 challengeId) view public returns (bytes32) {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    uint256 stepNumber = getStepNumber(challengeId);
    return c.assertedState[stepNumber];
  }

  function ProposeState(uint256 challengeId, bytes32 riscState) external {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    require(c.challenger == msg.sender, "must be challenger");
    require(isSearching(challengeId), "must be searching");

    uint256 stepNumber = getStepNumber(challengeId);
    require(c.assertedState[stepNumber] == bytes32(0), "state already proposed");
    c.assertedState[stepNumber] = riscState;
  }

  function RespondState(uint256 challengeId, bytes32 riscState) external {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    require(owner == msg.sender, "must be owner");
    require(isSearching(challengeId), "must be searching");

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
  // anyone can call these, right?

  event ChallengerWins(uint256 challengeId);
  event ChallengerLoses(uint256 challengeId);
  event ChallengerLosesByDefault(uint256 challengeId);

  function ConfirmStateTransition(uint256 challengeId) external {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    //require(c.challenger == msg.sender, "must be challenger");
    require(!isSearching(challengeId), "binary search not finished");
    bytes32 stepState = mips.Step(c.assertedState[c.L]);

    require(stepState == c.assertedState[c.R], "wrong asserted state for challenger");

    // pay out bounty!!
    (bool sent, ) = c.challenger.call{value: address(this).balance}("");
    require(sent, "Failed to send Ether");

    emit ChallengerWins(challengeId);
  }

  function DenyStateTransition(uint256 challengeId) external {
    ChallengeData storage c = challenges[challengeId];
    require(c.challenger != address(0), "invalid challenge");
    //require(owner == msg.sender, "must be owner");
    require(!isSearching(challengeId), "binary search not finished");
    bytes32 stepState = mips.Step(c.defendedState[c.L]);

    // NOTE: challenger can make c.defendedState[c.R] 0 if the search always went right
    // while the challenger can't win, you can't make them lose
    if (c.defendedState[c.R] == bytes32(0)) {
      emit ChallengerLosesByDefault(challengeId);
      return;
    }

    require(stepState == c.defendedState[c.R], "wrong asserted state for defender");

    // consider the challenger mocked
    emit ChallengerLoses(challengeId);
  }
}
