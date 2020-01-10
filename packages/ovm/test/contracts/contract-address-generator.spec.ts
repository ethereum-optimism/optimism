import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger } from '@pigi/core-utils'
import { utils } from 'ethers'

/* Contract Imports */
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

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
    // First deploy the execution manager
    contractAddressGenerator = await deployContract(
      wallet1,
      ContractAddressGenerator,
      [rlpEncode.address],
      {
        gasLimit: 6700000,
      }
    )
  })

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
      const nonce = 999999999
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
  describe('getAddressFromCREATE2', async () => {
    it('returns expected values', async () => {
      // TODO: Write this test.
      // Note you can find an example generating the address here: https://github.com/miguelmota/solidity-create2-example
    })
  })
})
