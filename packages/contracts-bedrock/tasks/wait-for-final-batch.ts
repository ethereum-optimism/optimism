import { task, types } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { Contract } from 'ethers'
import { sleep } from '@eth-optimism/core-utils'

task('wait-for-final-batch', 'Waits for the final batch to be submitted')
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

    const Deployment__CanonicalTransactionChain = await hre.deployments.get(
      'CanonicalTransactionChain'
    )
    const CanonicalTransactionChain = new hre.ethers.Contract(
      Deployment__CanonicalTransactionChain.address,
      Deployment__CanonicalTransactionChain.abi,
      l1Provider
    )

    const Deployment__StateCommitmentChain = await hre.deployments.get(
      'StateCommitmentChain'
    )

    const StateCommitmentChain = new hre.ethers.Contract(
      Deployment__StateCommitmentChain.address,
      Deployment__StateCommitmentChain.abi,
      l1Provider
    )

    const wait = async (contract: Contract) => {
      let height = await l2Provider.getBlockNumber()
      let totalElements = await contract.getTotalElements()
      // The genesis block was not batch submitted so subtract 1 from the height
      // when comparing with the total elements
      while (totalElements !== height - 1) {
        console.log('Total elements does not match')
        console.log(`  - real height: ${height}`)
        console.log(`  - height: ${height - 1}`)
        console.log(`  - totalElements: ${totalElements}`)
        totalElements = await contract.getTotalElements()
        height = await l2Provider.getBlockNumber()
        await sleep(2 * 1000)
      }
    }

    console.log('Waiting for the CanonicalTransactionChain...')
    await wait(CanonicalTransactionChain)
    console.log('All transaction batches have been submitted')

    console.log('Waiting for the StateCommitmentChain...')
    await wait(StateCommitmentChain)
    console.log('All state root batches have been submitted')

    console.log('All batches have been submitted')
  })
