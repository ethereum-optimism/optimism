import { Contract } from '../Contract';
import { Callable, Sendable } from '../Invokation';
import { encodedNumber } from '../Encoding';

export interface Proposal {
  id: number
  proposer: string
  eta: number
  targets: string[]
  values: number[]
  signatures: string[]
  calldatas: string[]
  startBlock: number
  endBlock: number
  forVotes: number
  againstVotes: number
  abstainVotes: number
}

export const proposalStateEnums = {
  0: "Pending",
  1: "Active",
  2: "Canceled",
  3: "Defeated",
  4: "Succeeded",
  5: "Queued",
  6: "Expired",
  7: "Executed"
}

export interface GovernorBravoMethods {
  admin(): Callable<string>;
  pendingAdmin(): Callable<string>;
  implementation(): Callable<string>;
  propose(targets: string[], values: encodedNumber[], signatures: string[], calldatas: string[], description: string): Sendable<void>
  proposals(proposalId: number): Callable<Proposal>;
  proposalCount(): Callable<number>;
  latestProposalIds(proposer: string): Callable<number>;
  getReceipt(proposalId: number, voter: string): Callable<{ hasVoted: boolean, support: number, votes: number }>;
  castVote(proposalId: number, support: number): Sendable<void>;
  castVoteWithReason(proposalId: number, support: number, reason: string): Sendable<void>;
  queue(proposalId: encodedNumber): Sendable<void>;
  execute(proposalId: encodedNumber): Sendable<void>;
  cancel(proposalId: encodedNumber): Sendable<void>;
  setBlockNumber(blockNumber: encodedNumber): Sendable<void>;
  setBlockTimestamp(blockTimestamp: encodedNumber): Sendable<void>;
  state(proposalId: encodedNumber): Callable<number>;
  proposalThreshold(): Callable<number>;
  votingPeriod(): Callable<number>;
  votingDelay(): Callable<number>;
  _setVotingDelay(newVotingDelay: encodedNumber): Sendable<void>;
  _setVotingPeriod(newVotingPeriod: encodedNumber): Sendable<void>;
  _setProposalThreshold(newProposalThreshold: encodedNumber): Sendable<void>;
  _initiate(governorAlpha: string): Sendable<void>;
  _initiate(): Sendable<void>;
  _setImplementation(address: string): Sendable<void>;
  _setPendingAdmin(address: string): Sendable<void>;
  _acceptAdmin(): Sendable<void>;
}

export interface GovernorBravo extends Contract {
  methods: GovernorBravoMethods;
  name: string;
}
