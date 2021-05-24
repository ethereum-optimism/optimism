/* External Imports */
import { fromHexString, remove0x } from '@eth-optimism/core-utils'
import { BigNumber, ethers } from 'ethers'
import semver from 'semver'
import { getBuildInfo, getContractArtifact } from './artifacts'

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

/**
 * Takes a slot value (in hex), left-pads it with zeros, and displaces it by a given offset.
 * @param val Hex string value to pad.
 * @param offset Number of bytes to offset from the right.
 * @return Padded hex string.
 */
const padHexSlotValue = (val: string, offset: number): string => {
  return (
    '0x' +
    remove0x(val)
      .padStart(64 - offset * 2, '0') // Pad the start with 64 - offset zero bytes.
      .padEnd(64, '0') // Pad the end (up to 64 bytes) with zero bytes.
      .toLowerCase() // Making this lower case makes assertions more consistent later.
  )
}

/**
 * Retrieves the storageLayout portion of the compiler artifact for a given contract by name. This
 * function is hardhat specific.
 * @param hre HardhatRuntimeEnvironment, required for the readArtifactSync function.
 * @param name Name of the contract to retrieve the storage layout for.
 * @return Storage layout object from the compiler output.
 */
export const getStorageLayout = async (
  name: string
): Promise<SolidityStorageLayout> => {
  const { sourceName, contractName } = await getContractArtifact(name)
  const buildInfo = await getBuildInfo(`${sourceName}:${contractName}`)
  const output = buildInfo.output.contracts[sourceName][contractName]

  if (!semver.satisfies(buildInfo.solcVersion, '>=0.4.x <0.9.x')) {
    throw new Error(
      `Storage layout for Solidity version ${buildInfo.solcVersion} not yet supported. Sorry!`
    )
  }

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
 * ref: https://docs.soliditylang.org/en/v0.8.4/internals/layout_in_storage.html#layout-of-state-variables-in-storage
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
    ).padStart(64, '0')

  if (variableType.encoding === 'inplace') {
    if (
      variableType.label === 'address' ||
      variableType.label.startsWith('contract')
    ) {
      if (!ethers.utils.isAddress(variable)) {
        throw new Error(`invalid address type: ${variable}`)
      }

      return [
        {
          key: slotKey,
          val: padHexSlotValue(variable, storageObj.offset),
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

      return [
        {
          key: slotKey,
          val: padHexSlotValue(variable ? '1' : '0', storageObj.offset),
        },
      ]
    } else if (variableType.label.startsWith('bytes')) {
      if (!ethers.utils.isHexString(variable, variableType.numberOfBytes)) {
        throw new Error(`invalid bytesN type`)
      }

      return [
        {
          key: slotKey,
          val: padHexSlotValue(
            remove0x(variable).padEnd(variableType.numberOfBytes * 2, '0'),
            storageObj.offset
          ),
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

      return [
        {
          key: slotKey,
          val: padHexSlotValue(
            BigNumber.from(variable).toHexString(),
            storageObj.offset
          ),
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
            nestedSlotOffset + parseInt(storageObj.slot as any, 10)
          )
        )
      }
      return slots
    }
  } else if (variableType.encoding === 'bytes') {
    if (storageObj.offset !== 0) {
      // string/bytes types are *not* packed by Solidity.
      throw new Error(`got offset for string/bytes type, should never happen`)
    }

    // `string` types are converted to utf8 bytes, `bytes` are left as-is (assuming 0x prefixed).
    const bytes =
      storageObj.type === 'string'
        ? ethers.utils.toUtf8Bytes(variable)
        : fromHexString(variable)

    // ref: https://docs.soliditylang.org/en/v0.8.4/internals/layout_in_storage.html#bytes-and-string
    if (bytes.length < 32) {
      // NOTE: Solidity docs (see above) specifies that strings or bytes with a length of 31 bytes
      // should be placed into a storage slot where the last byte of the storage slot is the length
      // of the variable in bytes * 2.
      return [
        {
          key: slotKey,
          val: ethers.utils.hexlify(
            ethers.utils.concat([
              ethers.utils
                .concat([bytes, ethers.constants.HashZero])
                .slice(0, 31),
              ethers.BigNumber.from(bytes.length * 2).toHexString(),
            ])
          ),
        },
      ]
    } else {
      throw new Error('large strings (>31 bytes) not supported')
    }
  } else if (variableType.encoding === 'mapping') {
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

  // Dealing with packed storage slots now. We know that a storage slot is packed when two storage
  // slots produced by the above encoding have the same key. In this case, we want to merge the two
  // values into a single bytes32 value. We'll throw an error if the two values overlap (have some
  // byte where both values are non-zero).
  slots = slots.reduce((prevSlots, slot) => {
    // Find some previous slot where we have the same key.
    const prevSlot = prevSlots.find((otherSlot) => {
      return otherSlot.key === slot.key
    })

    if (prevSlot === undefined) {
      // Slot doesn't share a key with any other slot so we can just push it and continue.
      prevSlots.push(slot)
    } else {
      // Slot shares a key with some previous slot.
      // First, we remove the previous slot from the list of slots since we'll be modifying it.
      prevSlots = prevSlots.filter((otherSlot) => {
        return otherSlot.key !== prevSlot.key
      })

      // Now we'll generate a merged value by taking the non-zero bytes from both values. There's
      // probably a more efficient way to do this, but this is relatively easy and straightforward.
      let mergedVal = '0x'
      const valA = remove0x(slot.val)
      const valB = remove0x(prevSlot.val)
      for (let i = 0; i < 64; i += 2) {
        const byteA = valA.slice(i, i + 2)
        const byteB = valB.slice(i, i + 2)

        if (byteA === '00' && byteB === '00') {
          mergedVal += '00'
        } else if (byteA === '00' && byteB !== '00') {
          mergedVal += byteB
        } else if (byteA !== '00' && byteB === '00') {
          mergedVal += byteA
        } else {
          // Should never happen, means our encoding is broken. Values should *never* overlap.
          throw new Error(
            'detected badly encoded packed value, should not happen'
          )
        }
      }

      prevSlots.push({
        key: slot.key,
        val: mergedVal,
      })
    }

    return prevSlots
  }, [])

  return slots
}
