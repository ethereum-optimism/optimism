import '../../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Internal Imports */
import { rlpTests } from '../../test-helpers/data/rlp.test.json'

/* Contract Imports */
import { RLPWriterContractDefinition as RLPWriter } from '../../../src'

/* Logging */
const log = getLogger('rlp-encode', true)

/* Tests */
describe('RLP Encoder', () => {
  const provider = createMockProvider()
  const [wallet1] = getWallets(provider)
  let rlpWriter: Contract

  before(async () => {
    rlpWriter = await deployContract(wallet1, RLPWriter, [], {
      gasLimit: 6700000,
    })
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
    for (const test of Object.keys(rlpTests)) {
      it(`should properly encode ${test}`, async () => {
        const input = rlpTests[test].in
        const encodedOutput = await encode(input)
        encodedOutput.should.equal(rlpTests[test].out)
      })
    }
  })
})
