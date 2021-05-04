/* External Imports */
import { fromHexString, toHexString } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'
import MerkleTree from 'merkletreejs'

export enum ChugSplashActionType {
  SET_CODE,
  SET_STORAGE,
}

export interface RawChugSplashAction {
  actionType: ChugSplashActionType
  target: string
  data: string
}

export interface SetCodeAction {
  target: string
  code: string
}

export interface SetStorageAction {
  target: string
  key: string
  value: string
}

export type ChugSplashAction = SetCodeAction | SetStorageAction

export interface ChugSplashActionBundle {
  root: string
  actions: Array<{
    action: RawChugSplashAction
    proof: {
      actionIndex: number
      siblings: string[]
    }
  }>
}

/**
 * Checks whether a given action is a SetStorage action.
 * @param action ChugSplash action to check.
 * @return `true` if the action is a SetStorage action, `false` otherwise.
 */
export const isSetStorageAction = (
  action: ChugSplashAction
): action is SetStorageAction => {
  return (
    (action as SetStorageAction).key !== undefined &&
    (action as SetStorageAction).value !== undefined
  )
}

/**
 * Converts the "nice" action structs into a "raw" action struct (better for Solidity but
 * worse for users here).
 * @param action ChugSplash action to convert.
 * @return Converted "raw" ChugSplash action.
 */
export const toRawChugSplashAction = (
  action: ChugSplashAction
): RawChugSplashAction => {
  if (isSetStorageAction(action)) {
    return {
      actionType: ChugSplashActionType.SET_STORAGE,
      target: action.target,
      data: ethers.utils.defaultAbiCoder.encode(
        ['bytes32', 'bytes32'],
        [action.key, action.value]
      ),
    }
  } else {
    return {
      actionType: ChugSplashActionType.SET_CODE,
      target: action.target,
      data: action.code,
    }
  }
}

/**
 * Computes the hash of an action.
 * @param action Action to compute the hash of.
 * @return Hash of the action.
 */
export const getActionHash = (action: RawChugSplashAction): string => {
  return ethers.utils.keccak256(
    ethers.utils.defaultAbiCoder.encode(
      ['uint8', 'address', 'bytes'],
      [action.actionType, action.target, action.data]
    )
  )
}

/**
 * Generates an action bundle from a set of actions. Effectively encodes the inputs that will be
 * provided to the ChugSplashDeployer contract.
 * @param actions Series of SetCode or SetStorage actions to bundle.
 * @return Bundled actions.
 */
export const getChugSplashActionBundle = (
  actions: ChugSplashAction[]
): ChugSplashActionBundle => {
  // First turn the "nice" action structs into raw actions.
  const rawActions = actions.map((action) => {
    return toRawChugSplashAction(action)
  })

  // Now compute the hash for each action.
  const elements = rawActions.map((action) => {
    return getActionHash(action)
  })

  // Pad the list of elements out with default hashes if len < a power of 2.
  const filledElements = []
  for (let i = 0; i < Math.pow(2, Math.ceil(Math.log2(elements.length))); i++) {
    if (i < elements.length) {
      filledElements.push(elements[i])
    } else {
      filledElements.push(ethers.utils.keccak256(ethers.constants.HashZero))
    }
  }

  // merkletreejs expects things to be buffers.
  const tree = new MerkleTree(
    filledElements.map((element) => {
      return fromHexString(element)
    }),
    (el: Buffer | string): Buffer => {
      return fromHexString(ethers.utils.keccak256(el))
    }
  )

  return {
    root: toHexString(tree.getRoot()),
    actions: rawActions.map((action, idx) => {
      return {
        action,
        proof: {
          actionIndex: idx,
          siblings: tree.getProof(getActionHash(action), idx).map((element) => {
            return element.data
          }),
        },
      }
    }),
  }
}
