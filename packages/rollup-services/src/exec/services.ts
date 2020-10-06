/* External Imports */
import { getLogger, logError, remove0x } from '@eth-optimism/core-utils'
import {
  BaseDB,
  DB,
  DefaultSequentialProcessingDataService,
  EthereumBlockProcessor,
  getLevelInstance,
  PostgresDB,
  RDB,
  SequentialProcessingDataService,
} from '@eth-optimism/core-db'
import {
  DeployResult,
  getContractDefinition,
  deployContracts,
} from '@eth-optimism/rollup-contracts'
import {
  CalldataTxEnqueuedLogHandler,
  CanonicalChainBatchCreator,
  CanonicalChainBatchSubmitter,
  DataService,
  DefaultDataService,
  DefaultL2NodeService,
  Environment,
  FraudDetector,
  GethSubmissionQueuer,
  L1ChainDataPersister,
  L1ToL2BatchAppendedLogHandler,
  L1ToL2TxEnqueuedLogHandler,
  L2ChainDataPersister,
  L2NodeService,
  QueueOrigin,
  QueuedGethSubmitter,
  SafetyQueueBatchAppendedLogHandler,
  SequencerBatchAppendedLogHandler,
  StateBatchAppendedLogHandler,
  StateCommitmentChainBatchCreator,
  StateCommitmentChainBatchSubmitter,
  updateEnvironmentVariables,
  getL1Provider,
  getL2Provider,
  getSequencerWallet,
  getSubmitToL2GethWallet,
  getStateRootSubmissionWallet,
  CanonicalChainBatchFinalizer,
  StateCommitmentChainBatchFinalizer,
} from '@eth-optimism/rollup-core'

import { Contract, ethers } from 'ethers'
import * as fs from 'fs'
import * as rimraf from 'rimraf'

const log = getLogger('service-entrypoint')

/**
 * Runs the configured Rollup services based on configured Environment variables.
 *
 * @returns The services being run.
 */
export const runServices = async (): Promise<any[]> => {
  if (Environment.shouldDeployContracts()) {
    log.info(`Configured to deploy contracts. Deploying contracts...`)
    await deployContractsAndValidateDeployment()
  }

  log.info(`Running services!`)
  const services: any[] = []
  let l1ChainDataPersister: L1ChainDataPersister
  let l2ChainDataPersister: L2ChainDataPersister

  if (Environment.runL1ChainDataPersister()) {
    log.info(`Running L1 Chain Data Persister`)
    l1ChainDataPersister = await createL1ChainDataPersister()
  }
  if (Environment.runL2ChainDataPersister()) {
    log.info(`Running L2 Chain Data Persister`)
    l2ChainDataPersister = await createL2ChainDataPersister()
  }
  if (Environment.runGethSubmissionQueuer()) {
    log.info(`Running Geth Submission Queuer`)
    services.push(await createGethSubmissionQueuer())
  }
  if (Environment.runQueuedGethSubmitter()) {
    log.info(`Running Queued Geth Submitter`)
    services.push(await createQueuedGethSubmitter())
  }
  if (Environment.runCanonicalChainBatchCreator()) {
    log.info(`Running Canonical Chain Batch Creator`)
    services.push(await createCanonicalChainBatchCreator())
  }
  if (Environment.runCanonicalChainBatchSubmitter()) {
    log.info(`Running Canonical Chain Batch Submitter`)
    services.push(await createCanonicalChainBatchSubmitter())
    services.push(await createCanonicalChainBatchFinalizer())
  }
  if (Environment.runStateCommitmentChainBatchCreator()) {
    log.info(`Running State Commitment Chain Batch Creator`)
    services.push(await createStateCommitmentChainBatchCreator())
  }
  if (Environment.runStateCommitmentChainBatchSubmitter()) {
    log.info(`Running State Commitment Chain Batch Submitter`)
    services.push(await createStateCommitmentChainBatchSubmitter())
    services.push(await createStateCommitmentChainBatchFinalizer())
  }
  if (Environment.runFraudDetector()) {
    log.info(`Running Fraud Detector`)
    services.push(await createFraudDetector())
  }

  if (!services.length && !l1ChainDataPersister && !l2ChainDataPersister) {
    log.error(`No services configured! Exiting =|`)
    process.exit(1)
  }

  await Promise.all(
    services.map((x) =>
      x.start().catch((e) => {
        logError(log, `Service threw an error. Exiting.`, e)
        process.exit(1)
      })
    )
  )

  const subscriptions: Array<Promise<any>> = []
  if (!!l1ChainDataPersister) {
    services.push(l1ChainDataPersister)
    const lastProcessedBlock = await l1ChainDataPersister.getLastIndexProcessed()
    const l1Processor: EthereumBlockProcessor = createL1BlockSubscriber(
      lastProcessedBlock + 1
    )
    log.info(`Starting to sync L1 chain`)
    subscriptions.push(
      l1Processor.subscribe(getL1Provider(), l1ChainDataPersister)
    )
  }
  if (!!l2ChainDataPersister) {
    services.push(l2ChainDataPersister)
    const lastProcessedBlock = await l2ChainDataPersister.getLastIndexProcessed()
    const l2Processor: EthereumBlockProcessor = createL2BlockSubscriber(
      lastProcessedBlock + 1
    )
    log.info(`Starting to sync L2 chain`)
    subscriptions.push(
      l2Processor.subscribe(getL2Provider(), l2ChainDataPersister)
    )
  }

  setInterval(() => {
    updateEnvironmentVariables()
  }, 179_000)

  if (!!subscriptions.length) {
    log.debug(`Awaiting chain subscriptions to sync`)
    await Promise.all(subscriptions)
    log.debug(`Awaiting chain subscriptions are synced!`)
  }

  return services
}

