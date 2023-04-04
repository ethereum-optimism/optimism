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
// - OptimistInviterProxy is deployed and points to the correct implementation
// - OptimistAllowlistImpl is deployed
//
// Steps
// 1. Deploy OptimistAllowlistProxy
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

  // Get the up to date deployment of the OptimistAllowlist contract
  const Deployment__OptimistAllowlistImpl = await hre.deployments.get(
    'OptimistAllowlist'
  )

  console.log(`Deploying OptimistAllowlistProxy with ${deployer}`)

  // Deploys the Proxy.sol contract with the `_admin` constructor param set to the ddd (=== deployer).
  const { deploy } = await hre.deployments.deterministic(
    'OptimistAllowlistProxy',
    {
      salt: hre.ethers.utils.solidityKeccak256(
        ['string'],
        ['OptimistAllowlistProxy']
      ),
      contract: 'Proxy',
      from: deployer,
      args: [deployer],
      log: true,
    }
  )

  // Deploy the Proxy contract
  await deploy()

  const Deployment__OptimistAllowlistProxy = await hre.deployments.get(
    'OptimistAllowlistProxy'
  )
  console.log(
    `OptimistAllowlistProxy deployed to ${Deployment__OptimistAllowlistProxy.address}`
  )

  // Deployed Proxy.sol contract
  const Proxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__OptimistAllowlistProxy.address
  )

  // Deployed Proxy.sol contract with the OptimistAllowlist interface
  const OptimistAllowlist = await hre.ethers.getContractAt(
    'OptimistAllowlist',
    Deployment__OptimistAllowlistProxy.address
  )

  // ethers.Signer for the ddd. Should be the current owner of the Proxy.
  const dddSigner = await hre.ethers.provider.getSigner(deployer)

  // intended admin of the Proxy
  const l2ProxyOwnerAddress = deployConfig.l2ProxyOwnerAddress

  // setup the Proxy contract with correct implementation and admin
  await setupProxyContract(Proxy, dddSigner, {
    targetImplAddress: Deployment__OptimistAllowlistImpl.address,
    targetProxyOwnerAddress: l2ProxyOwnerAddress,
  })

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )

  const Deployment__OptimistInviter = await hre.deployments.get(
    'OptimistInviterProxy'
  )

  await assert(
    getAddress(
      await Proxy.connect(ethers.constants.AddressZero).callStatic.admin()
    ) === getAddress(l2ProxyOwnerAddress)
  )

  await assertContractVariable(OptimistAllowlist, 'version', '1.0.0')

  await assertContractVariable(
    OptimistAllowlist,
    'ATTESTATION_STATION',
    Deployment__AttestationStationProxy.address
  )

  await assertContractVariable(
    OptimistAllowlist,
    'ALLOWLIST_ATTESTOR',
    deployConfig.optimistAllowlistAllowlistAttestor
  )

  await assertContractVariable(
    OptimistAllowlist,
    'COINBASE_QUEST_ATTESTOR',
    deployConfig.optimistAllowlistCoinbaseQuestAttestor
  )

  await assertContractVariable(
    OptimistAllowlist,
    'OPTIMIST_INVITER',
    Deployment__OptimistInviter.address
  )
}

deployFn.tags = ['OptimistAllowlistProxy', 'OptimistEnvironment']
deployFn.dependencies = [
  'AttestationStationProxy',
  'OptimistInviterProxy',
  'OptimistAllowlistImpl',
]

export default deployFn
