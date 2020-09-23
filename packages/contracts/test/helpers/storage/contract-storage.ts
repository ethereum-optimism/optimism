/* External Imports */
import bre, { ethers } from '@nomiclabs/buidler'
import { readArtifact } from '@nomiclabs/buidler/internal/artifacts'
import { Contract, BigNumber, ContractFactory } from 'ethers'
import { keccak256 } from 'ethers/lib/utils'
import _ from 'lodash'

/* Internal Imports */
import { remove0x } from '../utils'

const getStorageLayout = async (name: string): Promise<any> => {
  const artifact: any = await readArtifact(bre.config.paths.artifacts, name)
  return artifact.storageLayout
}

export const getModifiableStorageFactory = async (
  name: string
): Promise<ContractFactory> => {
  const contractFactory = await ethers.getContractFactory(name)
  const proxyFactory = await ethers.getContractFactory(
    'Helper_ModifiableStorage'
  )

  const originalDeployFn = contractFactory.deploy.bind(contractFactory)
  contractFactory.deploy = async (...args: any[]): Promise<Contract> => {
    const originalDefinePropertyFn = Object.defineProperty
    Object.defineProperty = (obj: any, pname: string, prop: any): void => {
      if (prop.writable === false) {
        prop.writable = true
      }

      originalDefinePropertyFn(obj, pname, prop)
    }

    const contract: any = await originalDeployFn(...args)
    const proxy = await proxyFactory.deploy(contract.address)

    Object.defineProperty = originalDefinePropertyFn

    contract.address = proxy.address
    contract.resolvedAddress = proxy.address

    contract.__setStorageSlot = proxy.__setStorageSlot.bind(proxy)
    contract.__getStorageSlot = proxy.__getStorageSlot.bind(proxy)

    contract.__setContractStorage = async (obj: any) => {
      await setContractStorage(contract, await getStorageLayout(name), obj)
    }

    contract.__checkContractStorage = async (obj: any) => {
      await checkContractStorage(contract, await getStorageLayout(name), obj)
    }

    return contract
  }

  return contractFactory
}

const flattenObject = (
  obj: Object,
  prefix: string = '',
  res: Object = {}
): Object => {
  if (_.isString(obj) || _.isNumber(obj) || _.isBoolean(obj)) {
    res[prefix] = obj
    return res
  } else if (_.isArray(obj)) {
    for (let i = 0; i < obj.length; i++) {
      const pre = _.isEmpty(prefix) ? `${i}` : `${prefix}.${i}`
      flattenObject(obj[i], pre, res)
    }
    return res
  } else if (_.isPlainObject(obj)) {
    for (const key of Object.keys(obj)) {
      const pre = _.isEmpty(prefix) ? key : `${prefix}.${key}`
      flattenObject(obj[key], pre, res)
    }
    return res
  } else {
    throw new Error('Cannot flatten unsupported object type.')
  }
}

interface InputSlot {
  label: string
  slot: number
}

const getInputSlots = (
  storageLayout: any,
  inputTypeName: string
): InputSlot[] => {
  const inputType = storageLayout.types[inputTypeName]

  if (inputType.encoding === 'mapping') {
    return getInputSlots(storageLayout, inputType.value)
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

interface StorageSlot {
  label: string
  hash: string
  value: string
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

const getStorageSlots = (storageLayout: any, obj: any): StorageSlot[] => {
  const slots: StorageSlot[] = []
  const flat = flattenObject(obj)

  for (const key of Object.keys(flat)) {
    const path = key.split('.')
    const variableLabel = path[0]

    const variableDef = storageLayout.storage.find((vDef: any) => {
      return vDef.label === variableLabel
    })

    if (!variableDef) {
      throw new Error(
        `Could not find a matching variable definition for ${variableLabel}`
      )
    }

    const baseSlot = parseInt(variableDef.slot, 10)
    const baseDepth = (variableDef.type.match(/t_mapping/g) || []).length
    const slotLabel =
      path.length > 1 + baseDepth ? path[path.length - 1] : 'default'

    const inputSlot = getInputSlots(storageLayout, variableDef.type).find(
      (iSlot) => {
        return iSlot.label === slotLabel
      }
    )

    if (!inputSlot) {
      throw new Error(
        `Could not find a matching slot definition for ${slotLabel}`
      )
    }

    let slotHash = toHexString32(baseSlot)
    for (let i = 0; i < baseDepth; i++) {
      slotHash = keccak256(toHexString32(path[i + 1]) + remove0x(slotHash))
    }

    slotHash = toHexString32(BigNumber.from(slotHash).add(inputSlot.slot))

    slots.push({
      label: key,
      hash: slotHash,
      value: toHexString32(flat[key]),
    })
  }

  return slots
}

const setContractStorage = async (
  contract: Contract,
  layout: any,
  obj: any
): Promise<void> => {
  obj = obj || {}

  if (!obj) {
    return
  }

  const slots = getStorageSlots(layout, obj)
  for (const slot of slots) {
    await contract.__setStorageSlot(slot.hash, slot.value)
  }
}

const checkContractStorage = async (
  contract: Contract,
  layout: any,
  obj: any
): Promise<void> => {
  obj = obj || {}

  if (!obj) {
    return
  }

  const slots = getStorageSlots(layout, obj)
  for (const slot of slots) {
    const value = await contract.__getStorageSlot(slot.hash)

    if (value !== slot.value) {
      throw new Error(
        `Resulting state of ${slot.label} (${value}) did not match expected state (${slot.value}).`
      )
    }
  }
}
