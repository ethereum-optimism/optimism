/* External Imports */
import * as Level from 'level'

import {
  BaseDB,
  DB,
  DefaultSignatureProvider,
  DefaultSignatureVerifier,
  getLogger,
} from '@pigi/core'
import {
  AggregatorServer,
  DefaultRollupStateMachine,
  RollupAggregator,
  RollupStateMachine,
  RollupBlockSubmitter,
  RollupBlock,
  DefaultRollupBlockSubmitter,
  genesisState,
} from '@pigi/wallet'
import { EthereumEventProcessor } from '@pigi/watch-eth'

import cors = require('cors')
import { Contract, Wallet } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

import { config } from 'dotenv'
import { resolve } from 'path'
// Starting from build/src/
config({ path: resolve(__dirname, `../../.env`) })

/* Internal Imports */
import * as RollupChain from '../contracts/RollupChain.json'

const log = getLogger('mock-aggregator')

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'

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

const rollupContractAddress = process.env.ROLLUP_CONTRACT_ADDRESS
const mnemonic = process.env.WALLET_MNEMONIC
const jsonRpcUrl = process.env.JSON_RPC_URL
const transitionsPerBlock: number = parseInt(
  process.env.TRANSITIONS_PER_BLOCK || '10',
  10
)

const mockMode = !rollupContractAddress || !mnemonic || !jsonRpcUrl

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

  let blockSubmitter: RollupBlockSubmitter
  let contract: Contract
  if (mockMode) {
    log.debug(`Using dummy block submitter`)
    blockSubmitter = new DummyBlockSubmitter()
  } else {
    log.debug(
      `Connecting to contract [${rollupContractAddress}] at [${jsonRpcUrl}]`
    )
    contract = new Contract(
      rollupContractAddress,
      RollupChain.interface,
      Wallet.fromMnemonic(mnemonic).connect(new JsonRpcProvider(jsonRpcUrl))
    )
    const blockSubmitterDB: DB = new BaseDB(
      (await Level('build/level/blockSubmitter', levelOptions)) as any,
      256
    )
    blockSubmitter = await DefaultRollupBlockSubmitter.create(
      blockSubmitterDB,
      contract
    )
    log.debug(`Connected`)
  }

  const aggregator = await RollupAggregator.create(
    blockDB,
    rollupStateMachine,
    blockSubmitter,
    new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC)),
    DefaultSignatureVerifier.instance(),
    transitionsPerBlock
  )

  if (mockMode) {
    await aggregator.onSyncCompleted()
  } else {
    const blockProcessorDB: DB = new BaseDB(
      (await Level('build/level/blockProcessor', levelOptions)) as any,
      256
    )
    const processor: EthereumEventProcessor = new EthereumEventProcessor(
      blockProcessorDB
    )
    await processor.subscribe(contract, 'NewRollupBlock', aggregator, true)
  }

  const aggregatorServer = new AggregatorServer(aggregator, host, port, [cors])

  // Just listen for requests!
  aggregatorServer.listen()

  // tslint:disable-next-line
  console.log('Listening on', host + ':' + port)
}

runAggregator()
