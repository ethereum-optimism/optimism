import { task } from 'hardhat/config'
import { LedgerSigner } from '@ethersproject/hardware-wallets'
import { PopulatedTransaction } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

import { getModuleConfigs, isSameConfig } from '../src/config/faucet'

task('install-faucet-auth-module-configs').setAction(async (args, hre) => {
  // Add this back in after testing
  // console.log(`connecting to ledger...`)
  // const signer = new LedgerSigner(
  //   hre.ethers.provider,
  //   'default',
  //   hre.ethers.utils.defaultPath
  // )

  const [owner] = await hre.ethers.getSigners()
  const signer = owner
  console.log(`connecting to Faucet...`)

  const Faucet = await hre.ethers.getContractAt(
    'Faucet',
    (
      await hre.deployments.get('Faucet')
    ).address,
    owner
  )

  console.log(`loading local versions of module configs for network...`)
  const configs = await getModuleConfigs(hre)

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

  console.log(`configuring modules...`)
  for (const [moduleName, moduleConfig] of Object.entries(configs)) {
    console.log(`checking config for module: ${moduleName}`)
    const contractModuleConfig = {
      name: moduleConfig.name,
      ttl: moduleConfig.ttl,
      enabled: moduleConfig.enabled,
      amount: moduleConfig.amount,
    }
    const moduleAddress = (
      await hre.deployments.get(moduleConfig.authModuleDeploymentName)
    ).address
    const currentModuleConfiguration = await Faucet.modules(moduleAddress)
    if (currentModuleConfiguration.name === '') {
      console.log(`module is not configured yet: ${moduleName}`)
      console.log(`configuring module...`)
      const tx = await Faucet.populateTransaction.configure(
        moduleAddress,
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
        moduleAddress,
        contractModuleConfig
      )
      await sendtx(tx)
    } else {
      console.log(`config is already installed`)
    }
  }

  console.log(`config is fully installed`)
})
