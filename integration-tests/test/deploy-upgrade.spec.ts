import { expect } from 'chai'

/* Imports: External */
import { Wallet } from 'ethers'
import hre from 'hardhat'
import { task } from 'hardhat/config'
import { waitUntilTrue } from '@eth-optimism/contracts'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { OptimismEnv } from './shared/env'

describe('Deploying an upgraded system', async () => {
  let env: OptimismEnv
  let sequencerWallet: Wallet
  let proposerWallet: Wallet
  let finalAddressManagerOwner: Wallet

  before(
    'Get addresses of the contracts which will be reused after the upgrade',
    async () => {
      env = await OptimismEnv.new()
      sequencerWallet = Wallet.createRandom().connect(env.l1Wallet.provider)
      proposerWallet = Wallet.createRandom().connect(env.l1Wallet.provider)
      finalAddressManagerOwner = Wallet.createRandom().connect(
        env.l1Wallet.provider
      )
    }
  )

  it('Should successfully upgrade the system', async () => {
    console.log(await env.addressManager.owner())
    console.log(finalAddressManagerOwner.address)

    const result = await hre.run('deploy', {
      ovmSequencerAddress: sequencerWallet.address,
      ovmProposerAddress: proposerWallet.address,
      ovmAddressManagerOwner: finalAddressManagerOwner.address,
      libAddressManager: env.addressManager.address,
      proxyL1CrossDomainMessenger: env.l1Messenger.address,
      proxyL1StandardBridge: env.l1Bridge.address,
      numDeployConfirmations: 0,
      tags: 'upgrade',
    })

    await waitUntilTrue(
      async () => {
        const currentOwner = await env.addressManager.owner()
        console.log('currentOwner:', currentOwner)
        console.log(
          'finalAddressManagerOwner.address:',
          finalAddressManagerOwner.address
        )
        return hexStringEquals(finalAddressManagerOwner.address, currentOwner)
      },
      {
        // Try every 30 seconds for 500 minutes.
        delay: 1_000,
        retries: 1000,
      }
    )

    console.log(result)
  })
  it.skip('Should do things', async () => {
    // what should we test??
    // Addresses that don't change
    // Addresses that do change
    // verify things that should be initialized
  })
})
