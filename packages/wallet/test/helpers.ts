/***********
 * HELPERS *
 ***********/

import { AGGREGATOR_ADDRESS, State, UNISWAP_ADDRESS } from '../src'
import * as assert from 'assert'

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'

export const getGenesisState = (aliceAddress: string = 'alice'): State => {
  return {
    [UNISWAP_ADDRESS]: {
      balances: {
        uni: 50,
        pigi: 50,
      },
    },
    [aliceAddress]: {
      balances: {
        uni: 50,
        pigi: 50,
      },
    },
    [AGGREGATOR_ADDRESS]: {
      balances: {
        uni: 1_000_000,
        pigi: 1_000_000,
      },
    },
  }
}

export const getGenesisStateLargeEnoughForFees = (): State => {
  return {
    [UNISWAP_ADDRESS]: {
      balances: {
        uni: 650_000,
        pigi: 500_000,
      },
    },
    alice: {
      balances: {
        uni: 5_000,
        pigi: 5_000,
      },
    },
    [AGGREGATOR_ADDRESS]: {
      balances: {
        uni: 1_000_000,
        pigi: 1_000_000,
      },
    },
  }
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
