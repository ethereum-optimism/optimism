import { utils, Wallet, providers, constants } from 'ethers'
import {
  CrossChainMessenger,
  ETHBridgeAdapter,
  StandardBridgeAdapter,
} from '@eth-optimism/sdk'
import { predeploys } from '@eth-optimism/contracts-bedrock'
import { sleep } from '@eth-optimism/core-utils'

import { actor, setupActor, run } from '../lib/convenience'

interface Context {
  wallet: Wallet
}

actor('Dev account sender', () => {
  let l1Provider: providers.JsonRpcProvider

  let l2Provider: providers.JsonRpcProvider

  let wallet: Wallet

  let messenger: CrossChainMessenger

  let contracts: any

  setupActor(async () => {
    l1Provider = new providers.JsonRpcProvider(process.env.L1_RPC)
    l2Provider = new providers.JsonRpcProvider(process.env.L2_RPC)
    wallet = new Wallet(process.env.PRIVATE_KEY)
    contracts = require(process.env.CONTRACTS_JSON_PATH)
    messenger = new CrossChainMessenger({
      l1SignerOrProvider: wallet.connect(l1Provider),
      l2SignerOrProvider: wallet.connect(l2Provider),
      l1ChainId: (await l1Provider.getNetwork()).chainId,
      l2ChainId: (await l2Provider.getNetwork()).chainId,
      bridges: {
        Standard: {
          Adapter: StandardBridgeAdapter,
          l1Bridge: contracts.L1StandardBridgeProxy,
          l2Bridge: predeploys.L2StandardBridge,
        },
        ETH: {
          Adapter: ETHBridgeAdapter,
          l1Bridge: contracts.L1StandardBridgeProxy,
          l2Bridge: predeploys.L2StandardBridge,
        },
      },
      contracts: {
        l1: {
          AddressManager: constants.AddressZero,
          StateCommitmentChain: constants.AddressZero,
          CanonicalTransactionChain: constants.AddressZero,
          BondManager: constants.AddressZero,
          L1StandardBridge: contracts.L1StandardBridgeProxy,
          L1CrossDomainMessenger: contracts.L1CrossDomainMessengerProxy,
          L2OutputOracle: contracts.L2OutputOracleProxy,
          OptimismPortal: contracts.OptimismPortalProxy,
        },
      },
      bedrock: true,
    })
  })

  run(async (b, ctx: Context, logger) => {
    const recipient = Wallet.createRandom().connect(l2Provider)
    logger.log(`Depositing funds to ${recipient.address}.`)
    const depositTx = await messenger.depositETH(utils.parseEther('0.0001'), {
      recipient: recipient.address,
    })
    logger.log(`Awaiting receipt for deposit tx ${depositTx.hash}.`)
    await depositTx.wait()
    // Temporary until this is supported in the SDK.
    for (let i = 0; i < 60; i++) {
      const recipBal = await recipient.getBalance()
      logger.log(`Polling L2 for deposit completion.`)
      if (recipBal.eq(utils.parseEther('0.0001'))) {
        logger.log('Deposit successful.')
        return
      }
      await sleep(1000)
    }
    throw new Error('Timed out.')
  })
})
