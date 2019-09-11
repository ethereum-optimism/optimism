/* External Imports */
import {
  State,
  UNISWAP_ADDRESS,
  AGGREGATOR_ADDRESS,
  MockAggregator,
} from '@pigi/wallet'
import cors = require('cors')
import { ethers } from 'ethers'

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'

/* Set the initial balances/state */
export const genesisState: State = {
  [UNISWAP_ADDRESS]: {
    balances: {
      uni: 1000,
      pigi: 1000,
    },
  },
  [AGGREGATOR_ADDRESS]: {
    balances: {
      uni: 1000000,
      pigi: 1000000,
    },
  },
}

// Create a new aggregator... and then...
const host = 'localhost'
const port = 3000
const aggregator = new MockAggregator(
  genesisState,
  host,
  port,
  AGGREGATOR_MNEMONIC,
  undefined,
  [cors]
)
// Just listen for requests!
aggregator.listen()

// tslint:disable-next-line
console.log('Listening on', host + ':' + port)
