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
  RollupTransition,
  LocalMachineError,
  LocalFraudProof,
  UniTokenType,
  PigiTokenType,
} from './types'
import { UNISWAP_GENESIS_STATE_INDEX } from '../test/helpers'
import { type } from 'os'

const log = getLogger('rollup-validator')
export class DefaultRollupStateValidator implements RollupStateValidator {
  public rollupMachine: DefaultRollupStateMachine
  private currentPosition: RollupTransitionPosition = {
    blockNumber: 0,
    transitionIndex: 0,
  }
  private storedBlocks: RollupBlock[] = []

  public static async create(
    genesisState: State[],
    stateMachineDb: DB
  ): Promise<DefaultRollupStateValidator> {
    // The validator spins up a local statte machine
    const theRollupMachine = (await DefaultRollupStateMachine.create(
      genesisState,
      stateMachineDb,
      ChecksumAgnosticIdentityVerifier.instance()
    )) as DefaultRollupStateMachine
    log.info('Successfully spun up a fresh rollupMachine for the validator.')
    return new DefaultRollupStateValidator(theRollupMachine)
  }

  constructor(theRollupMachine: DefaultRollupStateMachine) {
    this.rollupMachine = theRollupMachine
  }

  public async getCurrentVerifiedPosition(): Promise<RollupTransitionPosition> {
    return { ...this.currentPosition }
  }

  public async getInputStateSnapshots(
    transition: RollupTransition
  ): Promise<StateSnapshot[]> {
    let firstSlot: number
    let secondSlot: number
    if (isSwapTransition(transition)) {
      firstSlot = transition.senderSlotIndex
      secondSlot = UNISWAP_GENESIS_STATE_INDEX
      log.info(`Returning snapshots prepper for fraud`)
    } else if (isCreateAndTransferTransition(transition)) {
      firstSlot = transition.senderSlotIndex
      secondSlot = transition.recipientSlotIndex
    } else if (isTransferTransition(transition)) {
      firstSlot = transition.senderSlotIndex
      secondSlot = transition.recipientSlotIndex
    }
    log.info(`Returning snapshots for slots ${firstSlot} and ${secondSlot}.`)
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
  ): Promise<LocalFraudProof> {
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
    log.info(
      `Getting the pre-state inclusion proofs for a ${typeof nextTransition}: ${JSON.stringify(
        nextTransition
      )}`
    )
    preppedFraudInputs = await this.getInputStateSnapshots(nextTransition)
    // convert to transaction so we can apply to validator rollup machine
    log.info(`Converting the transition into a transaction...`)
    const inputAsTransaction: SignedTransaction = await this.getTransactionFromTransitionAndSnapshots(
      nextTransition,
      preppedFraudInputs
    )
    log.info(
      `Got back transaction: ${JSON.stringify(
        inputAsTransaction
      )}.  Attempting to apply it to local state machine.`
    )
    try {
      await this.rollupMachine.applyTransaction(inputAsTransaction)
      generatedPostRoot = await this.rollupMachine.getStateRoot()
    } catch (error) {
      if (isStateTransitionError(error)) {
        log.info(
          'The transaction did not pass the state machine, must be badly formed!  Returning fraud proof.'
        )
        return {
          fraudPosition: this.currentPosition,
          fraudInputs: preppedFraudInputs,
          fraudTransition: nextTransition,
        }
      } else {
        log.error(
          log,
          'Transaction ingestion threw an error--but for a reason unrelated to the transition itself not passing the state machine. Uh oh!',
          error
        )
        throw new LocalMachineError()
      }
    }

    if (generatedPostRoot.equals(transitionPostRoot)) {
      log.info(
        'Ingested valid transition and postRoot matched the aggregator claim.'
      )
      this.currentPosition.transitionIndex++
      return undefined
    } else {
      log.info(
        'Ingested valid transition and postRoot disagreed with the aggregator claim--returning fraud'
      )
      return {
        fraudPosition: this.currentPosition,
        fraudInputs: preppedFraudInputs,
        fraudTransition: nextTransition,
      }
    }
  }

