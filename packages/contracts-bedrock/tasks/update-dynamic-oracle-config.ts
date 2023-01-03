import readline from 'readline'

import { task, types } from 'hardhat/config'
import { ethers, Wallet } from 'ethers'

import { getContractsFromArtifacts } from '../src/deploy-utils'

task('update-dynamic-oracle-config', 'Updates the dynamic oracle config.')
  .addParam(
    'l2OutputOracleStartingTimestamp',
    'Starting timestamp for the L2 output oracle.',
    null,
    types.int
  )
  .addParam('noSend', 'Do not send the transaction.', true, types.boolean)
  .addOptionalParam(
    'privateKey',
    'Private key to send transaction',
    process.env.PRIVATE_KEY,
    types.string
  )
  .setAction(async (args, hre) => {
    const { l2OutputOracleStartingTimestamp, noSend, privateKey } = args
    const wallet = new Wallet(privateKey, hre.ethers.provider)
    const [SystemDictator] = await getContractsFromArtifacts(hre, [
      {
        name: 'SystemDictatorProxy',
        iface: 'SystemDictator',
        signerOrProvider: wallet,
      },
    ])

    const currStep = await SystemDictator.currentStep()
    if (currStep !== 5) {
      throw new Error(`Current step is ${currStep}, expected 5`)
    }
    if (await SystemDictator.dynamicConfigSet()) {
      throw new Error('Dynamic config already set')
    }

    const l2OutputOracleStartingBlockNumber =
      hre.deployConfig.l2OutputOracleStartingBlockNumber
    console.log(
      `This task will set the L2 output oracle's starting timestamp and block number.`
    )
    console.log(
      `It can only be run once. Please carefully check the values below:`
    )
    console.log(
      `L2OO starting block number:    ${l2OutputOracleStartingBlockNumber}`
    )
    console.log(
      `L2OO starting block timestamp: ${l2OutputOracleStartingTimestamp}`
    )

    await prompt('Press enter to continue...')

    if (noSend) {
      const tx =
        await SystemDictator.populateTransaction.updateL2OutputOracleDynamicConfig(
          {
            l2OutputOracleStartingBlockNumber,
            l2OutputOracleStartingTimestamp,
          }
        )
      console.log(`Sending is disabled. Transaction data:`)
      // Need to delete tx.from for Ethers to properly serialize the tx
      delete tx.from
      console.log(ethers.utils.serializeTransaction(tx))
      console.log(`Calldata (for multisigs):`)
      console.log(tx.data)
    } else {
      console.log(`Sending transaction...`)
      const tx = await SystemDictator.updateL2OutputOracleDynamicConfig({
        l2OutputOracleStartingBlockNumber,
        l2OutputOracleStartingTimestamp,
      })
      console.log(
        `Transaction sent with hash ${tx.hash}. Waiting for receipt...`
      )
      const receipt = await tx.wait(1)
      console.log(`Transaction included in block ${receipt.blockNumber}`)
    }
  })

const prompt = async (question: string) => {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  })

  return new Promise<void>((resolve) => {
    rl.question(question, () => {
      rl.close()
      resolve()
    })
  })
}