/******************************
 * SERVICE CREATION FUNCTIONS *
 ******************************/

/**
 * Creates and returns an L1ChainDataPersister based on configured environment variables.
 *
 * @returns The L1ChainDataPersister.
 */
const createL1ChainDataPersister = async (): Promise<L1ChainDataPersister> => {
  log.info(
    `Creating L1 Chain Data Persister with earliest block ${Environment.l1EarliestBlock()}`
  )
  return L1ChainDataPersister.create(
    getProcessingDataService(),
    getDataService(),
    getL1Provider(),
    [
      {
        topic: ethers.utils.id(
          'L1ToL2TxEnqueued(address,address,uint32,bytes)'
        ), // 7f897cd072f041e68ba57be8f0eec7b8933b0b113622ed8ef85685764f6e7986
        contractAddress: Environment.getOrThrow(
          Environment.l1ToL2TransactionQueueContractAddress
        ),
        handleLog: L1ToL2TxEnqueuedLogHandler,
      },
      {
        topic: ethers.utils.id('CalldataTxEnqueued()'), // 0x3bfa105e8848abd2ed7abb76aee8a24f81bfe56a1c72823d073797f56508dd9e
        contractAddress: Environment.getOrThrow(
          Environment.safetyTransactionQueueContractAddress
        ),
        handleLog: CalldataTxEnqueuedLogHandler,
      },
      {
        topic: ethers.utils.id('L1ToL2BatchAppended(bytes32)'), // 0xe2708ee9d6a896e5f32f6edc61bc83143a1b8e3fbdf2a038c350369d251afb19
        contractAddress: Environment.getOrThrow(
          Environment.canonicalTransactionChainContractAddress
        ),
        handleLog: L1ToL2BatchAppendedLogHandler,
      },
      {
        topic: ethers.utils.id('SafetyQueueBatchAppended(bytes32)'), // 0x23764fe059fb5258ab47583dab9717481569b4f9631b4bcc7cb8cf2c79d1d5c2
        contractAddress: Environment.getOrThrow(
          Environment.canonicalTransactionChainContractAddress
        ),
        handleLog: SafetyQueueBatchAppendedLogHandler,
      },
      {
        topic: ethers.utils.id('SequencerBatchAppended(bytes32)'), // 0x256fdb5de9be2f545c62f9b8c453a7f8246978d0e1dd70970cc538b3203ef5ae
        contractAddress: Environment.getOrThrow(
          Environment.canonicalTransactionChainContractAddress
        ),
        handleLog: SequencerBatchAppendedLogHandler,
      },
      {
        topic: ethers.utils.id('StateBatchAppended(bytes32)'), // 0x800e6b30fb1a01e9038f324a049522a0231964e8de0aa9e815b35fc0029e8d52
        contractAddress: Environment.getOrThrow(
          Environment.stateCommitmentChainContractAddress
        ),
        handleLog: StateBatchAppendedLogHandler,
      },
    ],
    Environment.l1EarliestBlock()
  )
}

