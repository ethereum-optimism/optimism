import { utils, Wallet, constants } from 'ethers'
import {
  CrossChainMessenger,
  ETHBridgeAdapter,
  StandardBridgeAdapter,
} from '@eth-optimism/sdk'
import { predeploys } from '@eth-optimism/contracts-bedrock'
import { sleep } from '@eth-optimism/core-utils'

import { actor, setupActor, run, setupRun } from '../lib/convenience'
import { l1Provider, l2Provider } from './utils'
import { Faucet } from '../lib/faucet'

interface Context {
  wallet: Wallet
  messenger: CrossChainMessenger
}

actor('Depositor', () => {
  let contracts: any

  setupActor(async () => {
    contracts = require(process.env.CONTRACTS_JSON_PATH)
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom().connect(l1Provider)
    const faucet = new Faucet(process.env.FAUCET_URL, l1Provider)
    await faucet.drip(wallet.address)

    return {
      wallet,
      messenger: new CrossChainMessenger({
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
      }),
    }
  })

  run(async (b, ctx: Context, logger) => {
    const { messenger } = ctx
    const recipient = Wallet.createRandom().connect(l2Provider)
    logger.log(`Depositing funds to ${recipient.address}.`)
    const depositTx = await messenger.depositETH(utils.parseEther('0.000001'), {
      recipient: recipient.address,
    })
    logger.log(`Awaiting receipt for deposit tx ${depositTx.hash}.`)
    await depositTx.wait()
    // Temporary until this is supported in the SDK.
    for (let i = 0; i < 60; i++) {
      const recipBal = await recipient.getBalance()
      logger.log(`Polling L2 for deposit completion.`)
      if (recipBal.eq(utils.parseEther('0.000001'))) {
        logger.log('Deposit successful.')
        return
      }
      await sleep(1000)
    }
    throw new Error('Timed out.')
  })
})
