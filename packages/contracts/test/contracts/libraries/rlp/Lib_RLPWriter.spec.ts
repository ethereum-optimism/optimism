import { Contract } from 'ethers'

import { expect } from '../../../setup'
import { Lib_RLPWriter_TEST_JSON } from '../../../data'
import { deploy } from '../../../helpers'

const encode = async (Lib_RLPWriter: Contract, input: any): Promise<void> => {
  if (Array.isArray(input)) {
    return Lib_RLPWriter.writeList(
      await Promise.all(
        input.map(async (el) => {
          return encode(Lib_RLPWriter, el)
        })
      )
    )
  } else if (Number.isInteger(input)) {
    return Lib_RLPWriter.writeUint(input)
  } else {
    return Lib_RLPWriter.writeString(input)
  }
}

describe('Lib_RLPWriter', () => {
  let Lib_RLPWriter: Contract
  before(async () => {
    Lib_RLPWriter = await deploy('TestLib_RLPWriter')
  })

  describe('Official Ethereum RLP Tests', () => {
    for (const [key, test] of Object.entries(Lib_RLPWriter_TEST_JSON)) {
      it(`should properly encode: ${key}`, async () => {
        expect(await encode(Lib_RLPWriter, test.in)).to.equal(test.out)
      })
    }
  })

  describe('writeBool', () => {
    it(`should encode bool: true`, async () => {
      expect(await Lib_RLPWriter.writeBool(true)).to.equal('0x01')
    })

    it(`should encode bool: false`, async () => {
      expect(await Lib_RLPWriter.writeBool(false)).to.equal('0x80')
    })
  })

  describe('Use of library with other memory-modifying operations', () => {
    it('should allow creation of a contract beforehand and still work', async () => {
      expect(
        await Lib_RLPWriter.callStatic.writeAddressWithTaintedMemory(
          '0x1234123412341234123412341234123412341234'
        )
      ).to.eq('0x941234123412341234123412341234123412341234')
    })
  })
})
