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

export const isSetStorageAction = (
  action: ChugSplashAction
): action is SetStorageAction => {
  return (
    (action as SetStorageAction).key !== undefined &&
    (action as SetStorageAction).value !== undefined
  )
}

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

export const getActionHash = (action: RawChugSplashAction): string => {
  return ethers.utils.keccak256(
    ethers.utils.defaultAbiCoder.encode(
      ['uint8', 'address', 'bytes'],
      [action.actionType, action.target, action.data]
    )
  )
}

export const getChugSplashActionBundle = (
  actions: ChugSplashAction[]
): ChugSplashActionBundle => {
  const rawActions = actions.map((action) => {
    return toRawChugSplashAction(action)
  })

  const elements = rawActions.map((action) => {
    return getActionHash(action)
  })

  const filledElements = []
  for (let i = 0; i < Math.pow(2, Math.ceil(Math.log2(elements.length))); i++) {
    if (i < elements.length) {
      filledElements.push(elements[i])
    } else {
      filledElements.push(ethers.utils.keccak256(ethers.constants.HashZero))
    }
  }

  const bufs = filledElements.map((element) => {
    return fromHexString(element)
  })

  const tree = new MerkleTree(
    bufs,
    (el: Buffer | string): Buffer => {
      return fromHexString(ethers.utils.keccak256(el))
    }
  )

  return {
    root: toHexString(tree.getRoot()),
    actions: rawActions.map((action, idx) => {
      return {
        action: action,
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
