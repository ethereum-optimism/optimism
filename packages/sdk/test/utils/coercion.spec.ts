import { Provider } from '@ethersproject/abstract-provider'
import { Contract } from 'ethers'
import { ethers } from 'hardhat'

import { expect } from '../setup'
import { toSignerOrProvider, toTransactionHash } from '../../src'

describe('type coercion utils', () => {
  describe('toSignerOrProvider', () => {
    it('should convert a string to a JsonRpcProvider', () => {
      const provider = toSignerOrProvider('http://localhost:8545')
      expect(Provider.isProvider(provider)).to.be.true
    })

    it('should not do anything with a provider', () => {
      const provider = toSignerOrProvider(ethers.provider)
      expect(provider).to.deep.equal(ethers.provider)
    })
  })

  describe('toTransactionHash', () => {
    describe('string inputs', () => {
      it('should return the input if the input is a valid transaction hash', () => {
        const input = '0x' + '11'.repeat(32)
        expect(toTransactionHash(input)).to.equal(input)
      })

      it('should throw an error if the input is a hex string but not a transaction hash', () => {
        const input = '0x' + '11'.repeat(31)
        expect(() => toTransactionHash(input)).to.throw(
          'Invalid transaction hash'
        )
      })

      it('should throw an error if the input is not a hex string', () => {
        const input = 'hi mom look at me go'
        expect(() => toTransactionHash(input)).to.throw(
          'Invalid transaction hash'
        )
      })
    })

    describe('transaction inputs', () => {
      let AbsolutelyNothing: Contract
      before(async () => {
        AbsolutelyNothing = (await (
          await ethers.getContractFactory('AbsolutelyNothing')
        ).deploy()) as any
      })

      it('should return the transaction hash if the input is a transaction response', async () => {
        const tx = await AbsolutelyNothing.doAbsolutelyNothing()
        expect(toTransactionHash(tx)).to.equal(tx.hash)
      })

      it('should return the transaction hash if the input is a transaction receipt', async () => {
        const tx = await AbsolutelyNothing.doAbsolutelyNothing()
        const receipt = await tx.wait()
        expect(toTransactionHash(receipt)).to.equal(receipt.transactionHash)
      })
    })

    describe('other types', () => {
      it('should throw if given a number as an input', () => {
        expect(() => toTransactionHash(1234 as any)).to.throw(
          'Invalid transaction'
        )
      })

      it('should throw if given a function as an input', () => {
        expect(() =>
          toTransactionHash((() => {
            return 1234
          }) as any)
        ).to.throw('Invalid transaction')
      })
    })
  })
})
