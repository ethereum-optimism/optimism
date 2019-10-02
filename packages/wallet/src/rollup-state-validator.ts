/* External Imports */
import * as AsyncLock from 'async-lock'

import {
  ChecksumAgnosticIdentityVerifier,
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
  AggregatorUnsupportedError,
  DefaultRollupBlock,
} from './index'

import {
  RollupBlock,
  RollupStateValidator,
  RollupTransitionPosition,
  FraudCheckResult,
  RollupStateMachine,
  RollupTransition,
  SlippageError,
  LocalMachineError,
  LocalFraudProof,
  UniTokenType,
  PigiTokenType,
} from './types'
import { RollupAggregator } from './aggregator'

const log = getLogger('rollup-guard')
export class DefaultRollupStateValidator implements RollupStateValidator {
  public rollupMachine: DefaultRollupStateMachine
  public currentPosition: RollupTransitionPosition = {
    blockNumber: 0,
    transitionIndex: 0,
  }
  private ingestedBlocks: RollupBlock[] = []

  public static async create(
    genesisState: State[],
    stateMachineDb: DB
  ): Promise<DefaultRollupStateValidator> {
    const theRollupMachine = (await DefaultRollupStateMachine.create(
      genesisState,
      stateMachineDb,
      ChecksumAgnosticIdentityVerifier.instance()
    )) as DefaultRollupStateMachine
    return new DefaultRollupStateValidator(theRollupMachine)
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
      const nextAccountKey: number = this.rollupMachine.getNextNewAccountSlot()
      const senderSnapshot: StateSnapshot = await this.rollupMachine.getSnapshotFromSlot(
        transition.senderSlotIndex
      )
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

  public async getTransactionFromTransitionAndSnapshots(
    transition: RollupTransition,
    snapshots: StateSnapshot[]
  ): Promise<SignedTransaction> {
    let convertedTx: RollupTransaction
    if (isCreateAndTransferTransition(transition)) {
      const sender: Address = snapshots[0].state.pubKey
      const recipient: Address = transition.createdAccountPubkey
      convertedTx = {
        sender,
        recipient,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        amount: transition.amount,
      } as Transfer
    } else if (isTransferTransition(transition)) {
      const sender: Address = snapshots[0].state.pubKey
      const recipient: Address = snapshots[1].state.pubKey
      convertedTx = {
        sender,
        recipient,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        amount: transition.amount,
      } as Transfer
      return {
        signature: transition.signature,
        transaction: convertedTx,
      }
    } else if (isSwapTransition(transition)) {
      const swapper: Address = snapshots[0].state.pubKey
        convertedTx = {
        sender: swapper,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        inputAmount: transition.inputAmount,
        minOutputAmount: transition.minOutputAmount,
        timeout: transition.timeout,
      } as Swap
    }

    return {
      signature: transition.signature,
      transaction: convertedTx,
    }  
  }

  public async checkNextTransition(
    nextTransition: RollupTransition
  ): Promise<FraudCheckResult> {
    let preppedFraudInputs: StateSnapshot[]
    let generatedPostRoot: Buffer

    const transitionPostRoot: Buffer = hexStrToBuf(nextTransition.stateRoot)

    if (isCreateAndTransferTransition(nextTransition)) {
      const slotIfSequential: number = await this.rollupMachine.getNextNewAccountSlot()

      // if the created slot is not sequential, for now it will break
      if (slotIfSequential !== nextTransition.recipientSlotIndex) {
        throw new AggregatorUnsupportedError()
      }
    }

    // In case there was fraud in this transaction, get state snapshots for each input so we can prove the fraud later.
    preppedFraudInputs = await this.getInputStateSnapshots(nextTransition)
    // let inputAsTransaction: SignedTransaction = await this.getTransactionFromTransition(nextTransition)
    const inputAsTransaction: SignedTransaction = await this.getTransactionFromTransitionAndSnapshots(
      nextTransition,
      preppedFraudInputs
    )
    try {
      await this.rollupMachine.applyTransaction(inputAsTransaction)
      generatedPostRoot = await this.rollupMachine.getStateRoot()
    } catch (error) {
      if (isStateTransitionError(error)) {
        // return the fraud proof, invalid transaction
        return {
          fraudPosition: this.currentPosition,
          fraudInputs: preppedFraudInputs,
          fraudTransition: nextTransition,
        }
      } else {
        throw error
        throw new LocalMachineError()
      }
    }

    if (generatedPostRoot.equals(transitionPostRoot)) {
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
  ): Promise<any> {
    // reset transition index, we are starting at 0 again!
    this.currentPosition.transitionIndex = 0

    this.ingestedBlocks[nextBlock.blockNumber] = nextBlock

    const nextBlockNumberToValidate: number = (await this.getCurrentVerifiedPosition()).blockNumber
    console.log('nextBlockNumberToValidate: ', nextBlockNumberToValidate)
    console.log('nextBlock.blockNumber: ', nextBlock.blockNumber)
    if (nextBlock.blockNumber !== nextBlockNumberToValidate) {
      throw new ValidationOutOfOrderError()
    }

    for (const transition of nextBlock.transitions) {
      const fraudCheck: FraudCheckResult = await this.checkNextTransition(
        transition
      )
      if (fraudCheck !== 'NO_FRAUD') {
        // then there was fraud, return the fraud proof to give to contract
        const generatedProof = await this.generateContractFraudProof(fraudCheck as LocalFraudProof, nextBlock)
        return generatedProof
      }
    }
    // otherwise
    this.currentPosition.blockNumber++
    return 'NO_FRAUD'
  }

  public async generateContractFraudProof(localProof: LocalFraudProof, block: RollupBlock): Promise<any> {
    const fraudInputs: StateSnapshot[] = localProof.fraudInputs as StateSnapshot[]
    const includedStorageSlots = [
      {
        storageSlot: {
          value: {
            pubkey: fraudInputs[0].state.pubKey,
            balances: [fraudInputs[0].state.balances[UNI_TOKEN_TYPE], fraudInputs[0].state.balances[PIGI_TOKEN_TYPE]]
          },
          slotIndex: fraudInputs[0].slotIndex
        },
        siblings: fraudInputs[0].inclusionProof
      },
      {
        storageSlot: {
          value: {
            pubkey: fraudInputs[1].state.pubKey,
            balances: [fraudInputs[1].state.balances[UNI_TOKEN_TYPE], fraudInputs[1].state.balances[PIGI_TOKEN_TYPE]]
          },
          slotIndex: fraudInputs[1].slotIndex
        },
        siblings: fraudInputs[1].inclusionProof
      },
    ]

    const merklizedBlock: DefaultRollupBlock = new DefaultRollupBlock(block.transitions, block.blockNumber)
    await merklizedBlock.generateTree()
    
    const curPosition = await this.getCurrentVerifiedPosition()
    const fraudulentTransitionIndex = curPosition.transitionIndex
    let validIncludedTransition
    console.log('found fraud at: ', curPosition)
    if (fraudulentTransitionIndex > 0) {
       validIncludedTransition = await merklizedBlock.getIncludedTransition(fraudulentTransitionIndex - 1)
    } else {
      console.log('got here with ingestedBlocks of', this.ingestedBlocks)
      // then we need to pull from the last block to get preRoot
      const prevRollupBlockNumber: number = curPosition.blockNumber - 1
      const prevRollupBlock: DefaultRollupBlock = new DefaultRollupBlock(
        this.ingestedBlocks[prevRollupBlockNumber].transitions,
        prevRollupBlockNumber
      )
      await prevRollupBlock.generateTree()

      const lastTransitionInLastBlockIndex: number = prevRollupBlock.transitions.length - 1
      validIncludedTransition = await prevRollupBlock.getIncludedTransition(lastTransitionInLastBlockIndex)
    }
    const fraudulentIncludedTransition = await merklizedBlock.getIncludedTransition(fraudulentTransitionIndex)

    return [validIncludedTransition, fraudulentIncludedTransition, includedStorageSlots]
  }
}
