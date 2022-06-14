/* eslint @typescript-eslint/no-var-requires: "off" */
import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { awaitCondition } from '@eth-optimism/core-utils'

import {
  getContractFromArtifact,
  fundAccount,
  sendImpersonatedTx,
  BIG_BALANCE,
} from '../src/deploy-utils'
import { names } from '../src/address-names'

const deployFn: DeployFunction = async (hre) => {
  if (!hre.deployConfig.isForkedNetwork) {
    return
  }

  console.log(`Running custom setup for forked experimental networks`)
  const { deployer } = await hre.getNamedAccounts()

  // Fund the deployer account so it can be used for the rest of this deployment.
  console.log(`Funding deployer account...`)
  await fundAccount(hre, deployer, BIG_BALANCE)

  // Get a reference to the AddressManager contract.
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    names.unmanaged.Lib_AddressManager
  )

  // Transfer ownership of the AddressManager to the deployer.
  console.log(`Setting AddressManager owner to ${deployer}`)
  await sendImpersonatedTx({
    hre,
    contract: Lib_AddressManager,
    fn: 'transferOwnership',
    from: await Lib_AddressManager.owner(),
    gas: ethers.BigNumber.from(2_000_000).toHexString(),
    args: [deployer],
  })

  console.log(`Waiting for owner to be correctly set...`)
  await awaitCondition(
    async () => {
      return (await Lib_AddressManager.owner()) === deployer
    },
    5000,
    100
  )

  // Get a reference to the L1StandardBridge contract.
  const Proxy__OVM_L1StandardBridge = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1StandardBridge'
  )

  // Transfer ownership of the L1StandardBridge to the deployer.
  console.log(`Setting L1StandardBridge owner to ${deployer}`)
  await sendImpersonatedTx({
    hre,
    contract: Proxy__OVM_L1StandardBridge,
    fn: 'setOwner',
    from: await Proxy__OVM_L1StandardBridge.callStatic.getOwner({
      from: hre.ethers.constants.AddressZero,
    }),
    gas: ethers.BigNumber.from(2_000_000).toHexString(),
    args: [deployer],
  })

  console.log(`Waiting for owner to be correctly set...`)
  await awaitCondition(
    async () => {
      return (
        (await Proxy__OVM_L1StandardBridge.callStatic.getOwner({
          from: hre.ethers.constants.AddressZero,
        })) === deployer
      )
    },
    5000,
    100
  )
}

deployFn.tags = ['hardhat', 'upgrade']

export default deployFn
