import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { utils, Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import { create2Tests } from '../../test-helpers/data/create2.test.json'
import { buildCreate2Address } from '../../test-helpers'

/* Tests */
describe('ContractAddressGenerator', () => {
  let wallet1: Signer
  let wallet2: Signer

  before(async () => {
    ;[wallet1, wallet2] = await ethers.getSigners()
  })

  let ContractAddressGenerator: ContractFactory
  beforeEach(async () => {
    ContractAddressGenerator = await ethers.getContractFactory(
      'ContractAddressGenerator'
    )
  })

  let contractAddressGenerator: Contract
  beforeEach(async () => {
    contractAddressGenerator = await ContractAddressGenerator.deploy()
  })

  describe('getAddressFromCREATE', async () => {
    it('returns expected address, nonce: 1', async () => {
      const nonce = 1
      const expectedAddress = utils.getContractAddress({
        from: await wallet1.getAddress(),
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        await wallet1.getAddress(),
        nonce
      )
      computedAddress.should.equal(expectedAddress)
    })

    it('returns expected address, nonce: 1, different origin address', async () => {
      const nonce = 1
      const expectedAddress = utils.getContractAddress({
        from: await wallet2.getAddress(),
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        await wallet2.getAddress(),
        nonce
      )
      computedAddress.should.equal(expectedAddress)
    })

    it('returns expected address, nonce: 999999999 ', async () => {
      const nonce = 999999999
      const expectedAddress = utils.getContractAddress({
        from: await wallet1.getAddress(),
        nonce,
      })
      const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
        await wallet1.getAddress(),
        nonce
      )
      computedAddress.should.equal(expectedAddress)
    })

    // test around nonce 128, or 0x80, due to edge cases. See https://github.com/ethereum/wiki/wiki/RLP#definition
    for (let nonce = 127; nonce < 129; nonce++) {
      it(`returns expected address, nonce: ${nonce}`, async () => {
        const expectedAddress = utils.getContractAddress({
          from: await wallet1.getAddress(),
          nonce,
        })
        const computedAddress = await contractAddressGenerator.getAddressFromCREATE(
          await wallet1.getAddress(),
          nonce
        )
        computedAddress.should.equal(expectedAddress)
      })
    }
  })

  describe('buildCreate2Address helper', async () => {
    for (const test of Object.keys(create2Tests)) {
      it(`should properly generate CREATE2 address from ${test}`, async () => {
        const { address, salt, init_code, result } = create2Tests[test]
        const computedAddress = buildCreate2Address(address, salt, init_code)
        computedAddress.should.equal(result.toLowerCase())
      })
    }
  })

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
