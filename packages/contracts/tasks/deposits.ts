import { task, types } from 'hardhat/config'
import { Contract, providers, utils, Wallet } from 'ethers'
import dotenv from 'dotenv'

task('deposit', 'Deposits funds onto L2.')
  .addParam('l1ProviderUrl', 'L1 provider URL.', null, types.string)
  .addParam('to', 'Recipient address.', null, types.string)
  .addParam('amountEth', 'Amount in ETH to send.', null, types.string)
  .addOptionalParam(
    'depositContractAddr',
    'Address of deposit contract.',
    'deaddeaddeaddeaddeaddeaddeaddeaddead0001',
    types.string
  )
  .setAction(async ({ l1ProviderUrl, to, amountEth, depositContractAddr }) => {
    const depositFeedArtifact = require('../artifacts/contracts/L1/DepositFeed.sol/DepositFeed.json')

    dotenv.config()

    if (!process.env.PRIVATE_KEY) {
      throw new Error('You must define PRIVATE_KEY in your environment.')
    }

    const l1Provider = new providers.JsonRpcProvider(l1ProviderUrl)
    const l1Wallet = new Wallet(process.env.PRIVATE_KEY!, l1Provider)
    const depositFeed = new Contract(
      depositContractAddr,
      depositFeedArtifact.abi
    ).connect(l1Wallet)

    const amountWei = utils.parseEther(amountEth)
    console.log(`Depositing ${amountEth} ETH to ${to}...`)
    // Below adds 0.01 ETH to account for gas.
    const tx = await depositFeed.depositTransaction(
      to,
      amountWei,
      '3000000',
      false,
      [],
      {
        value: amountWei.add(utils.parseEther('0.01')),
      }
    )
    console.log(`Got TX hash ${tx.hash}. Waiting...`)
    await tx.wait()
    console.log('Done.')
  })
