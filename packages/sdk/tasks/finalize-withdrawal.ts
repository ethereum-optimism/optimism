import { task, types } from 'hardhat/config'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { Wallet, providers } from 'ethers'
import { predeploys } from '@eth-optimism/contracts-bedrock'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'

import {
  CrossChainMessenger,
  StandardBridgeAdapter,
  MessageStatus,
} from '../src'

task('finalize-withdrawal', 'Finalize a withdrawal')
  .addParam(
    'transactionHash',
    'L2 Transaction hash to finalize',
    '',
    types.string
  )
  .addParam('l2Url', 'L2 HTTP URL', 'http://localhost:9545', types.string)
  .setAction(async (args, hre: HardhatRuntimeEnvironment) => {
    const txHash = args.transactionHash
    if (txHash === '') {
      console.log('No tx hash')
    }

    const signers = await hre.ethers.getSigners()
    if (signers.length === 0) {
      throw new Error('No configured signers')
    }
    const signer = signers[0]
    const address = await signer.getAddress()
    console.log(`Using signer: ${address}`)

    const l2Provider = new providers.StaticJsonRpcProvider(args.l2Url)
    const l2Signer = new Wallet(hre.network.config.accounts[0], l2Provider)

    let Deployment__L1StandardBridgeProxy = await hre.deployments.getOrNull(
      'L1StandardBridgeProxy'
    )
    if (Deployment__L1StandardBridgeProxy === undefined) {
      Deployment__L1StandardBridgeProxy = await hre.deployments.getOrNull(
        'Proxy__OVM_L1StandardBridge'
      )
    }

    let Deployment__L1CrossDomainMessengerProxy =
      await hre.deployments.getOrNull('L1CrossDomainMessengerProxy')
    if (Deployment__L1CrossDomainMessengerProxy === undefined) {
      Deployment__L1CrossDomainMessengerProxy = await hre.deployments.getOrNull(
        'Proxy__OVM_L1CrossDomainMessenger'
      )
    }

    const Deployment__L2OutputOracleProxy = await hre.deployments.getOrNull(
      'L2OutputOracleProxy'
    )
    const Deployment__OptimismPortalProxy = await hre.deployments.getOrNull(
      'OptimismPortalProxy'
    )

    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: signer,
      l2SignerOrProvider: l2Signer,
      l1ChainId: await signer.getChainId(),
      l2ChainId: await l2Signer.getChainId(),
      bridges: {
        Standard: {
          Adapter: StandardBridgeAdapter,
          l1Bridge: Deployment__L1StandardBridgeProxy?.address,
          l2Bridge: predeploys.L2StandardBridge,
        },
      },
      contracts: {
        l1: {
          L1StandardBridge: Deployment__L1StandardBridgeProxy?.address,
          L1CrossDomainMessenger:
            Deployment__L1CrossDomainMessengerProxy?.address,
          L2OutputOracle: Deployment__L2OutputOracleProxy?.address,
          OptimismPortal: Deployment__OptimismPortalProxy?.address,
        },
      },
    })

    console.log(`Fetching message status for ${txHash}`)
    const status = await messenger.getMessageStatus(txHash)
    console.log(`Status: ${MessageStatus[status]}`)

    if (status === MessageStatus.READY_TO_PROVE) {
      const proveTx = await messenger.proveMessage(txHash)
      const proveReceipt = await proveTx.wait()
      console.log('Prove receipt', proveReceipt)

      const finalizeInterval = setInterval(async () => {
        const currentStatus = await messenger.getMessageStatus(txHash)
        console.log(`Message status: ${MessageStatus[currentStatus]}`)
      }, 3000)

      try {
        await messenger.waitForMessageStatus(
          txHash,
          MessageStatus.READY_FOR_RELAY
        )
      } finally {
        clearInterval(finalizeInterval)
      }

      const tx = await messenger.finalizeMessage(txHash)
      const receipt = await tx.wait()
      console.log(receipt)
      console.log('Finalized withdrawal')
    }
  })
