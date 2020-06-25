import './setup'

/* External Imports */
import { add0x } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { addHandlerToProvider } from '@eth-optimism/rollup-full-node'
import { Contract, Wallet } from 'ethers'
import {
  getAddress,
  keccak256,
  solidityPack
} from 'ethers/utils'

/* Contract Imports */
import * as SimpleConstantCreate2 from '../build/SimpleConstantCreate2.json'
import * as SimpleStorage from '../build/SimpleStorage.json'

const getCreate2Address = (
  factoryAddress: string,
  salt: string,
  bytecode: string
): string => {
  const create2Inputs = [
    '0xff',
    factoryAddress,
    salt,
    keccak256(bytecode)
  ]
  const sanitizedInputs = `0x${create2Inputs.map(i => i.slice(2)).join('')}`
  return getAddress(`0x${keccak256(sanitizedInputs).slice(-40)}`)
}

describe.only('Large Constant deployment', () => {
  let wallet
  let simpleConstantCreate2: Contract
  let provider 
  const DEFAULT_SALT = '0x1234123412341234123412341234123412341234123412341234123412341234'

  beforeEach(async () => {
    provider = await createMockProvider()
    if (process.env.MODE === 'OVM') {
      provider = await addHandlerToProvider(provider)
    }
    const wallets = await getWallets(provider)
    const wallet = wallets[0]
    simpleConstantCreate2 = await deployContract(wallet, SimpleConstantCreate2, [])
  })

  it('should calculate address correctly for valid OVM bytecode in a', async () => {
    const salt = DEFAULT_SALT
    const bytecode = add0x(SimpleStorage.bytecode)
    await simpleConstantCreate2.create2(salt)
    const address = await simpleConstantCreate2.contractAddress()
    const expectedAddress = getCreate2Address(simpleConstantCreate2.address, salt, bytecode)
    address.should.equal(expectedAddress)
  })
})

