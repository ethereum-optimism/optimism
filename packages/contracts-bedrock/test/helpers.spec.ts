import { expect } from 'chai'
import { BigNumber } from 'ethers'

import { DepositTx, SourceHashDomain } from '../src'

describe('Helpers', () => {
  describe('DepositTx', () => {
    // TODO(tynes): this is out of date now that the subversion
    // byte has been added
    it('should serialize/deserialize and hash', () => {
      // constants serialized using optimistic-geth
      // TODO(tynes): more tests
      const hash =
        '0xf58e30138cb01330f6450b9a5e717a63840ad2e21f17340105b388ad3c668749'
      const raw =
        '0x7e00f862a0f923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc94f39fd6e51aad88f6f4ce6ab8827279cfffb9226694b79f76ef2c5f0286176833e7b2eee103b1cc3244880e043da617250000880de0b6b3a7640000832dc6c080'

      const tx = new DepositTx({
        from: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
        gas: '0x2dc6c0',
        data: '0x',
        to: '0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244',
        value: '0xde0b6b3a7640000',
        domain: SourceHashDomain.UserDeposit,
        l1BlockHash:
          '0xd25df7858efc1778118fb133ac561b138845361626dfb976699c5287ed0f4959',
        logIndex: 1,
        mint: '0xe043da617250000',
      })

      const sourceHash = tx.sourceHash()
      expect(sourceHash).to.deep.eq(
        '0xf923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc'
      )

      const encoded = tx.encode()
      expect(encoded).to.deep.eq(raw)
      const hashed = tx.hash()
      expect(hashed).to.deep.eq(hash)

      const decoded = DepositTx.decode(raw, {
        domain: SourceHashDomain.UserDeposit,
        l1BlockHash: tx.l1BlockHash,
        logIndex: tx.logIndex,
      })
      expect(decoded.from).to.deep.eq(tx.from)
      expect(decoded.gas).to.deep.eq(BigNumber.from(tx.gas))
      expect(decoded.data).to.deep.eq(tx.data)
      expect(decoded.to).to.deep.eq(tx.to)
      expect(decoded.value).to.deep.eq(BigNumber.from(tx.value))
      expect(decoded.domain).to.deep.eq(SourceHashDomain.UserDeposit)
      expect(decoded.l1BlockHash).to.deep.eq(tx.l1BlockHash)
      expect(decoded.logIndex).to.deep.eq(tx.logIndex)
      expect(decoded.mint).to.deep.eq(BigNumber.from(tx.mint))
    })
  })
})
