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
  hexStrToBuf,
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
  isStateTransitionError,
  ValidationOutOfOrderError,
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
  UniTokenType,
  PigiTokenType,
} from './types'
import { Transaction } from 'ethers/utils'
import {
  parseTransitionFromABI,
  parseTransactionFromABI,
} from './serialization'
import { assertThrows } from 'test/helpers'

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
      console.log('getting input snapshots for a non creation transfer')
      const senderSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.senderSlotIndex
      )
      console.log('got sender snaphsot')
      const recipientSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.recipientSlotIndex
      )
      return [senderSnapshot, recipientSnapshot]
    }
  }

  public async getTransactionFromTransitionAndSnapshots(
    transition: RollupTransition,
    snapshots: StateSnapshot[]
  ): Promise<SignedTransaction> {
    if (isTransferTransition(transition)) {
      const sender: Address = snapshots[0].state.pubKey
      const recipient: Address = snapshots[1].state.pubKey
      console.log(
        'parsed transition sender and recipient as: ',
        sender,
        recipient
      )
      console.log('parsed transition signature as: ', transition.signature)
      const convertedTx: Transfer = {
        sender,
        recipient,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        amount: transition.amount,
      }
      return {
        signature: transition.signature,
        transaction: convertedTx,
      }
    } else if (isSwapTransition(transition)) {
      const swapper: Address = snapshots[0].state.pubKey
      console.log('parsed transition swapper as: ', swapper)
      console.log('parsed transition signature as: ', transition.signature)
      const convertedTx: Swap = {
        sender: swapper,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        inputAmount: transition.inputAmount,
        minOutputAmount: transition.minOutputAmount,
        timeout: transition.timeout,
      }
      return {
        signature: transition.signature,
        transaction: convertedTx,
      }
    }

    return undefined
  }

  public async checkNextTransition(
    nextTransition: RollupTransition
  ): Promise<FraudCheckResult> {
    let preppedFraudInputs: StateSnapshot[]
    let generatedPostRoot: Buffer

    const transitionPostRoot: Buffer = hexStrToBuf(
      nextTransition.stateRoot
    )

    console.log('parsed transition is: ')
    console.log(nextTransition)

    // In case there was fraud in this transaction, get state snapshots for each input so we can prove the fraud later.
    preppedFraudInputs = await this.getInputStateSnapshots(nextTransition)
    console.log('prepped inputs successfully')
    // let inputAsTransaction: SignedTransaction = await this.getTransactionFromTransition(nextTransition)
    const inputAsTransaction: SignedTransaction = await this.getTransactionFromTransitionAndSnapshots(
      nextTransition,
      preppedFraudInputs
    )
    console.log('parsed transition to tx:')
    console.log(inputAsTransaction)
    try {
      await this.rollupMachine.applyTransaction(inputAsTransaction)
      generatedPostRoot = await this.rollupMachine.getStateRoot()
    } catch (error) {
      if (isStateTransitionError(error)) {
        console.log(error)
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
    console.log('got local post root:')
    console.log(bufToHexString(generatedPostRoot))
    console.log('compared to transition postroot: ')
    console.log(bufToHexString(transitionPostRoot))
    if (generatedPostRoot.equals(transitionPostRoot)) {
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
    const currentPosition: RollupTransitionPosition = await this.getCurrentVerifiedPosition()
    console.log('checking not equals of',nextBlock.number, currentPosition.blockNumber + 1 )
    if (nextBlock.number != currentPosition.blockNumber + 1) {
      throw new ValidationOutOfOrderError()
    }

    for (let transition of nextBlock.transitions) {
      const fraudCheck: FraudCheckResult = await this.checkNextTransition(transition)
      if (fraudCheck !='NO_FRAUD') {
        // then there was fraud, return the fraud check
        return fraudCheck
      }
    }
    // otherwise
    this.currentPosition.blockNumber++
    return 'NO_FRAUD'
  }
}
