/* External Imports */
import {
  ChecksumAgnosticIdentityVerifier,
  DB,
  getLogger,
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
  State,
  StateSnapshot,
  DefaultRollupStateMachine,
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
  RollupTransition,
  LocalMachineError,
  LocalFraudProof,
  UniTokenType,
  PigiTokenType,
} from './types'
import { UNISWAP_GENESIS_STATE_INDEX } from '../test/helpers'

const log = getLogger('rollup-validator')
export class DefaultRollupStateValidator implements RollupStateValidator {
  public rollupMachine: DefaultRollupStateMachine
  private currentPosition: RollupTransitionPosition = {
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
    return {...this.currentPosition}
  }

  public async getInputStateSnapshots(
    transition: RollupTransition
  ): Promise<StateSnapshot[]> {
    let firstSlot, secondSlot: number
    if (isSwapTransition(transition)) {
        firstSlot = transition.senderSlotIndex
        secondSlot = UNISWAP_GENESIS_STATE_INDEX
    } else if (isCreateAndTransferTransition(transition)) {
      firstSlot = transition.senderSlotIndex
      secondSlot = transition.recipientSlotIndex
    } else if (isTransferTransition(transition)) {
      firstSlot = transition.senderSlotIndex
      secondSlot = transition.recipientSlotIndex
    }
    return [
      await this.rollupMachine.getSnapshotFromSlot(firstSlot),
      await this.rollupMachine.getSnapshotFromSlot(secondSlot),
    ]
  }

  public async getTransactionFromTransitionAndSnapshots(
    transition: RollupTransition,
    snapshots: StateSnapshot[]
  ): Promise<SignedTransaction> {
    let convertedTx: RollupTransaction
    if (isCreateAndTransferTransition(transition)) {
      const sender: Address = snapshots[0].state.pubKey
      const recipient: Address = transition.createdAccountPubkey as Address
      convertedTx = {
        sender,
        recipient,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        amount: transition.amount,
      }
    } else if (isTransferTransition(transition)) {
      const sender: Address = snapshots[0].state.pubKey
      const recipient: Address = snapshots[1].state.pubKey
      convertedTx = {
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
      convertedTx = {
        sender: swapper,
        tokenType: transition.tokenType as UniTokenType | PigiTokenType,
        inputAmount: transition.inputAmount,
        minOutputAmount: transition.minOutputAmount,
        timeout: transition.timeout,
      }
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
        log.info('Ingested a transaction which does not pass the state machine, must be badly formed!  Returning fraud proof.')
        return {
          fraudPosition: this.currentPosition,
          fraudInputs: preppedFraudInputs,
          fraudTransition: nextTransition,
        }
      } else {
        log.info('Transaction ingestion threw an error--but for a reason unrelated to the transition itself not passing the state machine.  Uh oh!')
        throw new LocalMachineError()
      }
    }

    if (generatedPostRoot.equals(transitionPostRoot)) {
      log.info('Ingested valid transition and postRoot matched the aggregator claim.')
      this.currentPosition.transitionIndex++
      return undefined
    } else {
      log.info('Ingested valid transition and postRoot disagreed with the aggregator claim--returning fraud')
      return {
        fraudPosition: this.currentPosition,
        fraudInputs: preppedFraudInputs,
        fraudTransition: nextTransition,
      }
    }
  }

  public async checkNextBlock(nextBlock: RollupBlock): Promise<any> {
    // reset transition index, we are starting at 0 again!
    this.currentPosition.transitionIndex = 0

    this.ingestedBlocks[nextBlock.blockNumber] = nextBlock

    const nextBlockNumberToValidate: number = (await this.getCurrentVerifiedPosition())
      .blockNumber
    if (nextBlock.blockNumber !== nextBlockNumberToValidate) {
      throw new ValidationOutOfOrderError()
    }

    for (const transition of nextBlock.transitions) {
      const fraudCheck: FraudCheckResult = await this.checkNextTransition(
        transition
      )
      if (!!fraudCheck) {
        // then there was fraud, return the fraud proof to give to contract
        const generatedProof = await this.generateContractFraudProof(
          fraudCheck as LocalFraudProof,
          nextBlock
        )
        return generatedProof
      }
    }
    // otherwise
    this.currentPosition.blockNumber++
    return undefined
  }

  public async generateContractFraudProof(
    localProof: LocalFraudProof,
    block: RollupBlock
  ): Promise<any> {
    const fraudInputs: StateSnapshot[] = localProof.fraudInputs as StateSnapshot[]
    const includedStorageSlots = [
      {
        storageSlot: {
          value: {
            pubkey: fraudInputs[0].state.pubKey,
            balances: [
              fraudInputs[0].state.balances[UNI_TOKEN_TYPE],
              fraudInputs[0].state.balances[PIGI_TOKEN_TYPE],
            ],
          },
          slotIndex: fraudInputs[0].slotIndex,
        },
        siblings: fraudInputs[0].inclusionProof,
      },
      {
        storageSlot: {
          value: {
            pubkey: fraudInputs[1].state.pubKey,
            balances: [
              fraudInputs[1].state.balances[UNI_TOKEN_TYPE],
              fraudInputs[1].state.balances[PIGI_TOKEN_TYPE],
            ],
          },
          slotIndex: fraudInputs[1].slotIndex,
        },
        siblings: fraudInputs[1].inclusionProof,
      },
    ]

    const merklizedBlock: DefaultRollupBlock = new DefaultRollupBlock(
      block.transitions,
      block.blockNumber
    )
    await merklizedBlock.generateTree()

    const curPosition = await this.getCurrentVerifiedPosition()
    const fraudulentTransitionIndex = curPosition.transitionIndex
    let validIncludedTransition
    if (fraudulentTransitionIndex > 0) {
      validIncludedTransition = await merklizedBlock.getIncludedTransition(
        fraudulentTransitionIndex - 1
      )
    } else {
      // then we need to pull from the last block to get preRoot
      const prevRollupBlockNumber: number = curPosition.blockNumber - 1
      const prevRollupBlock: DefaultRollupBlock = new DefaultRollupBlock(
        this.ingestedBlocks[prevRollupBlockNumber].transitions,
        prevRollupBlockNumber
      )
      await prevRollupBlock.generateTree()

      const lastTransitionInLastBlockIndex: number =
        prevRollupBlock.transitions.length - 1
      validIncludedTransition = await prevRollupBlock.getIncludedTransition(
        lastTransitionInLastBlockIndex
      )
    }
    const fraudulentIncludedTransition = await merklizedBlock.getIncludedTransition(
      fraudulentTransitionIndex
    )

    return [
      validIncludedTransition,
      fraudulentIncludedTransition,
      includedStorageSlots,
    ]
  }
}
