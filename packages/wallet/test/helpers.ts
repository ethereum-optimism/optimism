/***********
 * HELPERS *
 ***********/

import { State, UNISWAP_ADDRESS, AGGREGATOR_ADDRESS } from '../src'

export const getGenesisState = (): State => {
  return {
    [UNISWAP_ADDRESS]: {
      balances: {
        uni: 50,
        pigi: 50,
      },
    },
    alice: {
      balances: {
        uni: 50,
        pigi: 50,
      },
    },
    [AGGREGATOR_ADDRESS]: {
      balances: {
        uni: 1000000,
        pigi: 1000000,
      },
    },
  }
}

export const genesisState: State = {
  [UNISWAP_ADDRESS]: {
    balances: {
      uni: 50,
      pigi: 50,
    },
  },
  alice: {
    balances: {
      uni: 50,
      pigi: 50,
    },
  },
  [AGGREGATOR_ADDRESS]: {
    balances: {
      uni: 1000000,
      pigi: 1000000,
    },
  },
}
