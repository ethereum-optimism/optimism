import { task } from 'hardhat/config'
import { LedgerSigner } from '@ethersproject/hardware-wallets'
import { PopulatedTransaction } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

import { isSameConfig, getDrippieConfig } from '../src'

task('install-drippie-config').setAction(async (args, hre) => {
  console.log(`connecting to ledger...`)
  const signer = new LedgerSigner(
    hre.ethers.provider,
    'default',
    hre.ethers.utils.defaultPath
  )

  console.log(`connecting to Drippie...`)
  const Drippie = await hre.ethers.getContractAt(
    'Drippie',
    (
      await hre.deployments.get('Drippie')
    ).address,
    signer
  )

  console.log(`loading local version of Drippie config for network...`)
  const config = await getDrippieConfig(hre)

  // Need this to deal with annoying Ethers/Ledger 1559 issue.
  const sendtx = async (tx: PopulatedTransaction): Promise<void> => {
    const gas = await signer.estimateGas(tx)
    tx.type = 1
    tx.gasLimit = gas
    const ret = await signer.sendTransaction(tx)
    console.log(`sent tx: ${ret.hash}`)
    console.log(`waiting for tx to be confirmed...`)
    await ret.wait()
    console.log(`tx confirmed`)
  }

  console.log(`installing Drippie config file...`)
  for (const [dripName, dripConfig] of Object.entries(config)) {
    console.log(`checking config for drip: ${dripName}`)
    const drip = await Drippie.drips(dripName)
    if (drip.status === 0) {
      console.log(`drip does not exist yet: ${dripName}`)
      console.log(`creating drip...`)
      const tx = await Drippie.populateTransaction.create(dripName, dripConfig)
      await sendtx(tx)
    } else if (!isSameConfig(dripConfig, drip.config)) {
      console.log(`drip exists but local config is different: ${dripName}`)
      console.log(`drips cannot be modified for security reasons`)
      console.log(`please do not modify the local config for existing drips`)
      console.log(`you can archive the old drip and create another`)
    } else {
      console.log(`drip is already installed`)
    }
  }

  console.log(`config is fully installed`)
})
