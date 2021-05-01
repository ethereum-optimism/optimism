import { expect } from '../setup'

/* Imports: External */
import { ethers } from 'ethers'
import MerkleTree from 'merkletreejs'
import { fromHexString } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  getChugSplashActionBundle,
  getActionHash,
  ChugSplashAction,
} from '../../src'

const makeAndVerifyBundle = (actions: ChugSplashAction[]): void => {
  const bundle = getChugSplashActionBundle(actions)

  const tree = new MerkleTree(
    [],
    (el: Buffer | string): Buffer => {
      return fromHexString(ethers.utils.keccak256(el))
    }
  )

  for (const action of bundle.actions) {
    expect(
      tree.verify(
        action.proof.siblings.map((sibling, idx) => {
          return {
            position:
              action.proof.actionIndex
                .toString(2)
                .split('')
                .reverse()[idx] === '1'
                ? 'left'
                : 'right',
            data: sibling,
          }
        }),
        fromHexString(getActionHash(action.action)),
        fromHexString(bundle.root)
      )
    ).to.equal(true)
  }
}

describe('ChugSplash action bundling', () => {
  describe('getChugSplashActionBundle', () => {
    it('should bundle a set code action', () => {
      makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          code: `0x${'22'.repeat(32)}`,
        },
      ])
    })

    it('should bundle a set storage action', () => {
      makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          key: `0x${'22'.repeat(32)}`,
          value: `0x${'22'.repeat(32)}`,
        },
      ])
    })

    it('should bundle multiple set code actions', () => {
      makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          code: `0x${'22'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          code: `0x${'33'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          code: `0x${'44'.repeat(32)}`,
        },
      ])
    })

    it('should bundle multiple set storage actions', () => {
      makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          key: `0x${'22'.repeat(32)}`,
          value: `0x${'22'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          key: `0x${'33'.repeat(32)}`,
          value: `0x${'33'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          key: `0x${'44'.repeat(32)}`,
          value: `0x${'44'.repeat(32)}`,
        },
      ])
    })

    it('should bundle a set code action and a set storage action', () => {
      makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          code: `0x${'22'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          key: `0x${'44'.repeat(32)}`,
          value: `0x${'44'.repeat(32)}`,
        },
      ])
    })

    it('should bundle multiple set code and set storage actions', () => {
      makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          code: `0x${'22'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          code: `0x${'33'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          key: `0x${'44'.repeat(32)}`,
          value: `0x${'44'.repeat(32)}`,
        },
        {
          target: ethers.constants.AddressZero,
          key: `0x${'55'.repeat(32)}`,
          value: `0x${'55'.repeat(32)}`,
        },
      ])
    })
  })
})
