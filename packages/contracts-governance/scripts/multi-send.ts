import fs from 'fs'

import { task } from 'hardhat/config'
import dotenv from 'dotenv'

import { prompt } from '../src/prompt'

dotenv.config()

task('multi-send', 'Send tokens to multiple addresses')
  .addOptionalParam(
    'privateKey',
    'Private Key for deployer account',
    process.env.PRIVATE_KEY_MULTI_SEND
  )
  .addParam('inFile', 'Distribution file')
  .setAction(async (args, hre) => {
    console.log(`Starting multi send on ${hre.network.name} network`)

    // Load the distribution setup
    const distributionJson = fs.readFileSync(args.inFile).toString()
    const distribution = JSON.parse(distributionJson)
    const sender = new hre.ethers.Wallet(args.privateKey).connect(
      hre.ethers.provider
    )

    const addr = await sender.getAddress()
    console.log(`Using deployer: ${addr}`)

    console.log('Performing multi send to the following addresses:')
    for (const [address, amount] of Object.entries(distribution)) {
      console.log(
        `${address}: ${amount} (${hre.ethers.utils.parseEther(
          amount as string
        )})`
      )
    }
    await prompt('Is this OK?')

    const governanceToken = (
      await hre.ethers.getContractAt(
        'GovernanceToken',
        '0x4200000000000000000000000000000000000042'
      )
    ).connect(sender)

    for (const [address, amount] of Object.entries(distribution)) {
      const amountBase = hre.ethers.utils.parseEther(amount as string)
      console.log(`Transferring ${amountBase} tokens to ${address}...`)
      const transferTx = await governanceToken.transfer(address, amountBase)
      console.log(`Waiting for tx ${transferTx.hash}`)
      await transferTx.wait()
    }

    console.log('Done.')
  })
