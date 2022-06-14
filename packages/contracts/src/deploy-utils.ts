import { ethers, Contract } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { sleep, awaitCondition, getChainId } from '@eth-optimism/core-utils'
import { HttpNetworkConfig } from 'hardhat/types'

/**
 * @param  {Any} hre Hardhat runtime environment
 * @param  {String} name Contract name from the names object
 * @param  {Any[]} args Constructor arguments
 * @param  {String} contract Name of the solidity contract
 * @param  {String} iface Alternative interface for calling the contract
 * @param  {Function} postDeployAction Called after deployment
 */

export const deployAndVerifyAndThen = async ({
  hre,
  name,
  args,
  contract,
  iface,
  postDeployAction,
}: {
  hre: any
  name: string
  args: any[]
  contract?: string
  iface?: string
  postDeployAction?: (contract: Contract) => Promise<void>
}) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  const result = await deploy(name, {
    contract,
    from: deployer,
    args,
    log: true,
    waitConfirmations: hre.deployConfig.numDeployConfirmations,
  })

  await hre.ethers.provider.waitForTransaction(result.transactionHash)

  if (result.newlyDeployed) {
    if (!(await isHardhatNode(hre))) {
      // Verification sometimes fails, even when the contract is correctly deployed and eventually
      // verified. Possibly due to a race condition. We don't want to halt the whole deployment
      // process just because that happens.
      try {
        console.log('Verifying on Etherscan...')
        await hre.run('verify:verify', {
          address: result.address,
          constructorArguments: args,
        })
        console.log('Successfully verified on Etherscan')
      } catch (error) {
        console.log('Error when verifying bytecode on Etherscan:')
        console.log(error)
      }

      try {
        console.log('Verifying on Sourcify...')
        await hre.run('sourcify')
        console.log('Successfully verified on Sourcify')
      } catch (error) {
        console.log('Error when verifying bytecode on Sourcify:')
        console.log(error)
      }
    }
    if (postDeployAction) {
      const signer = hre.ethers.provider.getSigner(deployer)
      let abi = result.abi
      if (iface !== undefined) {
        const factory = await hre.ethers.getContractFactory(iface)
        abi = factory.interface
      }
      await postDeployAction(
        getAdvancedContract({
          hre,
          contract: new Contract(result.address, abi, signer),
        })
      )
    }
  }
}

// Returns a version of the contract object which modifies all of the input contract's methods to:
// 1. Waits for a confirmed receipt with more than deployConfig.numDeployConfirmations confirmations.
// 2. Include simple resubmission logic, ONLY for Kovan, which appears to drop transactions.
export const getAdvancedContract = (opts: {
  hre: any
  contract: Contract
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
      // We want to use the gas price that has been configured at the beginning of the deployment.
      // However, if the function being triggered is a "constant" (static) function, then we don't
      // want to provide a gas price because we're prone to getting insufficient balance errors.
      let gasPrice = opts.hre.deployConfig.gasPrice || undefined
      if (contract.interface.getFunction(fnName).constant) {
        gasPrice = 0
      }

      const tx = await fn(...args, {
        gasPrice,
      })

      if (typeof tx !== 'object' || typeof tx.wait !== 'function') {
        return tx
      }

      // Special logic for:
      // (1) handling confirmations
      // (2) handling an issue on Kovan specifically where transactions get dropped for no
      //     apparent reason.
      const maxTimeout = 120
      let timeout = 0
      while (true) {
        await sleep(1000)
        const receipt = await contract.provider.getTransactionReceipt(tx.hash)
        if (receipt === null) {
          timeout++
          if (timeout > maxTimeout && opts.hre.network.name === 'kovan') {
            // Special resubmission logic ONLY required on Kovan.
            console.log(
              `WARNING: Exceeded max timeout on transaction. Attempting to submit transaction again...`
            )
            return contract[fnName](...args)
          }
        } else if (
          receipt.confirmations >= opts.hre.deployConfig.numDeployConfirmations
        ) {
          return tx
        }
      }
    }
  }

  return contract
}

export const fundAccount = async (
  hre: any,
  address: string,
  amount: ethers.BigNumber
) => {
  if (!hre.deployConfig.isForkedNetwork) {
    throw new Error('this method can only be used against a forked network')
  }

  console.log(`Funding account ${address}...`)
  await hre.ethers.provider.send('hardhat_setBalance', [
    address,
    amount.toHexString(),
  ])

  console.log(`Waiting for balance to reflect...`)
  await awaitCondition(
    async () => {
      const balance = await hre.ethers.provider.getBalance(address)
      return balance.gte(amount)
    },
    5000,
    100
  )

  console.log(`Account successfully funded.`)
}

export const sendImpersonatedTx = async (opts: {
  hre: any
  contract: ethers.Contract
  fn: string
  from: string
  gas: string
  args: any[]
}) => {
  if (!opts.hre.deployConfig.isForkedNetwork) {
    throw new Error('this method can only be used against a forked network')
  }

  console.log(`Impersonating account ${opts.from}...`)
  await opts.hre.ethers.provider.send('hardhat_impersonateAccount', [opts.from])

  console.log(`Funding account ${opts.from}...`)
  await fundAccount(opts.hre, opts.from, BIG_BALANCE)

  console.log(`Sending impersonated transaction...`)
  const tx = await opts.contract.populateTransaction[opts.fn](...opts.args)
  const provider = new opts.hre.ethers.providers.JsonRpcProvider(
    (opts.hre.network.config as HttpNetworkConfig).url
  )
  await provider.send('eth_sendTransaction', [
    {
      ...tx,
      from: opts.from,
      gas: opts.gas,
    },
  ])

  console.log(`Stopping impersonation of account ${opts.from}...`)
  await opts.hre.ethers.provider.send('hardhat_stopImpersonatingAccount', [
    opts.from,
  ])
}

export const getContractFromArtifact = async (
  hre: any,
  name: string,
  options: {
    iface?: string
    signerOrProvider?: Signer | Provider | string
  } = {}
): Promise<ethers.Contract> => {
  const artifact = await hre.deployments.get(name)
  await hre.ethers.provider.waitForTransaction(artifact.receipt.transactionHash)

  // Get the deployed contract's interface.
  let iface = new hre.ethers.utils.Interface(artifact.abi)
  // Override with optional iface name if requested.
  if (options.iface) {
    const factory = await hre.ethers.getContractFactory(options.iface)
    iface = factory.interface
  }

  let signerOrProvider: Signer | Provider = hre.ethers.provider
  if (options.signerOrProvider) {
    if (typeof options.signerOrProvider === 'string') {
      signerOrProvider = hre.ethers.provider.getSigner(options.signerOrProvider)
    } else {
      signerOrProvider = options.signerOrProvider
    }
  }

  return getAdvancedContract({
    hre,
    contract: new hre.ethers.Contract(
      artifact.address,
      iface,
      signerOrProvider
    ),
  })
}

export const isHardhatNode = async (hre) => {
  return (await getChainId(hre.ethers.provider)) === 31337
}

// Large balance to fund accounts with.
export const BIG_BALANCE = ethers.BigNumber.from(`0xFFFFFFFFFFFFFFFFFFFF`)
