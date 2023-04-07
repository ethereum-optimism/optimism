/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { assertContractVariable } from '@eth-optimism/contracts-bedrock/src/deploy-utils'
import { utils } from 'ethers'

import { setupProxyContract } from '../../src/helpers/setupProxyContract'
import type { DeployConfig } from '../../src'

const { getAddress } = utils

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const deployConfig = hre.deployConfig as DeployConfig

  const { deployer } = await hre.getNamedAccounts()
  const ddd = deployConfig.ddd

  if (getAddress(deployer) !== getAddress(ddd)) {
    throw new Error('Must deploy with the ddd')
  }

  const Deployment__OptimistImpl = await hre.deployments.get('Optimist')

  console.log(`Deploying OptimistProxy with ${deployer}`)

  const { deploy } = await hre.deployments.deterministic('OptimistProxy', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['OptimistProxy']),
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
  })

  await deploy()

  const Deployment__OptimistProxy = await hre.deployments.get('OptimistProxy')
  console.log(`OptimistProxy deployed to ${Deployment__OptimistProxy.address}`)

  const Proxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__OptimistProxy.address
  )

  const Optimist = await hre.ethers.getContractAt(
    'Optimist',
    Deployment__OptimistProxy.address
  )

  // ethers.Signer for the ddd. Should be the current owner of the Proxy.
  const dddSigner = await hre.ethers.provider.getSigner(deployer)

  // intended admin of the Proxy
  const l2ProxyOwnerAddress = deployConfig.l2ProxyOwnerAddress

  // Create the calldata for the call to `initialize()`
  const name = deployConfig.optimistName
  const symbol = deployConfig.optimistSymbol
  const initializeCalldata = Optimist.interface.encodeFunctionData(
    'initialize',
    [name, symbol]
  )

  // setup the Proxy contract with correct implementation and admin, and initialize atomically
  await setupProxyContract(Proxy, dddSigner, {
    targetImplAddress: Deployment__OptimistImpl.address,
    targetProxyOwnerAddress: l2ProxyOwnerAddress,
    postUpgradeCallCalldata: initializeCalldata,
  })

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )

  const Deployment__OptimistAllowlistProxy = await hre.deployments.get(
    'OptimistAllowlistProxy'
  )

  await assertContractVariable(Proxy, 'admin', l2ProxyOwnerAddress)
  await assertContractVariable(Optimist, 'name', deployConfig.optimistName)
  await assertContractVariable(Optimist, 'version', '2.0.0')
  await assertContractVariable(Optimist, 'symbol', deployConfig.optimistSymbol)
  await assertContractVariable(
    Optimist,
    'BASE_URI_ATTESTOR',
    deployConfig.optimistBaseUriAttestorAddress
  )

  await assertContractVariable(
    Optimist,
    'OPTIMIST_ALLOWLIST',
    Deployment__OptimistAllowlistProxy.address
  )
  await assertContractVariable(
    Optimist,
    'ATTESTATION_STATION',
    Deployment__AttestationStationProxy.address
  )
}

deployFn.tags = ['OptimistProxy', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStationProxy', 'OptimistImpl']

export default deployFn