/**
 * Creates and returns an L2ChainDataPersister based on configured environment variables.
 *
 * @returns The L2ChainDataPersister.
 */
const createL2ChainDataPersister = async (): Promise<L2ChainDataPersister> => {
  return L2ChainDataPersister.create(
    getProcessingDataService(),
    getDataService(),
    getL2Provider()
  )
}

/**
 * Creates and returns an GethSubmissionQueuer based on configured environment variables.
 *
 * @returns The GethSubmissionQueuer.
 */
const createGethSubmissionQueuer = async (): Promise<GethSubmissionQueuer> => {
  const queueOriginsToSendToGeth = [
    QueueOrigin.L1_TO_L2_QUEUE,
    QueueOrigin.SAFETY_QUEUE,
  ]
  if (!Environment.isSequencerStack()) {
    queueOriginsToSendToGeth.push(QueueOrigin.SEQUENCER)
  }
  log.info(
    `Creating GethSubmissionQueuer with queue origins to queue: ${JSON.stringify(
      queueOriginsToSendToGeth
    )} and a period of ${Environment.gethSubmissionQueuerPeriodMillis()} millis`
  )

  return new GethSubmissionQueuer(
    getDataService(),
    queueOriginsToSendToGeth,
    Environment.getOrThrow(Environment.gethSubmissionQueuerPeriodMillis)
  )
}

/**
 * Creates and returns an QueuedGethSubmitter based on configured environment variables.
 *
 * @returns The QueuedGethSubmitter.
 */
const createQueuedGethSubmitter = async (): Promise<QueuedGethSubmitter> => {
  log.info(
    `Creating QueuedGethSubmitter with a period of ${Environment.queuedGethSubmitterPeriodMillis()} millis`
  )

  return new QueuedGethSubmitter(
    getDataService(),
    getL2NodeService(),
    Environment.getOrThrow(Environment.queuedGethSubmitterPeriodMillis)
  )
}

/**
 * Creates and returns a CanonicalChainBatchCreator based on configured environment variables.
 *
 * @returns The CanonicalChainBatchCreator.
 */
const createCanonicalChainBatchCreator = (): CanonicalChainBatchCreator => {
  const minBatchCalldataBytese: number = Environment.canonicalChainBatchMinL2TxBytes()
  const maxBatchCalldataBytes: number = Environment.canonicalChainMaxBatchTxBytes()
  const period: number = Environment.getOrThrow(
    Environment.canonicalChainBatchCreatorPeriodMillis
  )
  log.info(
    `Creating CanonicalChainBatchCreator with a min/max batch size of [${minBatchCalldataBytese}/${maxBatchCalldataBytes}] and period of ${period} millis`
  )

  return new CanonicalChainBatchCreator(
    getDataService(),
    minBatchCalldataBytese,
    maxBatchCalldataBytes,
    period
  )
}

/**
 * Creates and returns a CanonicalChainBatchSubmitter based on configured environment variables.
 *
 * @returns The CanonicalChainBatchSubmitter.
 */
