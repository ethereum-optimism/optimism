/* External Imports */
import { ImplicationProofItem } from '@pigi/core'
import * as assert from 'assert'

/* Internal Imports */
import {
  AGGREGATOR_ADDRESS,
  PIGI_TOKEN_TYPE,
  RollupStateSolver,
  SignedStateReceipt,
  State,
  StateReceipt,
  UNI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
} from '../src'

export const ALICE_GENESIS_STATE_INDEX = 0
export const UNISWAP_GENESIS_STATE_INDEX = 1
export const ALICE_ADDRESS = '0xaaaf2795C3013711c240244aFF600aD9e8D9727D'
export const BOB_ADDRESS = '0xbbbCAAe85dfE709a25545E610Dba4082f6D02D73'

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'

export const getGenesisState = (
  aliceAddress: string = ALICE_ADDRESS
): State[] => {
  return [
    {
      pubKey: aliceAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 50,
        [PIGI_TOKEN_TYPE]: 50,
      },
    },
    {
      pubKey: UNISWAP_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 50,
        [PIGI_TOKEN_TYPE]: 50,
      },
    },
    {
      pubKey: AGGREGATOR_ADDRESS,
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
      pubKey: aliceAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 5_000,
        [PIGI_TOKEN_TYPE]: 5_000,
      },
    },
    {
      pubKey: UNISWAP_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 650_000,
        [PIGI_TOKEN_TYPE]: 650_000,
      },
    },
    {
      pubKey: AGGREGATOR_ADDRESS,
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

export const assertThrows = (func: () => any, errorType: any): void => {
  let succeeded = true
  try {
    func()
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

export const assertThrowsAsync = async (
  func: () => Promise<any>,
  errorType: any
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
