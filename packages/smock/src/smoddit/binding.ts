/* External Imports */
import { toHexString, fromHexString } from '@eth-optimism/core-utils'
import { HardhatNetworkProvider } from 'hardhat/internal/hardhat-network/provider/provider'

/* Internal Imports */
import { ModifiableContract } from './types'

/**
 * Checks to see if smoddit has been initialized already.
 * @param provider Base hardhat network provider to check.
 * @return Whether or not the provider has already been modified to support smoddit.
 */
const isSmodInitialized = (provider: HardhatNetworkProvider): boolean => {
  return (provider as any)._node._vm._smod !== undefined
}

/**
 * Initializes smodding functionality.
 * @param provider Base hardhat network provider to modify.
 */
const initializeSmod = (provider: HardhatNetworkProvider): void => {
  if (isSmodInitialized(provider)) {
    return
  }

  // Will need to reference these things.
  const node = (provider as any)._node
  const vm = node._vm
  const pStateManager = vm.pStateManager

  vm._smod = {
    contracts: {},
  }

  const originalGetStorageFn = pStateManager.getContractStorage.bind(
    pStateManager
  )
  pStateManager.getContractStorage = async (
    addressBuf: Buffer,
    keyBuf: Buffer
  ): Promise<Buffer> => {
    const originalReturnValue = await originalGetStorageFn(addressBuf, keyBuf)

    const address = toHexString(addressBuf).toLowerCase()
    const key = toHexString(keyBuf).toLowerCase()

    if (!(address in vm._smod.contracts)) {
      return originalReturnValue
    }

    const contract: ModifiableContract = vm._smod.contracts[address]
    if (!(key in contract._smodded)) {
      return originalReturnValue
    }

    return fromHexString(contract._smodded[key])
  }

  const originalPutStorageFn = pStateManager.putContractStorage.bind(
    pStateManager
  )
  pStateManager.putContractStorage = async (
    addressBuf: Buffer,
    keyBuf: Buffer,
    valBuf: Buffer
  ): Promise<void> => {
    await originalPutStorageFn(addressBuf, keyBuf, valBuf)

    const address = toHexString(addressBuf).toLowerCase()
    const key = toHexString(keyBuf).toLowerCase()

    if (!(address in vm._smod.contracts)) {
      return
    }

    const contract: ModifiableContract = vm._smod.contracts[address]
    if (!(key in contract._smodded)) {
      return
    }

    delete contract._smodded[key]
  }
}

/**
 * Binds the smodded contract to the VM.
 * @param contract Contract to bind.
 */
export const bindSmod = (
  contract: ModifiableContract,
  provider: HardhatNetworkProvider
): void => {
  if (!isSmodInitialized(provider)) {
    initializeSmod(provider)
  }

  const vm = (provider as any)._node._vm

  // Add mock to our list of mocks currently attached to the VM.
  vm._smod.contracts[contract.address.toLowerCase()] = contract
}
