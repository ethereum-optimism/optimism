/* External Imports */
import hre from 'hardhat'
import { fromHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import { ModifiableContract, ModifiableContractFactory } from './types'
import { getStorageLayout, getStorageSlots } from './storage'
import { toHexString32 } from '../utils'
import { findBaseHardhatProvider, toFancyAddress } from '../common'

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
  const vm: any = (provider as any)._node._vm
  const pStateManager = vm.pStateManager || vm.stateManager

  const layout = await getStorageLayout(name)
  const factory = (await (hre as any).ethers.getContractFactory(
    name,
    signer
  )) as ModifiableContractFactory

  const originalDeployFn = factory.deploy.bind(factory)
  factory.deploy = async (...args: any[]): Promise<ModifiableContract> => {
    const contract: ModifiableContract = await originalDeployFn(...args)
    contract._smodded = {}

    const put = async (storage: any) => {
      if (!storage) {
        return
      }

      const slots = getStorageSlots(layout, storage)
      for (const slot of slots) {
        await pStateManager.putContractStorage(
          toFancyAddress(contract.address),
          fromHexString(slot.hash.toLowerCase()),
          fromHexString(slot.value)
        )
      }
    }

    const check = async (storage: any) => {
      if (!storage) {
        return true
      }

      const slots = getStorageSlots(layout, storage)
      for (const slot of slots) {
        if (
          toHexString32(
            await pStateManager.getContractStorage(
              toFancyAddress(contract.address),
              fromHexString(slot.hash.toLowerCase())
            )
          ) !== slot.value
        ) {
          return false
        }
      }

      return true
    }

    contract.smodify = {
      put,
      check,
    }

    return contract
  }

  return factory
}
