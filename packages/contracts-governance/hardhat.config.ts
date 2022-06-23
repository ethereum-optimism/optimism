import dotenv from 'dotenv'
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-etherscan'
import '@nomiclabs/hardhat-waffle'
import 'hardhat-gas-reporter'
import 'solidity-coverage'
import { task, types } from 'hardhat/config'
import { providers, utils, Wallet } from 'ethers'
import { CrossChainMessenger } from '@eth-optimism/sdk'
import { getChainId } from '@eth-optimism/core-utils'

import './scripts/deploy-token'
import './scripts/multi-send'
import './scripts/mint-initial-supply'
import './scripts/generate-merkle-root'
import './scripts/create-airdrop-json'
import './scripts/deploy-distributor'
import './scripts/test-claims'
import './scripts/create-distributor-json'

dotenv.config()

task('accounts', 'Prints the list of accounts').setAction(async (args, hre) => {
  const accounts = await hre.ethers.getSigners()

  for (const account of accounts) {
    console.log(account.address)
  }
})

task('deposit', 'Deposits funds onto Optimism.')
  .addParam('to', 'Recipient address.', null, types.string)
  .addParam('amountEth', 'Amount in ETH to send.', null, types.string)
  .addParam('l1ProviderUrl', '', process.env.L1_PROVIDER_URL, types.string)
  .addParam('l2ProviderUrl', '', process.env.L2_PROVIDER_URL, types.string)
  .addParam('privateKey', '', process.env.PRIVATE_KEY, types.string)
  .setAction(async (args) => {
    const { to, amountEth, l1ProviderUrl, l2ProviderUrl, privateKey } = args
    if (!l1ProviderUrl || !l2ProviderUrl || !privateKey) {
      throw new Error(
        'You must define --l1-provider-url, --l2-provider-url, --private-key in your environment.'
      )
    }

    const l1Provider = new providers.JsonRpcProvider(l1ProviderUrl)
    const l2Provider = new providers.JsonRpcProvider(l2ProviderUrl)
    const l1Wallet = new Wallet(privateKey, l1Provider)
    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: l1Wallet,
      l2SignerOrProvider: l2Provider,
      l1ChainId: await getChainId(l1Provider),
      l2ChainId: await getChainId(l2Provider),
    })

    const amountWei = utils.parseEther(amountEth)
    console.log(`Depositing ${amountEth} ETH to ${to}...`)
    const tx = await messenger.depositETH(amountWei, {
      recipient: to,
    })
    console.log(`Got TX hash ${tx.hash}. Waiting...`)
    await tx.wait()

    const l1WalletOnL2 = new Wallet(privateKey, l2Provider)
    await l1WalletOnL2.sendTransaction({
      to,
      value: utils.parseEther(amountEth),
    })

    const balance = await l2Provider.getBalance(to)
    console.log('Funded account balance', balance.toString())
    console.log('Done.')
  })

const privKey = process.env.PRIVATE_KEY || '0x' + '11'.repeat(32)

/**
 * @type import("hardhat/config").HardhatUserConfig
 */
module.exports = {
  solidity: {
    version: '0.8.12',
    settings: {
      outputSelection: {
        '*': {
          '*': ['metadata', 'storageLayout'],
        },
      },
    },
  },
  networks: {
    optimism: {
      chainId: 17,
      url: 'http://localhost:8545',
      saveDeployments: false,
    },
    'optimism-kovan': {
      chainId: 69,
      url: 'https://kovan.optimism.io',
      accounts: [privKey],
    },
    'optimism-nightly': {
      chainId: 421,
      url: 'https://goerli-nightly-us-central1-a-sequencer.optimism.io',
      saveDeployments: true,
      accounts: [privKey],
    },
    'optimism-mainnet': {
      chainId: 10,
      url: 'https://mainnet.optimism.io',
      accounts: [privKey],
    },
    'hardhat-node': {
      url: 'http://localhost:9545',
      saveDeployments: false,
    },
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS !== undefined,
    currency: 'USD',
  },
  etherscan: {
    apiKey: process.env.ETHERSCAN_API_KEY,
  },
}
