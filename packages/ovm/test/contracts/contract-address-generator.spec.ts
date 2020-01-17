import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger } from '@pigi/core-utils'
import { Address } from '@pigi/rollup-core'
import { utils } from 'ethers'
import { create2Tests } from './test-files/create2test.json'

/* Contract Imports */
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

/* Internal Imports */
import { buildCreate2Address } from '../helpers'

const log = getLogger('contract-address-generator', true)

/*********
 * TESTS *
 *********/

describe('ContractAddressGenerator', () => {
  const [wallet1, wallet2] = getWallets(createMockProvider())
  // Create pointers to our contractAddressGenerator
  let contractAddressGenerator
  let rlpEncode

  /* Link libraries before tests */
  before(async () => {
    rlpEncode = await deployContract(wallet1, RLPEncode, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // First deploy the contract address
    contractAddressGenerator = await deployContract(
      wallet1,
      ContractAddressGenerator,
      [rlpEncode.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  /*
   * Test getAddressFromCREATE
   */
  describe('getAddressFromCREATE', async () => {
    it('returns expected address, nonce: 1', async () => {
      const nonce = 1
      const expectedAddress = utils.getContractAddress({
        from: wallet1.address,
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        wallet1.address,
        nonce
      )
      computedAddress.should.equal(expectedAddress)
    })
    it('returns expected address, nonce: 1, different origin address', async () => {
      const nonce = 1
      const expectedAddress = utils.getContractAddress({
        from: wallet2.address,
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        wallet2.address,
        nonce
      )
      computedAddress.should.equal(expectedAddress)
    })
    it('returns expected address, nonce: 999999999 ', async () => {
      const nonce = 999999999
      const expectedAddress = utils.getContractAddress({
        from: wallet1.address,
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        wallet1.address,
        nonce
      )
      computedAddress.should.equal(expectedAddress)
    })
    // test around nonce 128, or 0x80, due to edge cases. See https://github.com/ethereum/wiki/wiki/RLP#definition
    for (let nonce = 127; nonce < 129; nonce++) {
      it(`returns expected address, nonce: ${nonce}`, async () => {
        const expectedAddress = utils.getContractAddress({
          from: wallet1.address,
          nonce,
        })
        const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
          wallet1.address,
          nonce
        )
        computedAddress.should.equal(expectedAddress)
      })
    }
  })

  /*
   * Test buildCreate2Address helper function
   */
  describe('buildCreate2Address helper', async () => {
    for (const test of Object.keys(create2Tests)) {
      it(`should properly generate CREATE2 address from ${test}`, async () => {
        const { address, salt, init_code, result } = create2Tests[test]
        const computedAddress = buildCreate2Address(address, salt, init_code)
        computedAddress.should.equal(result.toLowerCase())
      })
    }
  })

  /*
   * Test getAddressFromCREATE2
   */
  describe('getAddressFromCREATE2', async () => {
    for (const test of Object.keys(create2Tests)) {
      it(`should properly generate CREATE2 address from ${test}`, async () => {
        const { address, salt, init_code, result } = create2Tests[test]
        const computedAddress = await contractAddressGenerator.getAddressFromCREATE2(
          address,
          salt,
          init_code
        )
        computedAddress.toLowerCase().should.equal(result.toLowerCase())
      })
    }
  })
})
