/* External Imports */
import {getLogger, ScheduledTask} from '@eth-optimism/core-utils'
import { BaseDB, getLevelInstance, PostgresDB } from '@eth-optimism/core-db'
import { getContractDefinition } from '@eth-optimism/rollup-contracts'
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
} from '@eth-optimism/rollup-core'

import { Contract, ethers, Wallet } from 'ethers'
import { InfuraProvider, JsonRpcProvider, Provider } from 'ethers/providers'

const log = getLogger('service-entrypoint')

/**
 * Runs the configured Rollup services based on configured Environment variables.
 *
 * @returns The services being run.
 */
export const runServices = async (): Promise<any[]> => {
  log.info(`Running services!`)
  const services: any[] = []
  const scheduledTasks: ScheduledTask[] = []

  if (Environment.runL1ChainDataPersister()) {
    log.info(`Running L1 Chain Data Persister`)
    services.push(await createL1ChainDataPersister())
  }
  if (Environment.runL2ChainDataPersister()) {
    log.info(`Running L2 Chain Data Persister`)
    services.push(await createL2ChainDataPersister())
  }
  if (Environment.runGethSubmissionQueuer()) {
    log.info(`Running Geth Submission Queuer`)
    scheduledTasks.push(await createGethSubmissionQueuer())
  }
  if (Environment.runQueuedGethSubmitter()) {
    log.info(`Running Queued Geth Submitter`)
    scheduledTasks.push(await createQueuedGethSubmitter())
  }
  if (Environment.runCanonicalChainBatchCreator()) {
    log.info(`Running Canonical Chain Batch Creator`)
    scheduledTasks.push(await createCanonicalChainBatchCreator())
  }
  if (Environment.runCanonicalChainBatchSubmitter()) {
    log.info(`Running Canonical Chain Batch Submitter`)
    scheduledTasks.push(await createCanonicalChainBatchSubmitter())
  }
  if (Environment.runStateCommitmentChainBatchCreator()) {
    log.info(`Running State Commitment Chain Batch Creator`)
    scheduledTasks.push(await createStateCommitmentChainBatchCreator())
  }
  if (Environment.runStateCommitmentChainBatchSubmitter()) {
    log.info(`Running State Commitment Chain Batch Submitter`)
    scheduledTasks.push(await createStateCommitmentChainBatchSubmitter())
  }
  if (Environment.runFraudDetector()) {
    log.info(`Running Fraud Detector`)
    scheduledTasks.push(await createFraudDetector())
  }

  services.push(...scheduledTasks)

  if (!services.length) {
    log.error(`No services configured! Exiting =|`)
    process.exit(1)
  }

  await Promise.all(scheduledTasks.map(x => x.start()))

  setInterval(() => {
    updateEnvironmentVariables()
  }, 179_000)

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
  return L1ChainDataPersister.create(
    new BaseDB(
      getLevelInstance(
        Environment.getOrThrow(Environment.l1ChainDataPersisterLevelDbPath)
      ),
      256
    ),
    getDataService(),
    getL1Provider(),
    [
      {
        topic: ethers.utils.id('L1ToL2TxEnqueued(bytes)'),
        contractAddress: Environment.getOrThrow(
          Environment.l1ToL2TransactionQueueContractAddress
        ),
        handleLog: L1ToL2TxEnqueuedLogHandler,
      },
      {
        topic: ethers.utils.id('event CalldataTxEnqueued()'),
        contractAddress: Environment.getOrThrow(
          Environment.safetyTransactionQueueContractAddress
        ),
        handleLog: CalldataTxEnqueuedLogHandler,
      },
      {
        topic: ethers.utils.id('L1ToL2BatchAppended(bytes32)'),
        contractAddress: Environment.getOrThrow(
          Environment.canonicalTransactionChainContractAddress
        ),
        handleLog: L1ToL2BatchAppendedLogHandler,
      },
      {
        topic: ethers.utils.id('SafetyQueueBatchAppended(bytes32)'),
        contractAddress: Environment.getOrThrow(
          Environment.canonicalTransactionChainContractAddress
        ),
        handleLog: SafetyQueueBatchAppendedLogHandler,
      },
      {
        topic: ethers.utils.id('SequencerBatchAppended(bytes32)'),
        contractAddress: Environment.getOrThrow(
          Environment.canonicalTransactionChainContractAddress
        ),
        handleLog: SequencerBatchAppendedLogHandler,
      },
      {
        topic: ethers.utils.id('StateBatchAppended(bytes32)'),
        contractAddress: Environment.getOrThrow(
          Environment.stateCommitmentChainContractAddress
        ),
        handleLog: StateBatchAppendedLogHandler,
      },
    ]
  )
}

/**
 * Creates and returns an L2ChainDataPersister based on configured environment variables.
 *
 * @returns The L2ChainDataPersister.
 */
