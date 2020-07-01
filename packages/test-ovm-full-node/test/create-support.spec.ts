import './setup'

/* External Imports */
import { add0x } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { addHandlerToProvider } from '@eth-optimism/rollup-full-node'
import { Contract, Wallet } from 'ethers'
<<<<<<< HEAD
import {
  getAddress,
  keccak256,
  solidityPack
} from 'ethers/utils'
=======
import { getAddress, keccak256, solidityPack } from 'ethers/utils'
>>>>>>> master

/* Contract Imports */
import * as SimpleCreate2 from '../build/SimpleCreate2.json'
import * as SimpleStorage from '../build/SimpleStorage.json'

const getCreate2Address = (
  factoryAddress: string,
  salt: string,
  bytecode: string
): string => {
<<<<<<< HEAD
  const create2Inputs = [
    '0xff',
    factoryAddress,
    salt,
    keccak256(bytecode)
  ]
  const sanitizedInputs = `0x${create2Inputs.map(i => i.slice(2)).join('')}`
=======
  const create2Inputs = ['0xff', factoryAddress, salt, keccak256(bytecode)]
  const sanitizedInputs = `0x${create2Inputs.map((i) => i.slice(2)).join('')}`
>>>>>>> master
  return getAddress(`0x${keccak256(sanitizedInputs).slice(-40)}`)
}

describe('Create2', () => {
  let wallet
  let simpleCreate2: Contract
<<<<<<< HEAD
  let provider 
  const DEFAULT_SALT = '0x1234123412341234123412341234123412341234123412341234123412341234'
=======
  let provider
  const DEFAULT_SALT =
    '0x1234123412341234123412341234123412341234123412341234123412341234'
>>>>>>> master

  beforeEach(async () => {
    provider = await createMockProvider()
    if (process.env.MODE === 'OVM') {
      provider = await addHandlerToProvider(provider)
    }
    const wallets = await getWallets(provider)
<<<<<<< HEAD
    const wallet = wallets[0]
    simpleCreate2 = await deployContract(wallet, SimpleCreate2, [])
  })

  // TODO uncomment this once ovmCREATE2 is fixed!
  // it('should calculate address correctly for invalid bytecode', async () => {
  //   const bytecode = '0x00'
  //   const salt = DEFAULT_SALT
  //   const tx = await simpleCreate2.create2(bytecode, salt)
  //   const receipt = await provider.getTransactionReceipt(tx.hash)
  //   const address = await simpleCreate2.contractAddress()
  //   const expectedAddress = getCreate2Address(simpleCreate2.address, salt, bytecode)
  //   address.should.equal(expectedAddress)
  // })
=======
    wallet = wallets[0]
    simpleCreate2 = await deployContract(wallet, SimpleCreate2, [])
  })

  // TODO unskip this once ovmCREATE2 is fixed with YAS-473!
  it.skip('should calculate address correctly for invalid bytecode', async () => {
    const bytecode = '0x00'
    const salt = DEFAULT_SALT
    await simpleCreate2.create2(bytecode, salt)
    const address = await simpleCreate2.contractAddress()
    const expectedAddress = getCreate2Address(
      simpleCreate2.address,
      salt,
      bytecode
    )
    address.should.equal(expectedAddress)
  })
>>>>>>> master

  it('should calculate address correctly for valid OVM bytecode', async () => {
    const bytecode = add0x(SimpleStorage.bytecode)
    const salt = DEFAULT_SALT
<<<<<<< HEAD
    const tx = await simpleCreate2.create2(bytecode, salt)
    const receipt = await provider.getTransactionReceipt(tx.hash)
    const address = await simpleCreate2.contractAddress()
    const expectedAddress = getCreate2Address(simpleCreate2.address, salt, bytecode)
    address.should.equal(expectedAddress)
  })
})

=======
    await simpleCreate2.create2(bytecode, salt)
    const address = await simpleCreate2.contractAddress()
    const expectedAddress = getCreate2Address(
      simpleCreate2.address,
      salt,
      bytecode
    )
    address.should.equal(expectedAddress)
  })
})
>>>>>>> master
