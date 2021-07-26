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

export interface GovernorMethods {
  guardian(): Callable<string>;
  propose(targets: string[], values: encodedNumber[], signatures: string[], calldatas: string[], description: string): Sendable<void>
  proposals(proposalId: number): Callable<Proposal>;
  proposalCount(): Callable<number>;
  latestProposalIds(proposer: string): Callable<number>;
  getReceipt(proposalId: number, voter: string): Callable<{ hasVoted: boolean, support: boolean, votes: number }>;
  castVote(proposalId: number, support: boolean): Sendable<void>;
  queue(proposalId: encodedNumber): Sendable<void>;
  execute(proposalId: encodedNumber): Sendable<void>;
  cancel(proposalId: encodedNumber): Sendable<void>;
  setBlockNumber(blockNumber: encodedNumber): Sendable<void>;
  setBlockTimestamp(blockTimestamp: encodedNumber): Sendable<void>;
  state(proposalId: encodedNumber): Callable<number>;
  __queueSetTimelockPendingAdmin(newPendingAdmin: string, eta: encodedNumber): Sendable<void>;
  __executeSetTimelockPendingAdmin(newPendingAdmin: string, eta: encodedNumber): Sendable<void>;
  __acceptAdmin(): Sendable<void>;
  __abdicate(): Sendable<void>;
}

export interface Governor extends Contract {
  methods: GovernorMethods;
  name: string;
}
