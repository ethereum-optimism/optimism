/* External Imports */
import hre from 'hardhat'
import { Artifacts } from 'hardhat/internal/artifacts'
import { ethers } from 'ethers'
import { remove0x } from '@eth-optimism/core-utils'
import _ from 'lodash'

/* Internal Imports */
import { toHexString32 } from '../utils'

interface InputSlot {
  label: string
  slot: number
}

interface StorageSlot {
  label: string
  hash: string
  value: string
}

/**
 * Reads the storage layout of a contract.
 * @param name Name of the contract to get a storage layout for.
 * @return Storage layout for the given contract name.
 */
export const getStorageLayout = async (name: string): Promise<any> => {
  const artifacts = new Artifacts(hre.config.paths.artifacts)
  const { sourceName, contractName } = artifacts.readArtifactSync(name)
  const buildInfo = await hre.artifacts.getBuildInfo(
    `${sourceName}:${contractName}`
  )
  const output = buildInfo.output.contracts[sourceName][contractName]

  if (!('storageLayout' in output)) {
    throw new Error(
      `Storage layout for ${name} not found. Did you forget to set the storage layout compiler option in your hardhat config? Read more: https://github.com/ethereum-optimism/smock#note-on-using-smoddit`
    )
  }

  return (output as any).storageLayout
}

/**
 * Converts storage into a list of storage slots.
 * @param storageLayout Contract storage layout.
 * @param obj Storage object to convert.
 * @returns List of storage slots.
 */
export const getStorageSlots = (
  storageLayout: any,
  obj: any
): StorageSlot[] => {
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
      slotHash = ethers.utils.keccak256(
        toHexString32(path[i + 1]) + remove0x(slotHash)
      )
    }

    slotHash = toHexString32(
      ethers.BigNumber.from(slotHash).add(inputSlot.slot)
    )

    slots.push({
      label: key,
      hash: slotHash,
      value: toHexString32(flat[key]),
    })
  }

  return slots
}

/**
 * Flattens an object.
 * @param obj Object to flatten.
 * @param prefix Current object prefix (used recursively).
 * @param res Current result (used recursively).
 * @returns Flattened object.
 */
const flattenObject = (
  obj: any,
  prefix: string = '',
  res: any = {}
): Object => {
  if (ethers.BigNumber.isBigNumber(obj)) {
    res[prefix] = obj.toNumber()
    return res
  } else if (_.isString(obj) || _.isNumber(obj) || _.isBoolean(obj)) {
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

/**
 * Gets the slot positions for a provided variable type.
 * @param storageLayout Contract's storage layout.
 * @param inputTypeName Variable type name.
 * @returns Slot positions.
 */
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
    throw new Error(`Encoding type not supported: ${inputType.encoding}`)
  }
}
