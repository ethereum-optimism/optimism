import { ethers } from 'hardhat'
import { Contract } from 'ethers'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

const DUMMY_ADDRESS = ethers.utils.getAddress('0x' + 'abba'.repeat(10))

describe('Lib_Strings', () => {
  let TestLib_Strings: Contract
  before(async () => {
    TestLib_Strings = await deploy('TestLib_Strings')
  })

  describe('addressToString', () => {
    it('should return a string type', () => {
      // uses the contract interface to find the function's return type
      const returnType =
        TestLib_Strings.interface.functions['addressToString(address)']
          .outputs[0].type

      expect(returnType).to.equal('string')
    })

    it('should convert an address to a lowercase ascii string without the 0x prefix', async () => {
      const asciiString = DUMMY_ADDRESS.substring(2).toLowerCase()

      expect(await TestLib_Strings.addressToString(DUMMY_ADDRESS)).to.equal(
        asciiString
      )
    })
  })

  describe('hexCharToAscii', () => {
    for (let hex = 0; hex < 16; hex++) {
      it(`should convert the hex character ${hex} to its ascii representation`, async () => {
        // converts hex characters to ascii in decimal representation
        const asciiDecimal =
          hex < 10
            ? hex + 48 // 48 is 0x30 in decimal
            : hex + 87 // 87 is 0x57 in decimal

        // converts decimal value to hexadecimal and prepends '0x'
        const asciiHexadecimal = '0x' + asciiDecimal.toString(16)

        expect(await TestLib_Strings.hexCharToAscii(hex)).to.equal(
          asciiHexadecimal
        )
      })
    }
  })
})
