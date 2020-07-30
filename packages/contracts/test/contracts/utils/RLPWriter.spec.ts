import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import { RLP_TEST_JSON } from '../../test-helpers'

/* Tests */
describe('RLPWriter', () => {
  let RlpWriter: ContractFactory
  let rlpWriter: Contract
  before(async () => {
    RlpWriter = await ethers.getContractFactory('MockRLPWriter')
    rlpWriter = await RlpWriter.deploy()
  })

  const encode = async (input: any) => {
    if (Array.isArray(input)) {
      // Handle lists.
      const encodedElements = []

      // Recursively encode every element in the list.
      for (const element of input) {
        const encodedElement = encode(element)
        encodedElements.push(encodedElement)
      }

      // Encode the list of encoded elements.
      const encodedList = await rlpWriter.encodeList(encodedElements)
      return encodedList
    } else if (Number.isInteger(input)) {
      // Handle integers.
      const encodedUint = await rlpWriter.encodeUint(input)
      return encodedUint
    } else if (input[0] === '#') {
      // Handle big numbers.
      // Remove '#'' from the input.
      input = input.slice(1)

      const encodedUint = await rlpWriter.encodeInt(input)
      return encodedUint
    } else {
      // Handle strings.
      const encodedString = await rlpWriter.encodeString(input)
      return encodedString
    }
  }

  describe('Official Ethereum RLP Tests', async () => {
    for (const test of Object.keys(RLP_TEST_JSON)) {
      it(`should properly encode ${test}`, async () => {
        const input = RLP_TEST_JSON[test].in
        const encodedOutput = await encode(input)
        encodedOutput.should.equal(RLP_TEST_JSON[test].out)
      })
    }
  })
})
