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
import { setupProxyContract } from '../../src/helpers/setupProxyContract'

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
    `OptimistInviterProxy deployed to ${Deployment__OptimistInviterProxy.address}`
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

  const name = deployConfig.optimistInviterName
  // Create the calldata for the call to `initialize()`
  const initializeCalldata = OptimistInviter.interface.encodeFunctionData(
    'initialize',
    [name]
  )

  // ethers.Signer for the ddd. Should be the current owner of the Proxy.
  const dddSigner = await hre.ethers.provider.getSigner(deployer)

  // intended admin of the Proxy
  const l2ProxyOwnerAddress = deployConfig.l2ProxyOwnerAddress

  // setup the Proxy contract with correct implementation and admin
  await setupProxyContract(Proxy, dddSigner, {
    targetImplAddress: Deployment__OptimistInviterImpl.address,
    targetProxyOwnerAddress: l2ProxyOwnerAddress,
    postUpgradeCallCalldata: initializeCalldata,
  })

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
deployFn.dependencies = ['AttestationStationProxy', 'OptimistInviterImpl']

export default deployFn
