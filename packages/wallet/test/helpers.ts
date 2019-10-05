/* External Imports */
import { getLogger, ImplicationProofItem, logError } from '@pigi/core'
import * as assert from 'assert'

/* Internal Imports */
import {
  AGGREGATOR_ADDRESS,
  PIGI_TOKEN_TYPE,
  RollupBlock,
  RollupBlockSubmitter,
  RollupStateSolver,
  SignedStateReceipt,
  State,
  StateReceipt,
  UNI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
  UNISWAP_STORAGE_SLOT,
} from '../src'

const log = getLogger('helpers', true)

export const UNISWAP_GENESIS_STATE_INDEX = UNISWAP_STORAGE_SLOT
export const ALICE_GENESIS_STATE_INDEX = 1
export const ALICE_ADDRESS = '0xaaaf2795C3013711c240244aFF600aD9e8D9727D'
export const BOB_ADDRESS = '0xbbbCAAe85dfE709a25545E610Dba4082f6D02D73'

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'

export const getGenesisState = (
  aliceAddress: string = ALICE_ADDRESS
): State[] => {
  return [
    {
      pubkey: UNISWAP_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 50,
        [PIGI_TOKEN_TYPE]: 50,
      },
    },
    {
      pubkey: aliceAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 50,
        [PIGI_TOKEN_TYPE]: 50,
      },
    },
    {
      pubkey: AGGREGATOR_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 1_000_000,
        [PIGI_TOKEN_TYPE]: 1_000_000,
      },
    },
  ]
}

export const getGenesisStateLargeEnoughForFees = (
  aliceAddress: string = ALICE_ADDRESS
): State[] => {
  return [
    {
      pubkey: UNISWAP_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 650_000,
        [PIGI_TOKEN_TYPE]: 650_000,
      },
    },
    {
      pubkey: aliceAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 5_000,
        [PIGI_TOKEN_TYPE]: 5_000,
      },
    },
    {
      pubkey: AGGREGATOR_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 1_000_000,
        [PIGI_TOKEN_TYPE]: 1_000_000,
      },
    },
  ]
}

/**
 * Calculates the expected amount of the other currency in a swap, given the
 * liquidity, trade amount, and fees.
 *
 * @param inputAmount The amount being traded
 * @param inputTokenLiquidity The total amount of the traded token at the exchange
 * @param outputTokenLiquidity The total amount of the received token at the exchange
 * @param feeBasisPoints The exchange fee
 * @returns The expected amount of the received token to receive
 */
export const calculateSwapWithFees = (
  inputAmount: number,
  inputTokenLiquidity: number,
  outputTokenLiquidity: number,
  feeBasisPoints
): number => {
  const exchangeRate = outputTokenLiquidity / inputTokenLiquidity

  const expectedOutputBeforeFees = inputAmount * exchangeRate
  const volumeFeePct =
    expectedOutputBeforeFees / (outputTokenLiquidity + expectedOutputBeforeFees)
  const feePct = volumeFeePct + feeBasisPoints / 10_000

  return Math.floor(
    expectedOutputBeforeFees - expectedOutputBeforeFees * feePct
  )
}

export const assertThrows = (func: () => any, errorType?: any): void => {
  let succeeded = true
  try {
    func()
    succeeded = false
  } catch (e) {
    if (!!errorType && !(e instanceof errorType)) {
      succeeded = false
      logError(log, `Threw wrong error. Expected ${typeof errorType}`, e)
    }
  }

  assert(
    succeeded,
    "Function didn't throw as expected or threw the wrong error."
  )
}

export const assertThrowsAsync = async (
  func: () => Promise<any>,
  errorType?: any
): Promise<void> => {
  let succeeded = true
  try {
    await func()
    succeeded = false
  } catch (e) {
    if (!!errorType && !(e instanceof errorType)) {
      succeeded = false
    }
  }

  assert(
    succeeded,
    "Function didn't throw as expected or threw the wrong error."
  )
}

export class DummyRollupStateSolver implements RollupStateSolver {
  private validityResult: boolean = true
  private fraudProof: ImplicationProofItem[]
  private storeErrorToThrow: Error

  public setFraudProof(fraudProof: ImplicationProofItem[]): void {
    this.fraudProof = fraudProof
  }

  public setValidityResult(validityResult: boolean): void {
    this.validityResult = validityResult
  }

  public setStoreErrorToThrow(storeErrorToThrow: Error): void {
    this.storeErrorToThrow = storeErrorToThrow
  }

  public async getFraudProof(
    stateReceipt: StateReceipt,
    signer: string
  ): Promise<ImplicationProofItem[]> {
    return this.fraudProof
  }

  public async isStateReceiptProvablyValid(
    stateReceipt: StateReceipt,
    signer: string
  ): Promise<boolean> {
    return this.validityResult
  }

  public async storeSignedStateReceipt(
    signedReceipt: SignedStateReceipt
  ): Promise<void> {
    if (!!this.storeErrorToThrow) {
      throw this.storeErrorToThrow
    }
  }
}

export class DummyBlockSubmitter implements RollupBlockSubmitter {
  public submitedBlocks: RollupBlock[] = []

  public async handleNewRollupBlock(rollupBlockNumber: number): Promise<void> {
    // no-op
  }

  public async submitBlock(block: RollupBlock): Promise<void> {
    this.submitedBlocks.push(block)
  }

  public getLastConfirmed(): number {
    return 0
  }

  public getLastQueued(): number {
    return 0
  }

  public getLastSubmitted(): number {
    return 0
  }
}
