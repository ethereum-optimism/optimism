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
import * as SimpleCreate2 from '../build/SimpleCreate2.json'

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

describe.only('Create2', () => {
  let wallet
  let simpleCreate2: Contract
  let provider 
  const ZERO_HASH = add0x('00'.repeat(32))

  beforeEach(async () => {
    provider = await createMockProvider()
    if (process.env.MODE === 'OVM') {
      provider = await addHandlerToProvider(provider)
    }
    const wallets = await getWallets(provider)
    const wallet = wallets[0]
    simpleCreate2 = await deployContract(wallet, SimpleCreate2, [])
  })

  it('should calculateAddress correctly', async () => {
    const bytecode = '0x00'
    const salt = ZERO_HASH
    const tx = await simpleCreate2.create2(bytecode, salt)
    const receipt = await provider.getTransactionReceipt(tx.hash)
    const address = await simpleCreate2.contractAddress()
    const expectedAddress = getCreate2Address(simpleCreate2.address, salt, bytecode)
    address.should.equal(expectedAddress)
  })
  
})

