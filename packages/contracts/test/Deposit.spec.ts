import './setup'

/* External Imports */
import { abi, AbiRange, AbiStateObject, AbiStateUpdate, hexStringify } from '@pigi/utils'
import BigNum = require('bn.js')
/* Contract Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle';
import * as BasicTokenMock from '../build/BasicTokenMock.json'
import * as Deposit from '../build/Deposit.json'
import * as Commitment from '../build/Commitment.json'
import * as OwnershipPredicate from '../build/OwnershipPredicate.json'

/* Logging */
import debug from 'debug'
const log = debug('test:info:state-ownership')

async function mineBlocks(provider: any, numBlocks: number = 1) {
  for (let i = 0; i < numBlocks ; i++) {
    await provider.send('evm_mine', []);
  }
}

async function depositErc20(wallet, token, depositContract, ownershipPredicate) {
  // Deposit some money into the ownership predicate
  await token.approve(depositContract.address, 500)
  const depositData = abi.encode(['address'], [wallet.address])
  const depositStateObject = new AbiStateObject(ownershipPredicate.address, depositData)
  await depositContract.deposit(100, depositStateObject)
}

describe('Deposit with Ownership', () => {
  const provider = createMockProvider()
  const [wallet, walletTo] = getWallets(provider)
  let token
  let depositContract
  let commitmentContract
  let ownershipPredicate

  beforeEach(async () => {
    token = await deployContract(wallet, BasicTokenMock, [wallet.address, 1000])
    commitmentContract = await deployContract(wallet, Commitment, [])
    depositContract = await deployContract(wallet, Deposit, [token.address, commitmentContract.address])
    ownershipPredicate = await deployContract(wallet, OwnershipPredicate, [depositContract.address])
  });

  describe('Deposit', () => {
    it('does not throw when deposit is called after approving erc20 movement', async () => {
      await token.approve(depositContract.address, 500)
      await depositContract.deposit(123, { predicateAddress: '0xF6c105ED2f0f5Ffe66501a4EEdaD86E10df19054', data: '0x1234' })
    })

    it('allows exits to be started and finalized on deposits', async () => {
      // Deposit some money into the ownership predicate
      await token.approve(depositContract.address, 500)
      const depositData = abi.encode(['address'], [wallet.address])
      const depositStateObject = new AbiStateObject(ownershipPredicate.address, depositData)
      await depositContract.deposit(100, depositStateObject)
      // Attempt to start an exit on this deposit
      const depositRange = { start: hexStringify(new BigNum(0)), end: hexStringify(new BigNum(100)) }
      await ownershipPredicate.startExit({
        stateUpdate: {
          range: depositRange,
          stateObject: depositStateObject,
          depositAddress: depositContract.address,
          plasmaBlockNumber: 0
        },
        subrange: depositRange
      })
      // Get the challenge peroid
      const challengePeroid = await depositContract.CHALLENGE_PERIOD()
      // Mine the blocks
      await mineBlocks(provider, challengePeroid + 1)
      // Now finalize the exit
      await ownershipPredicate.finalizeExit({
        stateUpdate: {
          range: depositRange,
          stateObject: depositStateObject,
          depositAddress: depositContract.address,
          plasmaBlockNumber: 0
        },
        subrange: depositRange
      }, 100)
    })
  })

  describe('startCheckpoint', () => {
    it('should create a checkpoint which can be exited without throwing', async () => {
      // Deposit some money into the ownership predicate
      await token.approve(depositContract.address, 500)
      const depositData = abi.encode(['address'], [wallet.address])
      const depositStateObject = new AbiStateObject(ownershipPredicate.address, depositData)
      await depositContract.deposit(100, depositStateObject)
      // Attempt to start a checkpoint on a stateUpdate
      const stateUpdateRange = { start: hexStringify(new BigNum(10)), end: hexStringify(new BigNum(20)) }
      const checkpoint = {
        stateUpdate: {
          range: stateUpdateRange,
          stateObject: depositStateObject,
          depositAddress: depositContract.address,
          plasmaBlockNumber: 10
        },
        subrange: stateUpdateRange
      }
      await depositContract.startCheckpoint(checkpoint, '0x1234', 100)
      await ownershipPredicate.startExit(checkpoint)
      // should not throw
    })
  })

  describe('deprecateExit', () => {
    it('should deprecate an exit without throwing', async () => {
      // Deposit some money into the ownership predicate
      await token.approve(depositContract.address, 500)
      const depositData = abi.encode(['address'], [wallet.address])
      const depositStateObject = new AbiStateObject(ownershipPredicate.address, depositData)
      await depositContract.deposit(100, depositStateObject)
      // Attempt to start an exit on this deposit
      const depositRange = { start: hexStringify(new BigNum(0)), end: hexStringify(new BigNum(100)) }
      const checkpoint = {
        stateUpdate: {
          range: depositRange,
          stateObject: depositStateObject,
          depositAddress: depositContract.address,
          plasmaBlockNumber: 0
        },
        subrange: depositRange
      }
      await ownershipPredicate.startExit(checkpoint)
      // Now deprecate the exit
      await ownershipPredicate.deprecateExit(checkpoint)
    })
  })

  describe('deleteOutdatedExit', () => {
    it('should delete an exit if there is a later checkpoint on that range', async () => {
      // Deposit some money into the ownership predicate
      await token.approve(depositContract.address, 500)
      const depositData = abi.encode(['address'], [wallet.address])
      const depositStateObject = new AbiStateObject(ownershipPredicate.address, depositData)
      await depositContract.deposit(100, depositStateObject)
      // Add a later checkpoint
      const stateUpdateRange = { start: hexStringify(new BigNum(10)), end: hexStringify(new BigNum(20)) }
      const checkpoint = {
        stateUpdate: {
          range: stateUpdateRange,
          stateObject: depositStateObject,
          depositAddress: depositContract.address,
          plasmaBlockNumber: 10
        },
        subrange: stateUpdateRange
      }
      await depositContract.startCheckpoint(checkpoint, '0x1234', 100)
      // Now fast forward until the checkpoint is finalized
      // Get the challenge peroid
      const challengePeroid = await depositContract.CHALLENGE_PERIOD()
      // Mine the blocks
      await mineBlocks(provider, challengePeroid + 1)
      // Now that we have a finalized checkpoint, attempt an exit on the original deposit
      const depositRange = { start: hexStringify(new BigNum(0)), end: hexStringify(new BigNum(100)) }
      const depositCheckpoint = {
        stateUpdate: {
          range: depositRange,
          stateObject: depositStateObject,
          depositAddress: depositContract.address,
          plasmaBlockNumber: 0
        },
        subrange: depositRange
      }
      await ownershipPredicate.startExit(depositCheckpoint)
      // Uh oh! This exit is invalid! Let's delete it
      await depositContract.deleteOutdatedExit(depositCheckpoint, checkpoint)
    })
  })

  describe('helper functions', () => {
    describe('subRange', () => {
      it('returns true for equal ranges', async () => {
        const res = await depositContract.isSubrange({
          start: 50,
          end: 100
        },
        {
          start: 50,
          end: 100
        })
        res.should.equal(true)
      })

      it('returns true for a strict subrange', async () => {
        const res = await depositContract.isSubrange({
          start: 51,
          end: 99
        },
        {
          start: 50,
          end: 100
        })
        res.should.equal(true)
      })

      it('returns false for not a subrange', async () => {
        const res = await depositContract.isSubrange({
          start: 49,
          end: 100
        },
        {
          start: 50,
          end: 100
        })
        res.should.equal(false)
      })
    })
  })

  describe('DepositedRanges', () => {
    it('can be extended', async () => {
      await depositContract.extendDepositedRanges(100)
      const res = await depositContract.depositedRanges(100)
      res.start.toString().should.equal('0')
      res.end.toString().should.equal('100')
    })

    it('can be extended twice', async () => {
      await depositContract.extendDepositedRanges(100)
      await depositContract.extendDepositedRanges(50)
      const res = await depositContract.depositedRanges(150)
      res.start.toString().should.equal('0')
      res.end.toString().should.equal('150')
      // make sure the other range is gone
      const deletedRange = await depositContract.depositedRanges(100)
      deletedRange.start.toString().should.equal('0')
      deletedRange.end.toString().should.equal('0')
    })

    it('can be extended and then deleted', async () => {
      await depositContract.extendDepositedRanges(100)
      await depositContract.removeDepositedRange({
        start: 0,
        end: 100
      }, 100)
      const res = await depositContract.depositedRanges(100)
      res.start.toString().should.equal('0')
      res.end.toString().should.equal('0')
    })

    it('can be extended and then shortend on the left side', async () => {
      await depositContract.extendDepositedRanges(100)
      await depositContract.removeDepositedRange({
        start: 0,
        end: 75
      }, 100)
      const res = await depositContract.depositedRanges(100)
      res.start.toString().should.equal('75')
      res.end.toString().should.equal('100')
    })

    it('can be extended and then shortend on the right side', async () => {
      await depositContract.extendDepositedRanges(100)
      await depositContract.removeDepositedRange({
        start: 25,
        end: 100
      }, 100)
      const res = await depositContract.depositedRanges(25)
      res.start.toString().should.equal('0')
      res.end.toString().should.equal('25')
    })

    it('can be extended and then split into two', async () => {
      await depositContract.extendDepositedRanges(100)
      await depositContract.removeDepositedRange({
        start: 25,
        end: 75
      }, 100)
      const range1 = await depositContract.depositedRanges(25)
      const range2 = await depositContract.depositedRanges(100)
      // check first range
      range1.start.toString().should.equal('0')
      range1.end.toString().should.equal('25')
      // check second range
      range2.start.toString().should.equal('75')
      range2.end.toString().should.equal('100')
    })

    it('can be extended, right side deleted, and then extended again', async () => {
      await depositContract.extendDepositedRanges(100)
      await depositContract.removeDepositedRange({
        start: 25,
        end: 100
      }, 100)
      await depositContract.extendDepositedRanges(50)
      // check that everything is there
      const range1 = await depositContract.depositedRanges(25)
      const range2 = await depositContract.depositedRanges(150)
      // check first range
      range1.start.toString().should.equal('0')
      range1.end.toString().should.equal('25')
      // check second range
      range2.end.toString().should.equal('150')
      range2.start.toString().should.equal('100')
    })
  })
})