const createL2ChainDataPersister = async (): Promise<L2ChainDataPersister> => {
  return L2ChainDataPersister.create(
    new BaseDB(
      getLevelInstance(
        Environment.getOrThrow(Environment.l2ChainDataPersisterLevelDbPath)
      )
    ),
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
  return new CanonicalChainBatchCreator(
    getDataService(),
    Environment.canonicalChainMinBatchSize(10),
    Environment.canonicalChainMaxBatchSize(100),
    Environment.getOrThrow(Environment.canonicalChainBatchCreatorPeriodMillis)
  )
}

/**
 * Creates and returns a CanonicalChainBatchSubmitter based on configured environment variables.
 *
 * @returns The CanonicalChainBatchSubmitter.
 */
const createCanonicalChainBatchSubmitter = (): CanonicalChainBatchSubmitter => {
  const contract: Contract = new Contract(
    Environment.getOrThrow(
      Environment.canonicalTransactionChainContractAddress
    ),
    getContractDefinition('CanonicalTransactionChain'),
    getSequencerWallet()
  )

  return new CanonicalChainBatchSubmitter(
    getDataService(),
    contract,
    Environment.getOrThrow(Environment.finalityDelayInBlocks),
    Environment.getOrThrow(Environment.canonicalChainBatchSubmitterPeriodMillis)
  )
}

/**
 * Creates and returns a StateCommitmentChainBatchCreator based on configured environment variables.
 *
 * @returns The StateCommitmentChainBatchCreator.
 */
const createStateCommitmentChainBatchCreator = (): StateCommitmentChainBatchCreator => {
  return new StateCommitmentChainBatchCreator(
    getDataService(),
    Environment.stateCommitmentChainMinBatchSize(10),
    Environment.stateCommitmentChainMaxBatchSize(100),
    Environment.getOrThrow(
      Environment.stateCommitmentChainBatchCreatorPeriodMillis
    )
  )
}

/**
 * Creates and returns a StateCommitmentChainBatchSubmitter based on configured environment variables.
 *
 * @returns The StateCommitmentChainBatchSubmitter.
 */
const createStateCommitmentChainBatchSubmitter = (): StateCommitmentChainBatchSubmitter => {
  return new StateCommitmentChainBatchSubmitter(
    getDataService(),
    new Contract(
      Environment.getOrThrow(Environment.stateCommitmentChainContractAddress),
      getContractDefinition('StateCommitmentChain'),
      getSequencerWallet()
    ),
    Environment.getOrThrow(Environment.finalityDelayInBlocks),
    Environment.getOrThrow(
      Environment.stateCommitmentChainBatchSubmitterPeriodMillis
    )
  )
}

/**
 * Creates and returns a FraudDetector based on configured environment variables.
 *
 * @returns The FraudDetector.
 */
const createFraudDetector = (): FraudDetector => {
  return new FraudDetector(
    getDataService(),
    undefined, // TODO: ADD FRAUD PROVER HERE WHEN THERE IS ONE
    Environment.getOrThrow(Environment.fraudDetectorPeriodMillis),
    Environment.getOrThrow(
      Environment.reAlertOnUnresolvedFraudEveryNFraudDetectorRuns
    )
  )
}

/*********************
 * HELPER SINGLETONS *
 *********************/

let dataService: DataService
const getDataService = (): DataService => {
  if (!dataService) {
    dataService = new DefaultDataService(
      new PostgresDB(
        Environment.getOrThrow(Environment.postgresHost),
        Environment.getOrThrow(Environment.postgresPort),
        Environment.getOrThrow(Environment.postgresUser),
        Environment.getOrThrow(Environment.postgresPassword),
        Environment.postgresDatabase('rollup'),
        Environment.postgresPoolSize(20),
        Environment.postgresUseSsl(false)
      )
    )
  }
  return dataService
}

let l1Provider: Provider
const getL1Provider = (): Provider => {
  if (!l1Provider) {
    if (
      !!Environment.l1NodeInfuraNetwork() &&
      !!Environment.l1NodeInfuraProjectId()
    ) {
      l1Provider = new InfuraProvider(
        Environment.getOrThrow(Environment.l1NodeInfuraNetwork),
        Environment.getOrThrow(Environment.l1NodeInfuraProjectId)
      )
    } else {
      l1Provider = new JsonRpcProvider(
        Environment.getOrThrow(Environment.l1NodeWeb3Url)
      )
    }
  }
  return l1Provider
}

let l2Provider: Provider
const getL2Provider = (): Provider => {
  if (!l2Provider) {
    l2Provider = new JsonRpcProvider(
      Environment.getOrThrow(Environment.l2NodeWeb3Url)
    )
  }
  return l2Provider
}

let l2NodeService: L2NodeService
const getL2NodeService = (): L2NodeService => {
  if (!l2NodeService) {
    l2NodeService = new DefaultL2NodeService(getSubmitToL2GethWallet())
  }
  return l2NodeService
}

let submitToL2GethWallet: Wallet
const getSubmitToL2GethWallet = (): Wallet => {
  if (!submitToL2GethWallet) {
    submitToL2GethWallet = new Wallet(
      Environment.getOrThrow(Environment.submitToL2GethPrivateKey),
      getL2Provider()
    )
  }
  return submitToL2GethWallet
}

let sequencerWallet: Wallet
const getSequencerWallet = (): Wallet => {
  if (!sequencerWallet) {
    sequencerWallet = new Wallet(
      Environment.getOrThrow(Environment.sequencerPrivateKey),
      getL1Provider()
    )
  }
  return sequencerWallet
}
