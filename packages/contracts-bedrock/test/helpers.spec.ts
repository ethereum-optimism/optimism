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
        '0xf5f97d03e8be48a4b20ed70c9d8b11f1c851bf949bf602b7580985705bb09077'
      const raw =
        '0x7ef862a077fc5994647d128a4d131d273a5e89e0306aac472494068a4f1fceab83dd073594de3829a23df1479438622a08a116e8eb3f620bb594b7e390864a90b7b923c9f9310c6f98aafe43f707880e043da617250000880de0b6b3a7640000832dc6c080'

      const tx = new DepositTx({
        from: '0xDe3829A23DF1479438622a08a116E8Eb3f620BB5',
        gas: '0x2dc6c0',
        data: '0x',
        to: '0xB7e390864a90b7b923C9f9310C6F98aafE43F707',
        value: '0xde0b6b3a7640000',
        domain: SourceHashDomain.UserDeposit,
        l1BlockHash:
          '0xd1a498e053451fc90bd8a597051a1039010c8e55e2659b940d3070b326e4f4c5',
        logIndex: 0,
        mint: '0xe043da617250000',
      })

      const sourceHash = tx.sourceHash()
      expect(sourceHash).to.deep.eq(
        '0x77fc5994647d128a4d131d273a5e89e0306aac472494068a4f1fceab83dd0735'
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
