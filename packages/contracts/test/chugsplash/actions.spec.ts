import { expect } from '../setup'

/* Imports: External */
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import MerkleTree from 'merkletreejs'
import { fromHexString } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  getChugSplashActionBundle,
  getActionHash,
  ChugSplashAction,
} from '../../src'

describe('ChugSplash action bundling', () => {
  let Helper_ChugSplashMock: Contract
  before(async () => {
    Helper_ChugSplashMock = await (
      await ethers.getContractFactory('Helper_ChugSplashMock')
    ).deploy()
  })

  const makeAndVerifyBundle = async (
    actions: ChugSplashAction[]
  ): Promise<void> => {
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
            const positions = action.proof.actionIndex
              .toString(2)
              .split('')
              .reverse()
            return {
              position: positions[idx] === '1' ? 'left' : 'right',
              data: sibling,
            }
          }),
          fromHexString(getActionHash(action.action)),
          fromHexString(bundle.root)
        )
      ).to.equal(true)

      await expect(
        Helper_ChugSplashMock.validateAction(
          bundle.root,
          bundle.actions.length,
          action.action,
          action.proof
        )
      ).to.not.be.reverted
    }
  }

  describe('getChugSplashActionBundle', () => {
    it('should bundle a set code action', async () => {
      await makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          code: `0x${'22'.repeat(32)}`,
        },
      ])
    })

    it('should bundle a set storage action', async () => {
      await makeAndVerifyBundle([
        {
          target: ethers.constants.AddressZero,
          key: `0x${'22'.repeat(32)}`,
          value: `0x${'22'.repeat(32)}`,
        },
      ])
    })

    it('should bundle multiple set code actions', async () => {
      await makeAndVerifyBundle([
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

    it('should bundle multiple set storage actions', async () => {
      await makeAndVerifyBundle([
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

    it('should bundle a set code action and a set storage action', async () => {
      await makeAndVerifyBundle([
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

    it('should bundle multiple set code and set storage actions', async () => {
      await makeAndVerifyBundle([
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
