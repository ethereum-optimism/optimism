/* External Imports */
import hre from 'hardhat'
import { fromHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import { ModifiableContract, ModifiableContractFactory, Smodify } from './types'
import { getStorageLayout, getStorageSlots } from './storage'
import { bindSmod } from './binding'
import { toHexString32 } from '../utils'
import { findBaseHardhatProvider } from '../common'

/**
 * Creates a modifiable contract factory.
 * @param name Name of the contract to smoddify.
 * @param signer Optional signer to attach to the factory.
 * @returns Smoddified contract factory.
 */
export const smoddit = async (
  name: string,
  signer?: any
): Promise<ModifiableContractFactory> => {
  // Find the provider object. See comments for `findBaseHardhatProvider`
  const provider = findBaseHardhatProvider(hre)

  // Sometimes the VM hasn't been initialized by the time we get here, depending on what the user
  // is doing with hardhat (e.g., sending a transaction before calling this function will
  // initialize the vm). Initialize it here if it hasn't been already.
  if ((provider as any)._node === undefined) {
    await (provider as any)._init()
  }

  // Pull out a reference to the vm's state manager.
  const pStateManager = (provider as any)._node._vm.pStateManager

  const layout = await getStorageLayout(name)
  const factory = (await hre.ethers.getContractFactory(
    name,
    signer
  )) as ModifiableContractFactory

  const originalDeployFn = factory.deploy.bind(factory)
  factory.deploy = async (...args: any[]): Promise<ModifiableContract> => {
    const contract: ModifiableContract = await originalDeployFn(...args)
    contract._smodded = {}

    const put = (storage: any) => {
      if (!storage) {
        return
      }

      const slots = getStorageSlots(layout, storage)
      for (const slot of slots) {
        contract._smodded[slot.hash.toLowerCase()] = slot.value
      }
    }

    const reset = () => {
      contract._smodded = {}
    }

    const set = (storage: any) => {
      contract.smodify.reset()
      contract.smodify.put(storage)
    }

    const check = async (storage: any) => {
      if (!storage) {
        return true
      }

      const slots = getStorageSlots(layout, storage)
      return slots.every(async (slot) => {
        return (
          toHexString32(
            await pStateManager.getContractStorage(
              fromHexString(contract.address),
              fromHexString(slot.hash.toLowerCase())
            )
          ) === slot.value
        )
      })
    }

    contract.smodify = {
      put,
      reset,
      set,
      check,
    }

    bindSmod(contract, provider)
    return contract
  }

  return factory
}
