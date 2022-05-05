import { task } from 'hardhat/config'
import { providers } from 'ethers'
import { getChainId } from '@eth-optimism/core-utils'

import { die, logStderr } from '../test/shared/utils'

task(
  'check-block-hashes',
  'Compares the block hashes of two different replicas.'
)
  .addPositionalParam('replicaA', 'The first replica')
  .addPositionalParam('replicaB', 'The second replica')
  .setAction(async ({ replicaA, replicaB }) => {
    const providerA = new providers.JsonRpcProvider(replicaA)
    const providerB = new providers.JsonRpcProvider(replicaB)

    let chainIdA
    let chainIdB
    try {
      chainIdA = await getChainId(providerA)
    } catch (e) {
      console.error(`Error getting network chainId from ${replicaA}:`)
      die(e)
    }
    try {
      chainIdB = await getChainId(providerB)
    } catch (e) {
      console.error(`Error getting network chainId from ${replicaB}:`)
      die(e)
    }

    if (chainIdA !== chainIdB) {
      die('Chain IDs do not match')
      return
    }

    logStderr('Getting block height.')
    const heightA = await providerA.getBlockNumber()
    const heightB = await providerB.getBlockNumber()
    const endHeight = Math.min(heightA, heightB)
    logStderr(`Chose block height: ${endHeight}`)

    for (let n = endHeight; n >= 1; n--) {
      const blocks = await Promise.all([
        providerA.getBlock(n),
        providerB.getBlock(n),
      ])

      const hashA = blocks[0].hash
      const hashB = blocks[1].hash
      if (hashA !== hashB) {
        console.log(`HASH MISMATCH! block=${n} a=${hashA} b=${hashB}`)
        continue
      }

      console.log(`HASHES OK! block=${n} hash=${hashA}`)
      return
    }
  })
