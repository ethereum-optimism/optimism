/* External Imports */
import MemDown from 'memdown'
import { BaseDB } from '@pigi/core'
import {
  State,
  UNISWAP_ADDRESS,
  AGGREGATOR_ADDRESS,
  RollupAggregator,
  RollupStateMachine,
  DefaultRollupStateMachine,
} from '@pigi/wallet'
import cors = require('cors')

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

async function runAggregator() {
  const stateDB = new BaseDB(new MemDown('state') as any)
  const blockDB = new BaseDB(new MemDown('blocks') as any)

  const rollupStateMachine: RollupStateMachine = await DefaultRollupStateMachine.create(
    genesisState,
    stateDB
  )

  const aggregator = new RollupAggregator(
    blockDB,
    rollupStateMachine,
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
}

runAggregator()