const createCanonicalChainBatchSubmitter = (): CanonicalChainBatchSubmitter => {
  const canonicalTxChainAddress: string = Environment.getOrThrow(
    Environment.canonicalTransactionChainContractAddress
  )
  const l1ToL2TransactionQueueAddress: string = Environment.getOrThrow(
    Environment.l1ToL2TransactionQueueContractAddress
  )
  const safetyQueueAddress: string = Environment.getOrThrow(
    Environment.safetyTransactionQueueContractAddress
  )
  const period: number = Environment.getOrThrow(
    Environment.canonicalChainBatchSubmitterPeriodMillis
  )
  log.info(
    `Creating CanonicalChainBatchSubmitter with the canonical chain contract address of ${canonicalTxChainAddress}, l1 to l2 contract address of ${l1ToL2TransactionQueueAddress}, safety queue contract address of ${safetyQueueAddress}, and period of ${period} millis.`
  )

  const canonicalTxChainContract: Contract = new Contract(
    canonicalTxChainAddress,
    getContractDefinition('CanonicalTransactionChain').abi,
    getSequencerWallet()
  )

  const l1ToL2TransactionChainContract: Contract = new Contract(
    l1ToL2TransactionQueueAddress,
    getContractDefinition('L1ToL2TransactionQueue').abi,
    getSequencerWallet()
  )

  const safetyQueueContract: Contract = new Contract(
    safetyQueueAddress,
    getContractDefinition('SafetyTransactionQueue').abi,
    getSequencerWallet()
  )

  return new CanonicalChainBatchSubmitter(
    getDataService(),
    canonicalTxChainContract,
    l1ToL2TransactionChainContract,
    safetyQueueContract,
    getSequencerWallet(),
    period
  )
}

/**
 * Creates and returns a CanonicalChainBatchFinalizer based on configured environment variables.
 *
 * @returns The CanonicalChainBatchFinalizer.
 */
const createCanonicalChainBatchFinalizer = (): CanonicalChainBatchFinalizer => {
  const finalityDelay: number = Environment.getOrThrow(
    Environment.finalityDelayInBlocks
  )
  const period: number = Environment.getOrThrow(
    Environment.canonicalChainBatchSubmitterPeriodMillis
  )
  log.info(
    `Creating CanonicalChainBatchFinalizer with finality delay of ${finalityDelay} blocks, and period of ${period} millis`
  )

  return new CanonicalChainBatchFinalizer(
    getDataService(),
    getL1Provider(),
    finalityDelay,
    period
  )
}

/**
 * Creates and returns a StateCommitmentChainBatchCreator based on configured environment variables.
 *
 * @returns The StateCommitmentChainBatchCreator.
 */
const createStateCommitmentChainBatchCreator = (): StateCommitmentChainBatchCreator => {
  const minSize: number = Environment.stateCommitmentChainMinBatchSize(10)
  const maxSize: number = Environment.stateCommitmentChainMaxBatchSize(100)
  const period: number = Environment.getOrThrow(
    Environment.stateCommitmentChainBatchCreatorPeriodMillis
  )
  log.info(
    `Creating StateCommitmentChainBatchCreator with a min/max batch size of [${minSize}/${maxSize}] and period of ${period} millis`
  )

  return new StateCommitmentChainBatchCreator(
    getDataService(),
    minSize,
    maxSize,
    period
  )
}

/**
 * Creates and returns a StateCommitmentChainBatchSubmitter based on configured environment variables.
 *
 * @returns The StateCommitmentChainBatchSubmitter.
 */
const createStateCommitmentChainBatchSubmitter = (): StateCommitmentChainBatchSubmitter => {
  const contractAddress: string = Environment.getOrThrow(
    Environment.stateCommitmentChainContractAddress
  )
  const period: number = Environment.getOrThrow(
    Environment.stateCommitmentChainBatchSubmitterPeriodMillis
  )
  log.info(
    `Creating StateCommitmentChainBatchSubmitter with the state commitment chain contract address of ${contractAddress}, and period of ${period} millis`
  )

  return new StateCommitmentChainBatchSubmitter(
    getDataService(),
    new Contract(
      contractAddress,
      getContractDefinition('StateCommitmentChain').abi,
      getStateRootSubmissionWallet()
    ),
    getStateRootSubmissionWallet(),
    period
  )
}

