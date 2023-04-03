/* Imports: External */
import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { assertContractVariable } from '@eth-optimism/contracts-bedrock/src/deploy-utils'
import { ethers, utils } from 'ethers'

import type { DeployConfig } from '../../src'

const { getAddress } = utils

// Required conditions before deploying - Specified in `deployFn.dependencies`
// - AttestationStationProxy is deployed and points to the correct implementation
// - OptimistInviterImpl is deployed
//
// Steps
// 1. Deploy OptimistInviterProxy
// 2. Point the newly deployed proxy to the implementation, if it hasn't been done already
// 3. Update the admin of the proxy to the l2ProxyOwnerAddress, if it hasn't been done already
// 4. Basic sanity checks for contract variables

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const deployConfig = hre.deployConfig as DeployConfig

  // Deployer should be set in hardhat.config.ts
  const { deployer } = await hre.getNamedAccounts()

  // We want the ability to deploy to a deterministic address, so we need the init bytecode to be
  // consistent across deployments. The ddd will quickly transfer the ownership of the Proxy to a
  // multisig after deployment.
  //
  // We need a consistent ddd, since the Proxy takes a `_admin` constructor argument, which
  // affects the init bytecode and hence deployed address.
  const ddd = deployConfig.ddd

  if (getAddress(deployer) !== getAddress(ddd)) {
    // Not a hard requirement. We can deploy with any account and just set the `_admin` to the
    // ddd, but requiring that the deployer is the same as the ddd minimizes number of hot wallets
    // we need to keep track of during deployment.
    throw new Error('Must deploy with the ddd')
  }

  // Get the up to date deployment of the OptimistInviter contract
  const Deployment__OptimistInviterImpl = await hre.deployments.get(
    'OptimistInviter'
  )

  console.log(`Deploying OptimistInviterProxy with ${deployer}`)

  // Deploys the Proxy.sol contract with the `_admin` constructor param set to the ddd (=== deployer).
  const { deploy } = await hre.deployments.deterministic(
    'OptimistInviterProxy',
    {
      salt: hre.ethers.utils.solidityKeccak256(
        ['string'],
        ['OptimistInviterProxy']
      ),
      contract: 'Proxy',
      from: deployer,
      args: [deployer],
      log: true,
    }
  )

  // Deploy the Proxy contract
  await deploy()

  const Deployment__OptimistInviterProxy = await hre.deployments.get(
    'OptimistInviterProxy'
  )
  console.log(
    `OptimistProxy deployed to ${Deployment__OptimistInviterProxy.address}`
  )

  // Deployed Proxy.sol contract
  const Proxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__OptimistInviterProxy.address
  )

  // Deployed Proxy.sol contract with the OptimistInviter interface
  const OptimistInviter = await hre.ethers.getContractAt(
    'OptimistInviter',
    Deployment__OptimistInviterProxy.address
  )

  // Gets the current implementation address the proxy is pointing to.
  // callStatic is used since the `Proxy.implementation()` is not a view function and ethers will
  // try to make a transaction if we don't use callStatic. Using the zero address as `from` lets us
  // call functions on the proxy and not trigger the delegatecall. See Proxy.sol proxyCallIfNotAdmin
  // modifier for more details.
  const implementation = await Proxy.connect(
    ethers.constants.AddressZero
  ).callStatic.implementation()
  console.log(`implementation set to ${implementation}`)

  if (
    getAddress(implementation) !==
    getAddress(Deployment__OptimistInviterImpl.address)
  ) {
    // If the proxy isn't pointing to the correct implementation, we need to set it to the correct
    // one, then call initialize() in the proxy's context.
    console.log(
      'implementation not set to OptimistInviter implementation contract'
    )
    console.log(
      `Setting implementation to ${Deployment__OptimistInviterImpl.address}`
    )

    const name = deployConfig.optimistInviterName

    // Create the calldata for the call to `initialize()`
    const calldata = OptimistInviter.interface.encodeFunctionData(
      'initialize',
      [name]
    )

    // ethers.Signer for the ddd
    const dddSigner = await hre.ethers.provider.getSigner(deployer)

    // Point the proxy to the deployed OptimistInviter implementation contract,
    // and call `initialize()` in the proxy's context
    const tx = await Proxy.connect(dddSigner).upgradeToAndCall(
      Deployment__OptimistInviterImpl.address,
      calldata
    )
    const receipt = await tx.wait()
    console.log(`implementation set in ${receipt.transactionHash}`)
  } else {
    console.log(
      'implementation already set to OptimistInviter implementation contract'
    )
  }

  const l2ProxyOwnerAddress = deployConfig.l2ProxyOwnerAddress

  // Get the current proxy admin address
  const admin = await Proxy.connect(
    ethers.constants.AddressZero
  ).callStatic.admin()

  console.log(`admin currently set to ${admin}`)

  if (getAddress(admin) !== getAddress(l2ProxyOwnerAddress)) {
    // If the proxy admin isn't the l2ProxyOwnerAddress, we need to update it
    // We're assuming that the proxy admin is the ddd right now.
    console.log('admin is not set to the l2ProxyOwnerAddress')
    console.log(`Setting admin to ${l2ProxyOwnerAddress}`)

    // ethers.Signer for the ddd
    const dddSigner = await hre.ethers.provider.getSigner(deployer)

    // change admin to the l2ProxyOwnerAddress
    const tx = await Proxy.connect(dddSigner).changeAdmin(l2ProxyOwnerAddress)
    const receipt = await tx.wait()
    console.log(`admin set in ${receipt.transactionHash}`)
  } else {
    console.log('admin already set to proxy owner address')
  }

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStationProxy'
  )

  await assert(
    getAddress(
      await Proxy.connect(ethers.constants.AddressZero).callStatic.admin()
    ) === getAddress(l2ProxyOwnerAddress)
  )

  await assertContractVariable(OptimistInviter, 'version', '1.0.0')

  await assertContractVariable(
    OptimistInviter,
    'INVITE_GRANTER',
    deployConfig.optimistInviterInviteGranter
  )

  await assertContractVariable(
    OptimistInviter,
    'ATTESTATION_STATION',
    Deployment__AttestationStation.address
  )

  await assertContractVariable(OptimistInviter, 'EIP712_VERSION', '1.0.0')
}

deployFn.tags = ['OptimistInviterProxy', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStationProxy', 'OptimistInviter']

export default deployFn
