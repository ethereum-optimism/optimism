import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger } from '@pigi/core-utils'
import { utils } from 'ethers'

/* Contract Imports */
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'

const log = getLogger('contract-address-generator', true)

/*********
 * TESTS *
 *********/

describe('ContractAddressGenerator', () => {
  const [wallet1] = getWallets(createMockProvider())
  // Create pointers to our contractAddressGenerator
  let contractAddressGenerator

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // First deploy the execution manager
    contractAddressGenerator = await deployContract(
      wallet1,
      ContractAddressGenerator,
      [],
      {
        gasLimit: 6700000,
      }
    )
  })

  describe.skip('getAddressFromCREATE', async () => {
    it('returns expected values simple address', async () => {
      const nonce = 1
      const expectedAddress = utils.getContractAddress({
        from: wallet1.address,
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        wallet1.address,
        nonce
      )
      // Check that they are equal
      computedAddress.should.equal(expectedAddress)
      // TODO: Fix this test so it actually works!
    })
  })

  describe.skip('getAddressFromCREATE2', async () => {
    it('returns expected values', async () => {
      // TODO: Write this test.
      // Note you can find an example generating the address here: https://github.com/miguelmota/solidity-create2-example
    })
  })
})
