import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { RLPEncodeContractDefinition as RLPEncode } from '@eth-optimism/rollup-contracts'

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { rlpTests } from './test-files/rlptest.json'

const log = getLogger('rlp-encode', true)

/* Begin tests */
describe('RLP Encoder', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let rlpEncode

  /* Link libraries before tests */
  before(async () => {
    rlpEncode = await deployContract(wallet1, RLPEncode, [], {
      gasLimit: 6700000,
    })
  })

  const encode = async (input) => {
    // handle lists
    if (Array.isArray(input)) {
      const encodedElements = []
      // recursively encode every element in the list
      for (const element of input) {
        const encodedElement = encode(element)
        encodedElements.push(encodedElement)
      }
      // encode the list of encoded elements
      const encodedList = await rlpEncode.encodeList(encodedElements)
      return encodedList
      // handle integers
    } else if (Number.isInteger(input)) {
      const encodedUint = await rlpEncode.encodeUint(input)
      return encodedUint
      // handle big numbers
    } else if (input[0] === '#') {
      // remove '#'' from big int
      input = input.slice(1)
      const encodedUint = await rlpEncode.encodeInt(input)
      return encodedUint
      // handle strings
    } else {
      const encodedString = await rlpEncode.encodeString(input)
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