/**
 * Creates and returns a StateCommitmentChainBatchFinalizer based on configured environment variables.
 *
 * @returns The StateCommitmentChainBatchFinalizer.
 */
const createStateCommitmentChainBatchFinalizer = (): StateCommitmentChainBatchFinalizer => {
  const finalityDelay: number = Environment.getOrThrow(
    Environment.finalityDelayInBlocks
  )
  const period: number = Environment.getOrThrow(
    Environment.stateCommitmentChainBatchSubmitterPeriodMillis
  )
  log.info(
    `Creating StateCommitmentChainBatchFinalizer with finality delay of ${finalityDelay} blocks, and period of ${period} millis.`
  )

  return new StateCommitmentChainBatchFinalizer(
    getDataService(),
    getL1Provider(),
    finalityDelay,
    period
  )
}

/**
 * Creates and returns a FraudDetector based on configured environment variables.
 *
 * @returns The FraudDetector.
 */
const createFraudDetector = (): FraudDetector => {
  const period: number = Environment.getOrThrow(
    Environment.fraudDetectorPeriodMillis
  )
  const realertEvery: number = Environment.getOrThrow(
    Environment.reAlertOnUnresolvedFraudEveryNFraudDetectorRuns
  )
  log.info(
    `Creating FraudDetector with a period of ${period} millis and a re-alert threshold of ${realertEvery} runs.`
  )

  return new FraudDetector(
    getDataService(),
    undefined, // TODO: ADD FRAUD PROVER HERE WHEN THERE IS ONE
    period,
    realertEvery
  )
}

const createL1BlockSubscriber = (
  lastBlockProcessed: number = 0
): EthereumBlockProcessor => {
  const startBlock = Math.max(
    lastBlockProcessed,
    Environment.getOrThrow(Environment.l1EarliestBlock)
  )
  log.info(`Starting subscription to L1 chain starting at block ${startBlock}`)
  return new EthereumBlockProcessor(
    getL1BlockProcessorDB(),
    startBlock,
    Environment.getOrThrow(Environment.finalityDelayInBlocks)
  )
}

const createL2BlockSubscriber = (
  lastBlockProcessed: number = 0
): EthereumBlockProcessor => {
  log.info(
    `Starting subscription to L2 node starting at block ${lastBlockProcessed}`
  )
  return new EthereumBlockProcessor(
    getL2BlockProcessorDB(),
    lastBlockProcessed,
    1
  )
}

/*********************
 * HELPER SINGLETONS *
 *********************/

const assertContractPresenceAndAddressMatch = (
  contract: any,
  expectedAddress: string,
  contractName: string
): void => {
  if (!contract) {
    const msg: string = `Contract ${contractName} was not deployed successfully!`
    log.error(msg)
    throw Error(msg)
  }

  if (remove0x(contract.address) !== remove0x(expectedAddress)) {
    const msg: string = `Contract ${contractName} does not have expected address. Expected ${expectedAddress}, got ${contract.address}.`
    log.error(msg)
    throw Error(msg)
  }
}

const deployContractsAndValidateDeployment = async (): Promise<void> => {
  let blockNumber: number
  let retries: number = 30
  while (retries > 0) {
    try {
      blockNumber = await getL1Provider().getBlockNumber()
      break
    } catch (e) {
      log.info(`Waiting for L1 node to be up... ${--retries} attempts left.`)
    }
  }

  if (blockNumber !== 0) {
    const msg: string = `L1 node returned block number ${blockNumber} when 0 was expected. Aborting deploy!`
    log.error(msg)
    throw Error(msg)
  }

  let res: DeployResult
  try {
    res = await deployContracts()
  } catch (e) {
    logError(log, `Error deploying contracts!`, e)
    throw e
  }

  assertContractPresenceAndAddressMatch(
    res.contracts.canonicalTransactionChain,
    Environment.canonicalTransactionChainContractAddress(),
    'CanonicalTransactionChain'
  )
  assertContractPresenceAndAddressMatch(
    res.contracts.stateCommitmentChain,
    Environment.stateCommitmentChainContractAddress(),
    'StateCommitmentChain'
  )
  assertContractPresenceAndAddressMatch(
    res.contracts.l1ToL2TransactionQueue,
    Environment.l1ToL2TransactionQueueContractAddress(),
    'L1ToL2TransactionQueue'
  )
  assertContractPresenceAndAddressMatch(
    res.contracts.safetyTransactionQueue,
    Environment.safetyTransactionQueueContractAddress(),
    'SafetyTransactionQueue'
  )

  log.info(`Contracts Deployed Successfully!`)
}

