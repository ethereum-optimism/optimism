/* External Imports */
import * as AsyncLock from 'async-lock'

import {
  IdentityVerifier,
  serializeObject,
  SignatureVerifier,
  DB,
  SparseMerkleTree,
  SparseMerkleTreeImpl,
  BigNumber,
  ONE,
  runInDomain,
  MerkleTreeInclusionProof,
  ZERO,
  getLogger,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  Balances,
  Swap,
  isSwapTransaction,
  Transfer,
  isTransferTransaction,
  RollupTransaction,
  SignedTransaction,
  UNISWAP_ADDRESS,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  TokenType,
  State,
  StateUpdate,
  StateSnapshot,
  InclusionProof,
  StateMachineCapacityError,
  SignatureError,
  AGGREGATOR_ADDRESS,
  abiEncodeTransaction,
  abiEncodeState,
  parseStateFromABI,
  DefaultRollupStateMachine,
} from './index'

import {
  RollupBlock,
  RollupStateGuard,
  RollupTransitionPosition,
  FraudCheckResult,
  RollupStateMachine,
} from './types'

const log = getLogger('rollup-aggregator')
export class DefaultRollupStateGuard implements RollupStateGuard {
  public rollupMachine: RollupStateMachine
  public currentPosition: RollupTransitionPosition = {
    blockNumber: 0,
    transitionIndex: 0,
  }

  public static async create(
    genesisState: State[],
    stateMachineDb: DB
  ): Promise<DefaultRollupStateGuard> {
    const theRollupMachine = await DefaultRollupStateMachine.create(
      genesisState,
      stateMachineDb,
      IdentityVerifier.instance()
    )
    return new DefaultRollupStateGuard(theRollupMachine)
  }

  constructor(theRollupMachine: RollupStateMachine) {
    this.rollupMachine = theRollupMachine
  }

  public async getCurrentVerifiedPosition(): Promise<RollupTransitionPosition> {
    return this.currentPosition
  }

  public async checkNextTransition(
    nextSignedTransaction: SignedTransaction,
    nextRolledUpRoot: string
  ): Promise<FraudCheckResult> {
    return 'NO_FRAUD'
  }

  public async checkNextBlock(
    nextBlock: RollupBlock
  ): Promise<FraudCheckResult> {
    return 'NO_FRAUD'
  }
}