  public async storeBlock(newBlock: RollupBlock): Promise<void> {
    this.storedBlocks[newBlock.blockNumber] = newBlock
    return
  }

  public async validateStoredBlock(blockNumber: number): Promise<any> {
    // grab the block itself from our stored blocks
    const blockToValidate: RollupBlock = this.storedBlocks[blockNumber]
    if (!blockToValidate) {
      log.error(
        'Tried to check next block, but it has not yet been stored yet.'
      )
      throw new ValidationOutOfOrderError()
    }

    log.info(
      'Starting validation for block ' + blockToValidate.blockNumber + '...'
    )
    const nextBlockNumberToValidate: number = (await this.getCurrentVerifiedPosition())
      .blockNumber
    if (blockToValidate.blockNumber !== nextBlockNumberToValidate) {
      throw new ValidationOutOfOrderError()
    }

    // Now loop through and apply the transitions one by one
    for (const transition of blockToValidate.transitions) {
      const fraudCheck: LocalFraudProof = await this.checkNextTransition(
        transition
      )
      if (!!fraudCheck) {
        log.info(
          `Found evidence of fraud at transition ${JSON.stringify(
            transition
          )}.  The current index is ${
            (await this.getCurrentVerifiedPosition()).transitionIndex
          }.  Submitting fraud proof.`
        )
        const generatedProof = await this.generateContractFraudProof(
          fraudCheck as LocalFraudProof,
          blockToValidate
        )
        return generatedProof
      }
    }
    log.info(
      `Found no fraud in block ${nextBlockNumberToValidate}, incrementing block number and reseting transition index`
    )
    // otherwise
    this.currentPosition.blockNumber++
    this.currentPosition.transitionIndex = 0
    return undefined
  }

  public async generateContractFraudProof(
    localProof: LocalFraudProof,
    block: RollupBlock
  ): Promise<any> {
    const fraudInputs: StateSnapshot[] = localProof.fraudInputs as StateSnapshot[]
    log.info(
      `Converting the LocalFraudProof's snapshots into contract-friendly includedStorageSlots...`
    )
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
    log.info(
      `Generating a Merklized block to build inclusions for the fraudulent transition...`
    )
    const merklizedBlock: DefaultRollupBlock = new DefaultRollupBlock(
      block.transitions,
      block.blockNumber
    )
    await merklizedBlock.generateTree()

    const curPosition = await this.getCurrentVerifiedPosition()
    const fraudulentTransitionIndex = curPosition.transitionIndex
    log.info(
      `Fraudlent transition index is ${fraudulentTransitionIndex}.  Getting inclusion proof in rollup block...`
    )
    const fraudulentIncludedTransition = await merklizedBlock.getIncludedTransition(
      fraudulentTransitionIndex
    )
    let validIncludedTransition
    if (fraudulentTransitionIndex > 0) {
      log.info(
        `Since the fraud transition index is > 0, we serve the valid pre-transition from the same block at the previous position.`
      )
      validIncludedTransition = await merklizedBlock.getIncludedTransition(
        fraudulentTransitionIndex - 1
      )
    } else {
      log.info(
        `Since the fraud transition index is  0, we need to grab the valid pre-transition as the last one in the previous block.`
      )
      const prevRollupBlockNumber: number = curPosition.blockNumber - 1
      const prevRollupBlock: DefaultRollupBlock = new DefaultRollupBlock(
        this.storedBlocks[prevRollupBlockNumber].transitions,
        prevRollupBlockNumber
      )
      log.info(
        `Generating a Merklized block to build transition inclusion from the previous block...`
      )
      await prevRollupBlock.generateTree()

      const lastTransitionInLastBlockIndex: number =
        prevRollupBlock.transitions.length - 1
      log.info(
        `Grabbing the last transition from the previous block, it has index ${lastTransitionInLastBlockIndex}`
      )
      validIncludedTransition = await prevRollupBlock.getIncludedTransition(
        lastTransitionInLastBlockIndex
      )
    }
    // TODO: submit to L1, these would be the arguments fed directly into the contract
    return [
      validIncludedTransition,
      fraudulentIncludedTransition,
      includedStorageSlots,
    ]
  }
}
