import { task } from 'hardhat/config'
import { BigNumber, PopulatedTransaction } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

import { getModuleConfigs, isSameConfig } from '../src/config/faucet'

task('install-faucet-auth-module-configs').setAction(async (args, hre) => {
  const [signer] = await hre.ethers.getSigners()
  console.log('signer', signer.address)
  console.log(`connecting to Faucet...`)

  const Faucet = await hre.ethers.getContractAt(
    'Faucet',
    '0x6f324a7306c430489941990A25bA7268a69fd63e',
    signer
  )

  console.log(`loading local versions of module configs for network...`)
  const configs = await getModuleConfigs(hre)

  // Need this to deal with annoying Ethers/Ledger 1559 issue.
  const sendtx = async (tx: PopulatedTransaction): Promise<void> => {
    console.log('estimating gas')
    const gas = await signer.estimateGas(tx)
    tx.type = 1
    tx.gasLimit = gas
    tx.gasPrice = BigNumber.from(2000000000)
    console.log('sending tx')
    const ret = await signer.sendTransaction(tx)
    console.log(`sent tx: ${ret.hash}`)
    console.log(`waiting for tx to be confirmed...`)
    await ret.wait()
    console.log(`tx confirmed`)
  }

  console.log(`configuring modules...`)
  for (const [moduleName, moduleConfig] of Object.entries(configs)) {
    console.log(`checking config for module: ${moduleName}`)
    const contractModuleConfig = {
      name: moduleConfig.name,
      ttl: moduleConfig.ttl,
      enabled: moduleConfig.enabled,
      amount: moduleConfig.amount,
    }

    const currentModuleConfiguration = await Faucet.modules(
      moduleConfig.address
    )
    if (currentModuleConfiguration.name === '') {
      console.log(`module is not configured yet: ${moduleName}`)
      console.log(`configuring module...`)
      const tx = await Faucet.populateTransaction.configure(
        moduleConfig.address,
        contractModuleConfig
      )
      await sendtx(tx)
    } else if (
      !isSameConfig(currentModuleConfiguration, contractModuleConfig)
    ) {
      console.log(
        `module config exists but local config is different: ${moduleName}`
      )
      console.log(`updating config`)
      const tx = await Faucet.populateTransaction.configure(
        moduleConfig.address,
        contractModuleConfig
      )
      await sendtx(tx)
    } else {
      console.log(`config is already installed`)
    }
  }

  console.log(`configs are fully installed`)
})
