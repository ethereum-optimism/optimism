/* External Imports */
import * as Level from 'level'
import { Contract, Wallet } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

import { BaseDB, DB, EthereumEventProcessor } from '@pigi/core-db'
import { getLogger } from '@pigi/core-utils'

import { config } from 'dotenv'
import { resolve } from 'path'

// Starting from build/src/validator/
config({ path: resolve(__dirname, `../../../../config/.env`) })

/* Internal Imports */
import * as RollupChain from '../../../build/contracts/RollupChain.json'
import * as fs from 'fs'
import { Address, RollupStateValidator, State } from '../../types'
import { DefaultRollupStateMachine, getGenesisState } from '../../common'
import { DefaultRollupStateValidator } from '../rollup-state-validator'
import { RollupFraudGuard } from '../rollup-fraud-guard'

const log = getLogger('monitor-and-validate')

const aggregatorAddress: Address = process.env.AGGREGATOR_ADDRESS
const rollupContractAddress: Address = process.env.ROLLUP_CONTRACT_ADDRESS
const mnemonic: string = process.env.VALIDATOR_MNEMONIC
const jsonRpcUrl: string = process.env.JSON_RPC_URL
const genesisStateFilePath: string =
  process.env.GENESIS_STATE_RELATIVE_FILE_PATH
let genesisState: State[]
if (!!genesisStateFilePath) {
  genesisState = JSON.parse(
    fs.readFileSync(resolve(__dirname, genesisStateFilePath)).toString('utf-8')
  )
  log.info(
    `Loaded genesis state from ${genesisStateFilePath}. ${genesisState.length} balances loaded.`
  )
} else {
  log.info('No genesis state provided!')
}

const waitForever = (): Promise<void> => {
  return new Promise(() => {
    // Don't ever resolve
  })
}

async function runValidator() {
  const levelOptions = {
    keyEncoding: 'binary',
    valueEncoding: 'binary',
  }
  const validatorDB = new BaseDB((await Level(
    'build/level/validator',
    levelOptions
  )) as any)

  log.debug(
    `Connecting to contract [${rollupContractAddress}] at [${jsonRpcUrl}]`
  )
  const contract: Contract = new Contract(
    rollupContractAddress,
    RollupChain.interface as any,
    Wallet.fromMnemonic(mnemonic).connect(new JsonRpcProvider(jsonRpcUrl))
  )
  log.debug(`Connected`)

  const rollupStateMachine: DefaultRollupStateMachine = (await DefaultRollupStateMachine.create(
    getGenesisState(aggregatorAddress, genesisState),
    validatorDB,
    aggregatorAddress
  )) as DefaultRollupStateMachine

  const validator: RollupStateValidator = new DefaultRollupStateValidator(
    rollupStateMachine
  )

  const fraudGuard: RollupFraudGuard = await RollupFraudGuard.create(
    validatorDB,
    validator,
    contract
  )

  const blockProcessorDB: DB = new BaseDB(
    (await Level('build/level/validator-blockProcessor', levelOptions)) as any,
    256
  )
  const processor: EthereumEventProcessor = new EthereumEventProcessor(
    blockProcessorDB
  )

  await processor.subscribe(contract, 'NewRollupBlock', fraudGuard, true)

  log.info(`Started. Waiting forever.`)
  await waitForever()
}

runValidator()
