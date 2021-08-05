import { providers } from 'ethers'
import { Logger } from '@eth-optimism/common-ts'

import { HealthcheckServerOptions } from './healthcheck-server'

export const readEnvOrQuitProcess = (envName: string | undefined): string => {
  if (!process.env[envName]) {
    console.error(`Missing environment variable: ${envName}`)
    process.exit(1)
  }
  return process.env[envName]
}

export const readConfig = (): HealthcheckServerOptions => {
  const network = readEnvOrQuitProcess('REPLICA_HEALTHCHECK__ETH_NETWORK')
  const gethRelease = readEnvOrQuitProcess(
    'REPLICA_HEALTHCHECK__L2GETH_IMAGE_TAG'
  )
  const sequencerRpcProvider = readEnvOrQuitProcess(
    'REPLICA_HEALTHCHECK__ETH_NETWORK_RPC_PROVIDER'
  )
  const replicaRpcProvider = readEnvOrQuitProcess(
    'REPLICA_HEALTHCHECK__ETH_REPLICA_RPC_PROVIDER'
  )

  if (!['mainnet', 'kovan', 'goerli'].includes(network)) {
    console.error(
      'Invalid ETH_NETWORK specified. Must be one of mainnet, kovan, or goerli'
    )
    process.exit(1)
  }

  const logger = new Logger({ name: 'replica-healthcheck' })

  return {
    network,
    gethRelease,
    sequencerRpcProvider,
    replicaRpcProvider,
    logger,
  }
}

export const binarySearchForMismatch = async (
  sequencerProvider: providers.JsonRpcProvider,
  replicaProvider: providers.JsonRpcProvider,
  latest: number,
  logger: Logger
): Promise<number> => {
  logger.info(
    'Executing a binary search to determine the first mismatched block...'
  )

  let start = 0
  let end = latest
  while (start !== end) {
    const middle = Math.floor((start + end) / 2)

    logger.info('Checking block', { blockNumber: middle })
    const [replicaBlock, sequencerBlock] = await Promise.all([
      replicaProvider.getBlock(middle) as any,
      sequencerProvider.getBlock(middle) as any,
    ])

    if (replicaBlock.stateRoot === sequencerBlock.stateRoot) {
      logger.info('State roots still matching', { blockNumber: middle })
      start = middle
    } else {
      logger.error('Found mismatched state roots', {
        blockNumber: middle,
        sequencerBlock,
        replicaBlock,
      })

      end = middle
    }
  }

  return end
}
