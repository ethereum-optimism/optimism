import '../../common/setup'

/* External Imports */
import { add0x } from '@eth-optimism/core-utils'
import { deployContract } from 'ethereum-waffle-v2'
import { Contract, Wallet } from 'ethers-v4'
import { getAddress, keccak256 } from 'ethers-v4/utils'

/* Internal Imports */
import { waffleV2 } from '../../../src/waffle/waffle-v2'

/* Contract Imports */
import * as SimpleCreate2 from '../../temp/build/waffle/SimpleCreate2.json'
import * as SimpleStorage from '../../temp/build/waffle/SimpleStorage.json'

const getCreate2Address = (
  factoryAddress: string,
  salt: string,
  bytecode: string
): string => {
  const create2Inputs = ['0xff', factoryAddress, salt, keccak256(bytecode)]
  const sanitizedInputs = `0x${create2Inputs.map((i) => i.slice(2)).join('')}`
  return getAddress(`0x${keccak256(sanitizedInputs).slice(-40)}`)
}

const overrides = {
  gasLimit: 20000000,
}
const DEFAULT_SALT =
  '0x1234123412341234123412341234123412341234123412341234123412341234'

describe('Create2 Support', () => {
  let wallet: Wallet
  let provider: any
  before(async () => {
    provider = new waffleV2.MockProvider(overrides)
    ;[wallet] = provider.getWallets()
  })

  let simpleCreate2: Contract
  beforeEach(async () => {
    simpleCreate2 = await deployContract(wallet, SimpleCreate2, [], overrides)
  })

  it('should calculate address correctly for invalid bytecode', async () => {
    const bytecode = '0x00'
    const salt = DEFAULT_SALT

    await simpleCreate2.create2(bytecode, salt, overrides)
    const address = await simpleCreate2.contractAddress()
    const expectedAddress = getCreate2Address(
      simpleCreate2.address,
      salt,
      bytecode
    )

    address.should.equal(expectedAddress)
  })

  it('should calculate address correctly for valid OVM bytecode', async () => {
    const bytecode = add0x(SimpleStorage.bytecode)
    const salt = DEFAULT_SALT

    await simpleCreate2.create2(bytecode, salt, overrides)
    const address = await simpleCreate2.contractAddress()
    const expectedAddress = getCreate2Address(
      simpleCreate2.address,
      salt,
      bytecode
    )

    address.should.equal(expectedAddress)
  })
})
