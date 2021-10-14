/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { predeploys } from '../src/predeploys'
import {
  getContractInterface,
  getContractDefinition,
} from '../src/contract-defs'
import {
  getDeployedContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager'
  )

  // Set up a reference to the proxy as if it were the L1StandardBridge contract.
  const contract = await getDeployedContract(hre, 'Proxy__L1StandardBridge', {
    iface: 'L1StandardBridge',
    signerOrProvider: deployer,
  })

  // Because of the `iface` parameter supplied to the deployment function above, the `contract`
  // variable that we here will have the interface of the L1StandardBridge contract. However,
  // we also need to interact with the contract as if it were a L1ChugSplashProxy contract so
  // we instantiate a new ethers.Contract object with the same address and signer but with the
  // L1ChugSplashProxy interface.
  const proxy = new ethers.Contract(
    contract.address,
    getContractInterface('L1ChugSplashProxy'),
    contract.signer
  )

  // First we need to set the correct implementation code. We'll set the code and then check
  // that the code was indeed correctly set.
  const bridgeArtifact = getContractDefinition('L1StandardBridge')
  const bridgeCode = bridgeArtifact.deployedBytecode

  console.log(`Setting bridge code...`)
  await proxy.setCode(bridgeCode)

  console.log(`Confirming that bridge code is correct...`)
  await waitUntilTrue(async () => {
    const implementation = await proxy.callStatic.getImplementation()
    return (
      !hexStringEquals(implementation, ethers.constants.AddressZero) &&
      hexStringEquals(
        await contract.provider.getCode(implementation),
        bridgeCode
      )
    )
  })

  // Next we need to set the `messenger` address by executing a setStorage operation. We'll
  // check that this operation was correctly executed by calling `messenger()` and checking
  // that the result matches the value we initialized.
  const l1CrossDomainMessengerAddress = await Lib_AddressManager.getAddress(
    'Proxy__L1CrossDomainMessenger'
  )

  console.log(`Setting messenger address...`)
  await proxy.setStorage(
    ethers.utils.hexZeroPad('0x00', 32),
    ethers.utils.hexZeroPad(l1CrossDomainMessengerAddress, 32)
  )

  console.log(`Confirming that messenger address was correctly set...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(
      await contract.messenger(),
      l1CrossDomainMessengerAddress
    )
  })

  // Now we set the bridge address in the same manner as the messenger address.
  console.log(`Setting l2 bridge address...`)
  await proxy.setStorage(
    ethers.utils.hexZeroPad('0x01', 32),
    ethers.utils.hexZeroPad(predeploys.L2StandardBridge, 32)
  )

  console.log(`Confirming that l2 bridge address was correctly set...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(
      await contract.l2TokenBridge(),
      predeploys.L2StandardBridge
    )
  })

  // Finally we transfer ownership of the proxy to the ovmAddressManagerOwner address.
  console.log(`Setting owner address...`)
  const owner = (hre as any).deployConfig.ovmAddressManagerOwner
  await proxy.setOwner(owner)

  console.log(`Confirming that owner address was correctly set...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(
      await proxy.connect(proxy.signer.provider).callStatic.getOwner({
        from: ethers.constants.AddressZero,
      }),
      owner
    )
  })
}

deployFn.tags = ['L1StandardBridge', 'upgrade']

export default deployFn
