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
  bufToHexString,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  Balances,
  Swap,
  Transfer,
  isSwapTransition,
  isCreateAndTransferTransition,
  isTransferTransition,
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
  InsufficientBalanceError,
  NegativeAmountError,
  InvalidTransactionTypeError,
  InvalidTokenTypeError,
  
} from './index'

import {
  RollupBlock,
  RollupStateGuard,
  RollupTransitionPosition,
  FraudCheckResult,
  RollupStateMachine,
  RollupTransition,
  SlippageError,
  LocalMachineError,
  FraudProof,
} from './types'
import { Transaction } from 'ethers/utils'
import { parseTransitionFromABI, parseTransactionFromABI } from './serialization';

const log = getLogger('rollup-guard')
export class DefaultRollupStateGuard implements RollupStateGuard {
  public rollupMachine: DefaultRollupStateMachine
  public currentPosition: RollupTransitionPosition = {
    blockNumber: 0,
    transitionIndex: 0,
  }

  public static async create(
    genesisState: State[],
    stateMachineDb: DB
  ): Promise<DefaultRollupStateGuard> {
    const theRollupMachine = (await DefaultRollupStateMachine.create(
      genesisState,
      stateMachineDb,
      IdentityVerifier.instance()
    )) as DefaultRollupStateMachine
    return new DefaultRollupStateGuard(theRollupMachine)
  }

  constructor(theRollupMachine: DefaultRollupStateMachine) {
    this.rollupMachine = theRollupMachine
  }

  public async getCurrentVerifiedPosition(): Promise<RollupTransitionPosition> {
    return this.currentPosition
  }

  public async getInputStateSnapshots(
    transition: RollupTransition
  ): Promise<StateSnapshot[]> {
    if (isSwapTransition(transition)) {
      const swapperSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.senderSlotIndex
      )
      const uniSnapshot: StateSnapshot = await this.rollupMachine.getState(
        UNISWAP_ADDRESS
      )
      return [swapperSnapshot, uniSnapshot]
    } else if (isCreateAndTransferTransition(transition)) {
        console.log('went here!')
      const nextAccountKey: number = this.rollupMachine.getNextNewAccountSlot()
      const senderSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.senderSlotIndex
      )
      console.log('recip key is' + transition.recipientSlotIndex)
      const recipientSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.recipientSlotIndex
      )
      return [senderSnapshot, recipientSnapshot]
    } else if (isTransferTransition(transition)) {
      const senderSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.senderSlotIndex
      )
      const recipientSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.recipientSlotIndex
      )
      return [senderSnapshot, recipientSnapshot]
    }
  }

  public async getTransactionFromTransition(transition: RollupTransition): Promise<SignedTransaction> {
      return undefined
  }

  public async checkNextEncodedTransition(
    encodedNextTransition: string,
    nextRolledUpRoot: Buffer
  ): Promise<FraudCheckResult> {
    let postRoot: Buffer
    let preppedFraudInputs: StateSnapshot[] = undefined
    
    let nextTransition: RollupTransition = parseTransitionFromABI(encodedNextTransition)

    console.log('parsed transition is: ')
    console.log(nextTransition)

    // In case there was fraud in this transaction, get state snapshots for each input so we can prove the fraud later.
    preppedFraudInputs = await this.getInputStateSnapshots(nextTransition)

    // let inputAsTransaction: SignedTransaction = await this.getTransactionFromTransition(nextTransition)
    let inputAsTransaction: SignedTransaction = await this.getTransactionFromTransition(nextTransition)

    try {
      await this.rollupMachine.applyTransaction(inputAsTransaction)
      postRoot = await this.rollupMachine.getStateRoot()
    } catch (error) {
      if (isStateTransitionError(error)) {
        // return the fraud proof, invalid transaction
        return {
          fraudPosition: this.currentPosition,
          fraudInputs: preppedFraudInputs,
          fraudTransition: nextTransition,
        }
      } else {
        throw new LocalMachineError()
      }
    }
    console.log('got post root:')
    console.log(bufToHexString(postRoot))
    console.log('compared to next root: ')
    console.log(bufToHexString(nextRolledUpRoot))
    if (postRoot.equals(nextRolledUpRoot)) {
      this.currentPosition.blockNumber++
      this.currentPosition.transitionIndex++
      return 'NO_FRAUD'
    } else {
      // return the fraud proof, invalid root
      return {
        fraudPosition: this.currentPosition,
        fraudInputs: preppedFraudInputs,
        fraudTransition: nextTransition,
      }
    }
  }

  public async checkNextBlock(
    nextBlock: RollupBlock
  ): Promise<FraudCheckResult> {
    // TODO: compare nextBlock.number to currentPosition to ensure that this is indeed the sequential block.
    return 'NO_FRAUD'
  }
}

function isStateTransitionError(error: Error) {
  return (
    error instanceof SlippageError ||
    error instanceof InsufficientBalanceError ||
    error instanceof NegativeAmountError ||
    error instanceof InvalidTransactionTypeError ||
    error instanceof StateMachineCapacityError ||
    error instanceof InvalidTokenTypeError ||
    error instanceof SignatureError
  )
}
