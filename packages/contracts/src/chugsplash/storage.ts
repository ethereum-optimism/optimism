/* External Imports */
import { remove0x } from '@eth-optimism/core-utils'
import { BigNumber, ethers } from 'ethers'

// Represents the JSON objects outputted by the Solidity compiler that describe the structure of
// state within the contract. See
// https://docs.soliditylang.org/en/v0.8.3/internals/layout_in_storage.html for more information.
interface SolidityStorageObj {
  astId: number
  contract: string
  label: string
  offset: number
  slot: number
  type: string
}

// Represents the JSON objects outputted by the Solidity compiler that describe the types used for
// the various pieces of state in the contract. See
// https://docs.soliditylang.org/en/v0.8.3/internals/layout_in_storage.html for more information.
interface SolidityStorageType {
  encoding: 'inplace' | 'mapping' | 'dynamic_array' | 'bytes'
  label: string
  numberOfBytes: number
  key?: string
  value?: string
  base?: string
  members?: SolidityStorageObj[]
}

// Container object returned by the Solidity compiler. See
// https://docs.soliditylang.org/en/v0.8.3/internals/layout_in_storage.html for more information.
export interface SolidityStorageLayout {
  storage: SolidityStorageObj[]
  types: {
    [name: string]: SolidityStorageType
  }
}

interface StorageSlotPair {
  key: string
  val: string
}

