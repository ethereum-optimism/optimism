import assert from 'assert'

import { ethers, Contract } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { sleep } from '@eth-optimism/core-utils'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { Deployment, DeployResult } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

/**
 * Wrapper around hardhat-deploy with some extra features.
 *
 * @param opts Options for the deployment.
 * @param opts.hre HardhatRuntimeEnvironment.
 * @param opts.contract Name of the contract to deploy.
 * @param opts.name Name to use for the deployment file.
 * @param opts.iface Interface to use for the returned contract.
 * @param opts.args Arguments to pass to the contract constructor.
 * @param opts.postDeployAction Action to perform after the contract is deployed.
 * @returns Deployed contract object.
 */
export const deploy = async ({
  hre,
  name,
  iface,
  args,
  contract,
  postDeployAction,
}: {
  hre: HardhatRuntimeEnvironment
  name: string
  args: any[]
  contract?: string
  iface?: string
  postDeployAction?: (contract: Contract) => Promise<void>
}) => {
  const { deployer } = await hre.getNamedAccounts()

  // Hardhat deploy will usually do this check for us, but currently doesn't also consider
  // external deployments when doing this check. By doing the check ourselves, we also get to
  // consider external deployments. If we already have the deployment, return early.
  let result: Deployment | DeployResult = await hre.deployments.getOrNull(name)
  if (result) {
    console.log(`skipping ${name}, using existing at ${result.address}`)
  } else {
    result = await hre.deployments.deploy(name, {
      contract,
      from: deployer,
      args,
      log: true,
      waitConfirmations: hre.deployConfig.numDeployConfirmations,
    })
  }

  // Always wait for the transaction to be mined, just in case.
  await hre.ethers.provider.waitForTransaction(result.transactionHash)

  // Create the contract object to return.
  const created = asAdvancedContract({
    confirmations: hre.deployConfig.numDeployConfirmations,
    gasPrice: hre.deployConfig.gasPrice,
    contract: new Contract(
      result.address,
      iface !== undefined
        ? (await hre.ethers.getContractFactory(iface)).interface
        : result.abi,
      hre.ethers.provider.getSigner(deployer)
    ),
  })

  // Run post-deploy actions if necessary.
  if ((result as DeployResult).newlyDeployed) {
    if (postDeployAction) {
      await postDeployAction(created)
    }
  }

  return created
}

/**
 * Returns a version of the contract object which modifies all of the input contract's methods to
 * automatically await transaction receipts and confirmations. Will also throw if we timeout while
 * waiting for a transaction to be included in a block.
 *
 * @param opts Options for the contract.
 * @param opts.hre HardhatRuntimeEnvironment.
 * @param opts.contract Contract to wrap.
 * @returns Wrapped contract object.
 */
export const asAdvancedContract = (opts: {
  contract: Contract
  confirmations?: number
  gasPrice?: number
}): Contract => {
  // Temporarily override Object.defineProperty to bypass ether's object protection.
  const def = Object.defineProperty
  Object.defineProperty = (obj, propName, prop) => {
    prop.writable = true
    return def(obj, propName, prop)
  }

  const contract = new Contract(
    opts.contract.address,
    opts.contract.interface,
    opts.contract.signer || opts.contract.provider
  )

  // Now reset Object.defineProperty
  Object.defineProperty = def

  // Override each function call to also `.wait()` so as to simplify the deploy scripts' syntax.
  for (const fnName of Object.keys(contract.functions)) {
    const fn = contract[fnName].bind(contract)
    ;(contract as any)[fnName] = async (...args: any) => {
      // We want to use the configured gas price but we need to set the gas price to zero if we're
      // triggering a static function.
      let gasPrice = opts.gasPrice
      if (contract.interface.getFunction(fnName).constant) {
        gasPrice = 0
      }

      // Now actually trigger the transaction (or call).
      const tx = await fn(...args, {
        gasPrice,
      })

      // Meant for static calls, we don't need to wait for anything, we get the result right away.
      if (typeof tx !== 'object' || typeof tx.wait !== 'function') {
        return tx
      }

      // Wait for the transaction to be included in a block and wait for the specified number of
      // deployment confirmations.
      const maxTimeout = 120
      let timeout = 0
      while (true) {
        await sleep(1000)
        const receipt = await contract.provider.getTransactionReceipt(tx.hash)
        if (receipt === null && timeout++ > maxTimeout) {
          throw new Error('timeout exceeded waiting for txn to be mined')
        } else if (receipt.confirmations >= (opts.confirmations || 0)) {
          return tx
        }
      }
    }
  }

  return contract
}

