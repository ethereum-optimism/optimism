/* External Imports */
import {
  MerkleIntervalTree,
  MerkleIntervalTreeNode,
  MerkleStateIntervalTree,
} from '@pigi/core'
import BigNum = require('bn.js')

/* Contract Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as Commitment from '../build/CommitmentChain.json'
/* Logging */
import debug from 'debug'
import { check } from 'ethers/utils/wordlist'
const log = debug('test:info:state-ownership')
/* Testing Setup */
import chai = require('chai')
export const should = chai.should()

describe.skip('Commitment Contract', () => {
  const provider = createMockProvider()
  const [wallet, walletTo] = getWallets(provider)
  let commitmentContract

  beforeEach(async () => {
    commitmentContract = await deployContract(wallet, Commitment, [])
  })

  describe('Commitment Contract', () => {
    it('correctly calculates the parent of two state subtree nodes', async () => {
      const leftSibling = {
        hashValue:
          '0x1111111111111111111111111111111111111111111111111111111111111111',
        start: '0x00',
      }
      const rightSibling = {
        hashValue:
          '0x1111111111111111111111111111111111111111111111111111111111111111',
        start: '0xaa',
      }
      const abiPacked = await commitmentContract.getAbiPacked(
        leftSibling,
        rightSibling
      )
      //   console.log('abipacked from solidity is: ', abiPacked)
      const contractParent = await commitmentContract.stateSubtreeParent(
        leftSibling,
        rightSibling
      )
      //   console.log('contract parent is: ', contractParent)

      //   const tsLeftSibling = new MerkleIntervalTreeNode(
      //     Buffer.from(leftSibling.hashValue.slice(2), 'hex'),
      //     new BigNum(leftSibling.start.slice(2), 'hex').toBuffer('be', 16)
      //   )
      //   //   console.log('typescript left sibling is ', tsLeftSibling)
      //   const tsRightSibling = new MerkleIntervalTreeNode(
      //     Buffer.from(rightSibling.hashValue.slice(2), 'hex'),
      //     new BigNum(rightSibling.start.slice(2), 'hex').toBuffer('be', 16)
      //   )
      //   //   console.log('typescript right sibling is ', tsRightSibling)

      //   const tsParent = MerkleIntervalTree.parent(tsLeftSibling, tsRightSibling)
      //   //   console.log('typescript parent is: ', tsParent)
    })
  })
})
