/* External Imports */
import * as Level from 'level'

import { BaseDB, DefaultSignatureProvider, newInMemoryDB } from '@pigi/core'
import {
  AggregatorServer,
  DefaultRollupStateMachine,
  AGGREGATOR_ADDRESS,
  PIGI_TOKEN_TYPE,
  UNI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
  RollupAggregator,
  RollupStateMachine,
  State,
  RollupBlockSubmitter,
  RollupBlock,
} from '@pigi/wallet'
import cors = require('cors')
import { Wallet } from 'ethers'

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'

/* Set the initial balances/state */
export const genesisState: State[] = [
  {
    pubKey: UNISWAP_ADDRESS,
    balances: {
      [UNI_TOKEN_TYPE]: 1000,
      [PIGI_TOKEN_TYPE]: 1000,
    },
  },
  {
    pubKey: AGGREGATOR_ADDRESS,
    balances: {
      [UNI_TOKEN_TYPE]: 1000000,
      [PIGI_TOKEN_TYPE]: 1000000,
    },
  },
]

class DummyBlockSubmitter implements RollupBlockSubmitter {
  public async handleNewRollupBlock(rollupBlockNumber: number): Promise<void> {
    // no-op
  }

  public async submitBlock(block: RollupBlock): Promise<void> {
    // no-op
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

// Create a new aggregator... and then...
const host = '0.0.0.0'
const port = 3000

async function runAggregator() {
  const levelOptions = {
    keyEncoding: 'binary',
    valueEncoding: 'binary',
  }
  const stateDB = new BaseDB((await Level(
    'build/level/state',
    levelOptions
  )) as any)
  const blockDB = new BaseDB(
    (await Level('build/level/blocks', levelOptions)) as any,
    4
  )

  const rollupStateMachine: RollupStateMachine = await DefaultRollupStateMachine.create(
    genesisState,
    stateDB
  )

  // TODO: Actually populate this.
  const blockSubmitter: RollupBlockSubmitter = new DummyBlockSubmitter()

  const aggregator = await RollupAggregator.create(
    blockDB,
    rollupStateMachine,
    blockSubmitter,
    new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC))
  )

  // TODO: sync blocks and remove this line
  await aggregator.onSyncCompleted()

  const aggregatorServer = new AggregatorServer(aggregator, host, port, [cors])

  // Just listen for requests!
  aggregatorServer.listen()

  // tslint:disable-next-line
  console.log('Listening on', host + ':' + port)
}

runAggregator()