/**
 * Creates a contract object from a deployed artifact.
 *
 * @param hre HardhatRuntimeEnvironment.
 * @param name Name of the deployed contract to get an object for.
 * @param opts Options for the contract.
 * @param opts.iface Optional interface to use for the contract object.
 * @param opts.signerOrProvider Optional signer or provider to use for the contract object.
 * @returns Contract object.
 */
export const getContractFromArtifact = async (
  hre: HardhatRuntimeEnvironment,
  name: string,
  opts: {
    iface?: string
    signerOrProvider?: Signer | Provider | string
  } = {}
): Promise<ethers.Contract> => {
  const artifact = await hre.deployments.get(name)
  await hre.ethers.provider.waitForTransaction(artifact.receipt.transactionHash)

  // Get the deployed contract's interface.
  let iface = new hre.ethers.utils.Interface(artifact.abi)
  // Override with optional iface name if requested.
  if (opts.iface) {
    const factory = await hre.ethers.getContractFactory(opts.iface)
    iface = factory.interface
  }

  let signerOrProvider: Signer | Provider = hre.ethers.provider
  if (opts.signerOrProvider) {
    if (typeof opts.signerOrProvider === 'string') {
      signerOrProvider = hre.ethers.provider.getSigner(opts.signerOrProvider)
    } else {
      signerOrProvider = opts.signerOrProvider
    }
  }

  return asAdvancedContract({
    confirmations: hre.deployConfig.numDeployConfirmations,
    gasPrice: hre.deployConfig.gasPrice,
    contract: new hre.ethers.Contract(
      artifact.address,
      iface,
      signerOrProvider
    ),
  })
}

/**
 * Gets multiple contract objects from their respective deployed artifacts.
 *
 * @param hre HardhatRuntimeEnvironment.
 * @param configs Array of contract names and options.
 * @returns Array of contract objects.
 */
export const getContractsFromArtifacts = async (
  hre: HardhatRuntimeEnvironment,
  configs: Array<{
    name: string
    iface?: string
    signerOrProvider?: Signer | Provider | string
  }>
): Promise<ethers.Contract[]> => {
  const contracts = []
  for (const config of configs) {
    contracts.push(await getContractFromArtifact(hre, config.name, config))
  }
  return contracts
}

/**
 * Helper function for asserting that a contract variable is set to the expected value.
 *
 * @param contract Contract object to query.
 * @param variable Name of the variable to query.
 * @param expected Expected value of the variable.
 */
export const assertContractVariable = async (
  contract: ethers.Contract,
  variable: string,
  expected: any
) => {
  // Need to make a copy that doesn't have a signer or we get the error that contracts with
  // signers cannot override the from address.
  const temp = new ethers.Contract(
    contract.address,
    contract.interface,
    contract.provider
  )

  const actual = await temp.callStatic[variable]({
    from: ethers.constants.AddressZero,
  })

  if (ethers.utils.isAddress(expected)) {
    assert(
      actual.toLowerCase() === expected.toLowerCase(),
      `[FATAL] ${variable} is ${actual} but should be ${expected}`
    )
    return
  }

  assert(
    actual === expected || (actual.eq && actual.eq(expected)),
    `[FATAL] ${variable} is ${actual} but should be ${expected}`
  )
}

/**
 * Returns the address for a given deployed contract by name.
 *
 * @param hre HardhatRuntimeEnvironment.
 * @param name Name of the deployed contract.
 * @returns Address of the deployed contract.
 */
export const getDeploymentAddress = async (
  hre: HardhatRuntimeEnvironment,
  name: string
): Promise<string> => {
  const deployment = await hre.deployments.get(name)
  return deployment.address
}
