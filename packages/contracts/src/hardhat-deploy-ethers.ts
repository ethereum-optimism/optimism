/* Imports: External */
import { Contract } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { sleep, hexStringEquals } from '@eth-optimism/core-utils'

export const waitUntilTrue = async (
  check: () => Promise<boolean>,
  opts: {
    retries?: number
    delay?: number
  } = {}
) => {
  opts.retries = opts.retries || 100
  opts.delay = opts.delay || 5000

  let retries = 0
  while (!(await check())) {
    if (retries > opts.retries) {
      throw new Error(`check failed after ${opts.retries} attempts`)
    }
    retries++
    await sleep(opts.delay)
  }
}

export const registerAddress = async ({
  hre,
  name,
  address,
}): Promise<void> => {
  // TODO: Cache these 2 across calls?
  const { deployer } = await hre.getNamedAccounts()
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager',
    {
      signerOrProvider: deployer,
    }
  )

  const currentAddress = await Lib_AddressManager.getAddress(name)
  if (address === currentAddress) {
    console.log(
      `✓ Not registering address for ${name} because it's already been correctly registered`
    )
    return
  }

  console.log(`Registering address for ${name} to ${address}...`)
  await Lib_AddressManager.setAddress(name, address)

  console.log(`Waiting for registration to reflect on-chain...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(await Lib_AddressManager.getAddress(name), address)
  })

  console.log(`✓ Registered address for ${name}`)
}

export const deployAndRegister = async ({
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

    await registerAddress({
      hre,
      name,
      address: result.address,
    })
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
      const tx = await fn(...args, {
        gasPrice: opts.hre.deployConfig.gasprice || undefined,
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

export const getDeployedContract = async (
  hre: any,
  name: string,
  options: {
    iface?: string
    signerOrProvider?: Signer | Provider | string
  } = {}
): Promise<Contract> => {
  const deployed = await hre.deployments.get(name)

  await hre.ethers.provider.waitForTransaction(deployed.receipt.transactionHash)

  // Get the correct interface.
  let iface = new hre.ethers.utils.Interface(deployed.abi)
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
    contract: new Contract(deployed.address, iface, signerOrProvider),
  })
}
