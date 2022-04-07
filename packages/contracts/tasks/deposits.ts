import { task, types } from 'hardhat/config'
import { Contract, providers, utils, Wallet } from 'ethers'
import dotenv from 'dotenv'

dotenv.config()

task('deposit', 'Deposits funds onto L2.')
  .addParam(
    'l1ProviderUrl',
    'L1 provider URL.',
    'http://localhost:8545',
    types.string
  )
  .addParam('to', 'Recipient address.', null, types.string)
  .addParam('amountEth', 'Amount in ETH to send.', null, types.string)
  .addOptionalParam(
    'privateKey',
    'Private key to send transaction',
    process.env.PRIVATE_KEY,
    types.string
  )
  .addOptionalParam(
    'depositContractAddr',
    'Address of deposit contract.',
    'deaddeaddeaddeaddeaddeaddeaddeaddead0001',
    types.string
  )
  .setAction(async (args) => {
    const { l1ProviderUrl, to, amountEth, depositContractAddr, privateKey } =
      args
    const depositFeedArtifact = require('../artifacts/contracts/L1/DepositFeed.sol/DepositFeed.json')

    const l1Provider = new providers.JsonRpcProvider(l1ProviderUrl)

    let l1Wallet: Wallet | providers.JsonRpcSigner
    if (privateKey) {
      l1Wallet = new Wallet(privateKey, l1Provider)
    } else {
      l1Wallet = l1Provider.getSigner()
    }

    const from = await l1Wallet.getAddress()
    console.log(`Sending from ${from}`)
    const balance = await l1Wallet.getBalance()
    if (balance.eq(0)) {
      throw new Error(`${from} has no balance`)
    }

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
