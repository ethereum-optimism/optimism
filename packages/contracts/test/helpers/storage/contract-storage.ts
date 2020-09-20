import bre, { ethers } from '@nomiclabs/buidler'
import { Contract, BigNumber, ContractFactory } from 'ethers'
import { keccak256, defaultAbiCoder } from 'ethers/lib/utils'

import { remove0x } from '../byte-utils'
import { readArtifact } from '@nomiclabs/buidler/internal/artifacts'

const getFlattenedKeys = (depth: number, value: any): string[] => {
  if (depth === 0) {
    return []
  }

  let keys = Object.keys(value)
  if (depth > 1) {
    keys = keys.concat(getFlattenedKeys(depth - 1, Object.values(value)[0]))
  }

  return keys
}

const toHexString32 = (
  value: string | number | BigNumber | boolean
): string => {
  if (typeof value === 'string') {
    return '0x' + remove0x(value).padStart(64, '0').toLowerCase()
  } else if (typeof value === 'boolean') {
    return toHexString32(value ? 1 : 0)
  } else {
    return toHexString32(BigNumber.from(value).toHexString())
  }
}

const getFlattenedValues = (depth: number, value: any): any[] => {
  if (depth > 0) {
    return getFlattenedValues(depth - 1, Object.values(value)[0])
  }

  if (typeof value === 'object' && value !== null) {
    return Object.keys(value).map((key) => {
      return {
        label: key,
        value: toHexString32(value[key]),
      }
    })
  } else {
    return [
      {
        label: 'default',
        value: toHexString32(value),
      },
    ]
  }
}

const getStorageSlotHash = (
  slot: number,
  depth: number,
  value: any
): string => {
  let keys = []
  if (typeof value === 'object' && value !== null) {
    keys = getFlattenedKeys(depth, value)
  }

  if (keys.length === 0) {
    return defaultAbiCoder.encode(['uint256'], [slot])
  } else {
    let slotHash = toHexString32(slot)
    for (const key of keys) {
      slotHash = keccak256(toHexString32(key) + remove0x(slotHash))
    }
    return slotHash
  }
}

const parseInputSlots = (layout: any, inputTypeName: string): any[] => {
  const inputType = layout.types[inputTypeName]

  if (inputType.encoding === 'mapping') {
    return parseInputSlots(layout, inputType.value)
  } else if (inputType.encoding === 'inplace') {
    if (inputType.members) {
      return inputType.members.map((member: any) => {
        return {
          label: member.label,
          slot: member.slot,
        }
      })
    } else {
      return [
        {
          label: 'default',
          slot: 0,
        },
      ]
    }
  } else {
    throw new Error('Encoding type not supported.')
  }
}

export const getModifiableStorageFactory = async (
  name: string
): Promise<ContractFactory> => {
  const contractFactory = await ethers.getContractFactory(name)
  const proxyFactory = await ethers.getContractFactory(
    'Helper_ModifiableStorage'
  )

  const originalDeploy = contractFactory.deploy.bind(contractFactory)
  contractFactory.deploy = async (...args: any[]): Promise<Contract> => {
    const originalDefinePropertyFn = Object.defineProperty
    Object.defineProperty = (
      object: any,
      propName: string,
      props: any
    ): void => {
      if (props.writable === false) {
        props.writable = true
      }

      originalDefinePropertyFn(object, propName, props)
    }

    const contract = await originalDeploy(...args)
    const proxy = await proxyFactory.deploy(contract.address)
    ;(contract as any).address = proxy.address
    ;(contract as any).resolvedAddress = proxy.address
    ;(contract as any).__setStorageSlot = proxy.__setStorageSlot.bind(proxy)
    ;(contract as any).__getStorageSlot = proxy.__getStorageSlot.bind(proxy)
    ;(contract as any).__setContractStorage = async (value: any) => {
      await setContractStorage(
        contract,
        ((await readArtifact(bre.config.paths.artifacts, name)) as any)
          .storageLayout,
        value
      )
    }
    ;(contract as any).__checkContractStorage = async (value: any) => {
      await checkContractStorage(
        contract,
        ((await readArtifact(bre.config.paths.artifacts, name)) as any)
          .storageLayout,
        value
      )
    }

    Object.defineProperty = originalDefinePropertyFn
    return contract
  }

  return contractFactory
}

export const setContractStorage = async (
  contract: Contract,
  layout: any,
  storage: any
): Promise<void> => {
  storage = storage || {}

  for (const [key, value] of Object.entries(storage)) {
    const layoutMap = layout.storage.find((lmap: any) => {
      return lmap.label === key
    })
    const inputSlots = parseInputSlots(layout, layoutMap.type)

    const slot = parseInt(layoutMap.slot, 10)
    const depth = (layoutMap.type.match(/t_mapping/g) || []).length

    if (typeof value !== 'object') {
      const slotHash = getStorageSlotHash(slot, depth, value)
      await contract.__setStorageSlot(slotHash, toHexString32(value as string))
    } else {
      for (const [subKey, subValue] of Object.entries(value)) {
        const baseSlotHash = getStorageSlotHash(slot, depth, {
          [subKey]: subValue,
        })
        const slotValues = getFlattenedValues(depth, {
          [subKey]: subValue,
        })

        for (const slotValue of slotValues) {
          const slotIndex = inputSlots.find((inputSlot) => {
            return inputSlot.label === slotValue.label
          }).slot
          const slotHash = toHexString32(
            BigNumber.from(baseSlotHash).add(slotIndex)
          )

          await contract.__setStorageSlot(slotHash, slotValue.value)
        }
      }
    }
  }
}

export const checkContractStorage = async (
  contract: Contract,
  layout: any,
  storage: any
): Promise<void> => {
  storage = storage || {}

  for (const [key, value] of Object.entries(storage)) {
    const layoutMap = layout.storage.find((lmap: any) => {
      return lmap.label === key
    })
    const inputSlots = parseInputSlots(layout, layoutMap.type)

    const slot = parseInt(layoutMap.slot, 10)
    const depth = (layoutMap.type.match(/t_mapping/g) || []).length

    if (typeof value !== 'object') {
      const slotHash = getStorageSlotHash(slot, depth, value)
      const retSlotValue = await contract.__getStorageSlot(slotHash)

      if (retSlotValue !== toHexString32(value as string)) {
        throw new Error(
          `Resulting state of ${key} (${retSlotValue}) did not match expected state (${toHexString32(
            value as string
          )})`
        )
      }
    } else {
      for (const [subKey, subValue] of Object.entries(value)) {
        const baseSlotHash = getStorageSlotHash(slot, depth, {
          [subKey]: subValue,
        })
        const slotValues = getFlattenedValues(depth, {
          [subKey]: subValue,
        })

        for (const slotValue of slotValues) {
          const slotIndex = inputSlots.find((inputSlot) => {
            return inputSlot.label === slotValue.label
          }).slot
          const slotHash = toHexString32(
            BigNumber.from(baseSlotHash).add(slotIndex)
          )

          const retSlotValue = await contract.__getStorageSlot(slotHash)

          if (retSlotValue !== slotValue.value) {
            throw new Error(
              `Resulting state of ${slotValue.label} (${retSlotValue}) did not match expected state (${slotValue.value}).`
            )
          }
        }
      }
    }
  }
}