export const getStorageLayout = async (
  hre: any, //HardhatRuntimeEnvironment,
  name: string
): Promise<SolidityStorageLayout> => {
  const { sourceName, contractName } = hre.artifacts.readArtifactSync(name)
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
 * Encodes a single variable as a series of key/value storage slot pairs using some storage layout
 * as instructions for how to perform this encoding. Works recursively with struct types.
 * @param variable Variable to encode as key/value slot pairs.
 * @param storageObj Solidity compiler JSON output describing the layout for this
 * @param storageTypes Full list of storage types allowed for this encoding.
 * @param nestedSlotOffset For nested data structures, keeps track of a value to be added onto the
 *  keys for nested values.
 * @returns Variable encoded as a series of key/value slot pairs.
 */
const encodeVariable = (
  variable: any,
  storageObj: SolidityStorageObj,
  storageTypes: {
    [name: string]: SolidityStorageType
  },
  nestedSlotOffset = 0
): Array<StorageSlotPair> => {
  const variableType = storageTypes[storageObj.type]

  // Slot key will be the same no matter what so we can just compute it here.
  const slotKey =
    '0x' +
    remove0x(
      BigNumber.from(
        parseInt(storageObj.slot as any, 10) + nestedSlotOffset
      ).toHexString()
    )
      .padStart(64 - storageObj.offset * 2, '0')
      .padEnd(64, '0')

  if (variableType.encoding === 'inplace') {
    if (
      variableType.label === 'address' ||
      variableType.label.startsWith('contract')
    ) {
      if (!ethers.utils.isAddress(variable)) {
        throw new Error(`invalid address type: ${variable}`)
      }

      // Addresses are right-aligned.
      const slotVal =
        '0x' +
        remove0x(variable)
          .padStart(64, '0')
          .toLowerCase()

      return [
        {
          key: slotKey,
          val: slotVal,
        },
      ]
    } else if (variableType.label === 'bool') {
      // Do some light parsing here to make sure "true" and "false" are recognized.
      if (typeof variable === 'string') {
        if (variable === 'false') {
          variable = false
        }
        if (variable === 'true') {
          variable = true
        }
      }

      if (typeof variable !== 'boolean') {
        throw new Error(`invalid bool type: ${variable}`)
      }

      // Booleans are right-aligned and represented as 0 or 1.
      const slotVal = '0x' + (variable ? '1' : '0').padStart(64, '0')

      return [
        {
          key: slotKey,
          val: slotVal,
        },
      ]
    } else if (variableType.label.startsWith('bytes')) {
      if (!ethers.utils.isHexString(variable, variableType.numberOfBytes)) {
        throw new Error(`invalid bytesN type`)
      }

      // BytesN are **left** aligned (eyeroll).
      const slotVal =
        '0x' +
        remove0x(variable)
          .padEnd(64, '0')
          .toLowerCase()

      return [
        {
          key: slotKey,
          val: slotVal,
        },
      ]
    } else if (variableType.label.startsWith('uint')) {
      if (
        remove0x(BigNumber.from(variable).toHexString()).length / 2 >
        variableType.numberOfBytes
      ) {
        throw new Error(
          `provided ${variableType.label} is too big: ${variable}`
        )
      }

      // Uints are right aligned.
      const slotVal =
        '0x' +
        remove0x(BigNumber.from(variable).toHexString())
          .padStart(64, '0')
          .toLowerCase()

      return [
        {
          key: slotKey,
          val: slotVal,
        },
      ]
    } else if (variableType.label.startsWith('struct')) {
      // Structs are encoded recursively, as defined by their `members` field.
      let slots = []
      for (const [varName, varVal] of Object.entries(variable)) {
        slots = slots.concat(
          encodeVariable(
            varVal,
            variableType.members.find((member) => {
              return member.label === varName
            }),
            storageTypes,
            nestedSlotOffset + storageObj.slot
          )
        )
      }
      return slots
    }
  } else if (variableType.encoding === 'bytes') {
    if (storageObj.offset !== 0) {
      throw new Error(`offset not supported for string/bytes types`)
    }

    // ref: https://docs.soliditylang.org/en/v0.8.4/internals/layout_in_storage.html#bytes-and-string
    const strUtf8 = ethers.utils.toUtf8Bytes(variable)
    if (strUtf8.length < 32) {
      const slotVal = ethers.utils.hexlify(
        ethers.utils.concat([
          ethers.utils
            .concat([strUtf8, ethers.constants.HashZero])
            .slice(0, 31),
          ethers.BigNumber.from(strUtf8.length * 2).toHexString(),
        ])
      )

      return [
        {
          key: slotKey,
          val: slotVal,
        },
      ]
    } else {
      throw new Error('large strings (>31 bytes) not supported')
    }
  } else if (variableType.encoding === 'mapping') {
    console.log(variable, variableType)

    throw new Error('mapping types not yet supported')
  } else if (variableType.encoding === 'dynamic_array') {
    throw new Error('array types not yet supported')
  } else {
    throw new Error(
      `unknown unsupported type ${variableType.encoding} ${variableType.label}`
    )
  }
}

/**
 * Computes the key/value storage slot pairs that would be used if a given set of variable values
 * were applied to a given contract.
 * @param storageLayout Solidity storage layout to use as a template for determining storage slots.
 * @param variables Variable values to apply against the given storage layout.
 * @returns An array of key/value storage slot pairs that would result in the desired state.
 */
export const computeStorageSlots = (
  storageLayout: SolidityStorageLayout,
  variables: any = {}
): Array<StorageSlotPair> => {
  let slots: StorageSlotPair[] = []
  for (const [variableName, variableValue] of Object.entries(variables)) {
    // Find the entry in the storage layout that corresponds to this variable name.
    const storageObj = storageLayout.storage.find((entry) => {
      return entry.label === variableName
    })

    // Complain very loudly if attempting to set a variable that doesn't exist within this layout.
    if (!storageObj) {
      throw new Error(
        `variable name not found in storage layout: ${variableName}`
      )
    }

    // Encode this variable as series of storage slot key/value pairs and save it.
    slots = slots.concat(
      encodeVariable(variableValue, storageObj, storageLayout.types)
    )
  }

  // TODO: Deal with packed storage slots.

  const seen = {}
  for (const slot of slots) {
    if (seen[slot.key]) {
      throw new Error(`packed storage slots not supported`)
    } else {
      seen[slot.key] = true
    }
  }

  return slots
}