let l1BlockProcessorDb: DB
const getL1BlockProcessorDB = (): DB => {
  if (!l1BlockProcessorDb) {
    clearDataIfNecessary(
      Environment.getOrThrow(Environment.l1ChainDataPersisterLevelDbPath)
    )
    l1BlockProcessorDb = new BaseDB(
      getLevelInstance(
        Environment.getOrThrow(Environment.l1ChainDataPersisterLevelDbPath)
      ),
      256
    )
  }
  return l1BlockProcessorDb
}

let l2BlockProcessorDb: DB
const getL2BlockProcessorDB = (): DB => {
  if (!l2BlockProcessorDb) {
    clearDataIfNecessary(
      Environment.getOrThrow(Environment.l2ChainDataPersisterLevelDbPath)
    )
    l2BlockProcessorDb = new BaseDB(
      getLevelInstance(
        Environment.getOrThrow(Environment.l2ChainDataPersisterLevelDbPath)
      ),
      256
    )
  }
  return l2BlockProcessorDb
}

let rdb: RDB
const getRDBInstance = (): RDB => {
  if (!rdb) {
    rdb = new PostgresDB(
      Environment.getOrThrow(Environment.postgresHost),
      Environment.getOrThrow(Environment.postgresPort),
      Environment.getOrThrow(Environment.postgresUser),
      Environment.getOrThrow(Environment.postgresPassword),
      Environment.postgresDatabase('rollup'),
      Environment.postgresPoolSize(20),
      Environment.postgresUseSsl(false)
    )
  }
  return rdb
}

let dataService: DataService
const getDataService = (): DataService => {
  if (!dataService) {
    dataService = new DefaultDataService(getRDBInstance())
  }
  return dataService
}

let processingDataService: SequentialProcessingDataService
const getProcessingDataService = (): SequentialProcessingDataService => {
  if (!processingDataService) {
    processingDataService = new DefaultSequentialProcessingDataService(
      getRDBInstance()
    )
  }
  return processingDataService
}

let l2NodeService: L2NodeService
const getL2NodeService = (): L2NodeService => {
  if (!l2NodeService) {
    l2NodeService = new DefaultL2NodeService(getSubmitToL2GethWallet())
  }
  return l2NodeService
}

/**
 * Clears filesystem data at provided path if the Clear Data Key is set and changed
 * since the last startup.
 *
 * @param basePath The path to the data directory.
 */
const clearDataIfNecessary = (basePath: string): void => {
  if (
    Environment.clearDataKey() &&
    !fs.existsSync(getClearDataFilePath(basePath))
  ) {
    log.info(`Detected change in CLEAR_DATA_KEY. Purging data from ${basePath}`)
    rimraf.sync(`${basePath}/{*,.*}`)
    log.info(`Data purged from '${basePath}/{*,.*}'`)
    makeDataDirectory(basePath)
  }
}

/**
 * Makes a data directory at the provided base path.
 *
 * @param basePath The path at which a data directory should be created.
 */
const makeDataDirectory = (basePath: string) => {
  fs.mkdirSync(basePath, { recursive: true })
  if (Environment.clearDataKey()) {
    fs.writeFileSync(getClearDataFilePath(basePath), '')
  }
}

/**
 * Gets the path of the Clear Data file for the provided base path.
 *
 * @param basePath The path to the data directory.
 * @returns The full path to the clearData file.
 */
const getClearDataFilePath = (basePath: string): string => {
  return `${basePath}/.clear_data_key_${Environment.clearDataKey()}`
}
