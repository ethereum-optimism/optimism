import { ethers } from 'hardhat'
import { Contract, BigNumber } from 'ethers'
import { MerkleTree } from 'merkletreejs'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'
import { smock, FakeContract } from '@defi-wonderland/smock'

import { expect } from '../../../setup'
import { deploy, NON_NULL_BYTES32 } from '../../../helpers'

const NODE_COUNTS = [
  2, 3, 7, 9, 13, 63, 64, 123, 128, 129, 255, 1021, 1023, 1024,
]

const makeMerkleTree = (elements: string[]): MerkleTree => {
  const filled: string[] = []
  for (let i = 0; i < Math.pow(2, Math.ceil(Math.log2(elements.length))); i++) {
    if (i < elements.length) {
      filled.push(elements[i])
    } else {
      filled.push(ethers.utils.keccak256('0x' + '00'.repeat(32)))
    }
  }

  return new MerkleTree(
    filled.map(fromHexString),
    (el: Buffer | string): Buffer => {
      return fromHexString(ethers.utils.keccak256(el))
    }
  )
}

describe('Lib_MerkleTree', () => {
  let Lib_MerkleTree: Contract
  let Fake__LibMerkleTree: FakeContract<Contract>
  before(async () => {
    Lib_MerkleTree = await deploy('TestLib_MerkleTree')
    Fake__LibMerkleTree = await smock.fake('TestLib_MerkleTree')
  })

  describe('getMerkleRoot', () => {
    describe('when no elements are provided', () => {
      const elements = []

      it('should revert', async () => {
        await expect(Lib_MerkleTree.getMerkleRoot(elements)).to.be.revertedWith(
          'Lib_MerkleTree: Must provide at least one leaf hash.'
        )
      })
    })

    describe('when a single element is provided', () => {
      const elements = [ethers.utils.keccak256('0x1234')]

      it('should return the input element', async () => {
        expect(await Lib_MerkleTree.getMerkleRoot(elements)).to.equal(
          elements[0]
        )
      })
    })

    describe('when more than one element is provided', () => {
      for (const size of NODE_COUNTS) {
        it(`should generate the correct root when ${size} elements are provided`, async () => {
          const tree = makeMerkleTree(
            [...Array(size)].map((_, i) => {
              return ethers.utils.keccak256(BigNumber.from(i).toHexString())
            })
          )

          expect(await Lib_MerkleTree.getMerkleRoot(tree.getLeaves())).to.equal(
            toHexString(tree.getRoot())
          )
        })
      }
    })

    describe('when odd number of elements is provided', () => {
      it(`should generate the correct root when odd number of elements are provided`, async () => {
        const elements = ['0x12', '0x34', '0x56'].map((value) =>
          ethers.utils.keccak256(value)
        )

        Fake__LibMerkleTree.getMerkleRoot.returns()

        // expect(await Fake__LibMerkleTree.getMerkleRoot(elements)).to.not.be.reverted
        await expect(Lib_MerkleTree.getMerkleRoot(elements)).to.not.be.reverted
      })
    })
  })

  describe('verify', () => {
    describe('when total elements is zero', () => {
      const totalLeaves = 0

      it('should revert', async () => {
        await expect(
          Lib_MerkleTree.verify(
            ethers.constants.HashZero,
            ethers.constants.HashZero,
            0,
            [],
            totalLeaves
          )
        ).to.be.revertedWith(
          'Lib_MerkleTree: Total leaves must be greater than zero.'
        )
      })
    })

    describe('when an index is out of bounds', () => {
      const totalLeaves = 1
      const index = 2

      it('should revert', async () => {
        await expect(
          Lib_MerkleTree.verify(
            ethers.constants.HashZero,
            ethers.constants.HashZero,
            index,
            [],
            totalLeaves
          )
        ).to.be.revertedWith('Lib_MerkleTree: Index out of bounds.')
      })
    })

    describe('when total siblings does not match provided total leaves', () => {
      const totalLeaves = 8
      const siblings = [ethers.constants.HashZero, ethers.constants.HashZero]

      it('should revert', async () => {
        await expect(
          Lib_MerkleTree.verify(
            ethers.constants.HashZero,
            ethers.constants.HashZero,
            0,
            siblings,
            totalLeaves
          )
        ).to.be.revertedWith(
          'Lib_MerkleTree: Total siblings does not correctly correspond to total leaves.'
        )
      })
    })

    describe('with valid proof for a single element', () => {
      const root = NON_NULL_BYTES32
      const leaf = NON_NULL_BYTES32
      const index = 0
      const siblings = []
      const totalLeaves = 1

      it('should succeed', async () => {
        expect(
          await Lib_MerkleTree.verify(root, leaf, index, siblings, totalLeaves)
        ).to.equal(true)
      })
    })

    describe('with valid proof for more than one element', () => {
      for (const size of NODE_COUNTS) {
        describe(`for a tree with ${size} total elements`, () => {
          const tree = makeMerkleTree(
            [...Array(size)].map((_, i) => {
              return ethers.utils.keccak256(BigNumber.from(i).toHexString())
            })
          )

          for (let i = 0; i < size; i += Math.ceil(size / 8)) {
            it(`should verify a proof for the ${i}(th/st/rd, whatever) element`, async () => {
              const proof = tree
                .getProof(tree.getLeaves()[i], i)
                .map((element) => {
                  return element.data
                })

              expect(
                await Lib_MerkleTree.verify(
                  tree.getRoot(),
                  tree.getLeaves()[i],
                  i,
                  proof,
                  size
                )
              ).to.equal(true)
            })
          }
        })
      }
    })
  })
})
