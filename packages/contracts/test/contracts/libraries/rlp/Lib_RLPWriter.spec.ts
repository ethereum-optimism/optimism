/* tslint:disable:no-empty */
import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract } from 'ethers'

/* Internal Imports */
import { Lib_RLPWriter_TEST_JSON } from '../../../data'

const encode = async (Lib_RLPWriter: Contract, input: any): Promise<void> => {
  if (Array.isArray(input)) {
    const elements = await Promise.all(
      input.map(async (el) => {
        return encode(Lib_RLPWriter, el)
      })
    )

    return Lib_RLPWriter.writeList(elements)
  } else if (Number.isInteger(input)) {
    return Lib_RLPWriter.writeUint(input)
  } else if (input[0] === '#') {
    return Lib_RLPWriter.writeInt(input.slice(1))
  } else {
    return Lib_RLPWriter.writeString(input)
  }
}

describe('Lib_RLPWriter', () => {
  let Lib_RLPWriter: Contract
  before(async () => {
    Lib_RLPWriter = await (
      await ethers.getContractFactory('TestLib_RLPWriter')
    ).deploy()
  })

  describe('Official Ethereum RLP Tests', () => {
    for (const [key, test] of Object.entries(Lib_RLPWriter_TEST_JSON)) {
      it(`should properly encode: ${key}`, async () => {
        expect(await encode(Lib_RLPWriter, test.in)).to.equal(test.out)
      })
    }
  })
})
