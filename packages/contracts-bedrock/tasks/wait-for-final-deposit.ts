import { task, types } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { BigNumber } from 'ethers'
import { sleep, toRpcHexString } from '@eth-optimism/core-utils'

task('wait-for-final-deposit', 'Waits for the final deposit to be ingested')
  .addParam(
    'l1RpcUrl',
    'L1 RPC URL remote node',
    'http://127.0.0.1:8545',
    types.string
  )
  .addParam(
    'l2RpcUrl',
    'L2 RPC URL remote node',
    'http://127.0.0.1:9545',
    types.string
  )
  .setAction(async (args, hre: HardhatRuntimeEnvironment) => {
    const l1Provider = new hre.ethers.providers.StaticJsonRpcProvider(
      args.l1RpcUrl
    )
    const l2Provider = new hre.ethers.providers.StaticJsonRpcProvider(
      args.l2RpcUrl
    )

    // Handle legacy deployments
    let Deployment__AddressManager = await hre.deployments.getOrNull(
      'Lib_AddressManager'
    )
    if (!Deployment__AddressManager) {
      Deployment__AddressManager = await hre.deployments.get('AddressManager')
    }

    const AddressManager = new hre.ethers.Contract(
      Deployment__AddressManager.address,
      Deployment__AddressManager.abi,
      l1Provider
    )

    const Deployment__CanonicalTransactionChain = await hre.deployments.get(
      'CanonicalTransactionChain'
    )
    const CanonicalTransactionChain = new hre.ethers.Contract(
      Deployment__CanonicalTransactionChain.address,
      Deployment__CanonicalTransactionChain.abi,
      l1Provider
    )

    // Wait for DTL_SHUTOFF_BLOCK block to be set in the AddressManager
    let dtlShutoffBlock = BigNumber.from(0)
    while (true) {
      console.log('Waiting for DTL shutoff block to be set...')
      const val = await AddressManager.getAddress('DTL_SHUTOFF_BLOCK')
      dtlShutoffBlock = BigNumber.from(val)
      if (!dtlShutoffBlock.eq(0)) {
        break
      }
      await sleep(3000)
    }

    console.log(`DTL shutoff block ${dtlShutoffBlock.toString()}`)

    // Now query the number of queue elements in the CTC
    const queueLength = await CanonicalTransactionChain.getQueueLength()
    console.log(`Total number of deposits: ${queueLength}`)

    console.log('Searching backwards for final deposit')
    let height = await l2Provider.getBlockNumber()
    while (true) {
      console.log(`Trying block ${height}`)
      const hex = toRpcHexString(height)
      const b = await l2Provider.send('eth_getBlockByNumber', [hex, true])
      const tx = b.transactions[0]
      if (tx === undefined) {
        throw new Error(`unable to fetch transaction`)
      }

      if (tx.queueOrigin === 'l1') {
        const queueIndex = BigNumber.from(tx.queueIndex).toNumber()
        if (queueIndex === queueLength) {
          break
        }
        if (queueIndex < queueLength) {
          console.log()
          throw new Error(
            `Missed the final deposit. queueIndex ${queueIndex}, queueLength ${queueLength}`
          )
        }
      }
      height--
    }

    console.log('Final deposit has been ingested by l2geth')
  })
