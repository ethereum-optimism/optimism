import { expect } from 'chai'
import assert = require('assert')
import { JsonRpcProvider, TransactionResponse } from '@ethersproject/providers'
import { BigNumber, Contract, Wallet, utils } from 'ethers'
import { getContractInterface } from '@metis.io/contracts'
import { Watcher } from '@eth-optimism/core-utils'
import dotenv = require('dotenv')
import * as path from 'path';

export const getEnvironment = async (): Promise<{
    l1Provider: JsonRpcProvider,
    l2Provider: JsonRpcProvider,
    l1Wallet: Wallet,
    l2Wallet: Wallet,
    AddressManager: Contract,
    watcher: Watcher
}> => {
    const l1Provider = new JsonRpcProvider("http://localhost:9545")
    const l2Provider = new JsonRpcProvider("http://localhost:8545")
    const l1Wallet = new Wallet("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", l1Provider)
    const l2Wallet = new Wallet("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", l2Provider)

    const addressManagerInterface = null
    const AddressManager = null
    const watcher = null
    

    return {
        l1Provider,
        l2Provider,
        l1Wallet,
        l2Wallet,
        AddressManager,
        watcher
    }
}


const MVM_GasOracle_ADDRESS = '0x420000000000000000000000000000000000AAAA'
const PROXY_SEQUENCER_ENTRYPOINT_ADDRESS = '0x4200000000000000000000000000000000000004'
const TAX_ADDRESS = '0x1234123412341234123412341234123412341234'

let l1Provider: JsonRpcProvider
let l2Provider: JsonRpcProvider
let l1Wallet: Wallet
let l2Wallet: Wallet
let AddressManager: Contract
let watcher: Watcher

describe('Fee Payment Integration Tests', async () => {
  const envPath = path.join(__dirname, '/.env');
  dotenv.config({ path: envPath })
  
  let OVM_L1ETHGateway: Contract
  let MVM_GasOracle: Contract
  let OVM_L2CrossDomainMessenger: Contract
  
  before(async () => {
    const system = await getEnvironment()
    l1Provider = system.l1Provider
    l2Provider = system.l2Provider
    l1Wallet = system.l1Wallet
    l2Wallet = system.l2Wallet
    
    const addressManagerAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
    const addressManagerInterface = getContractInterface('Lib_AddressManager')
    const AddressManager = new Contract(addressManagerAddress, addressManagerInterface, l1Provider)
    MVM_GasOracle = new Contract(
      MVM_GasOracle_ADDRESS,
      getContractInterface('MVM_GasOracle'),
      l2Wallet
    )
    
    console.log(
      await AddressManager.getAddress('Proxy__OVM_L1ETHGateway'),
      await AddressManager.getAddress('OVM_L1ERC20Gateway'),
      await AddressManager.getAddress('OVM_L2MessageRelayer')
    )
    const l1GatewayInterface = getContractInterface('OVM_L1ETHGateway')
    OVM_L1ETHGateway = new Contract(
      await AddressManager.getAddress('Proxy__OVM_L1ETHGateway'),
      l1GatewayInterface,
      l1Wallet
    )
    OVM_L2CrossDomainMessenger = new Contract(
      '0x4200000000000000000000000000000000000007',
      getContractInterface('OVM_L2CrossDomainMessenger'),
      l2Wallet
    )
  })

  beforeEach(async () => {
    
  })
  
  it.only('sequencer rejects transaction with a non-multiple-of-1M gasPrice', async () => {
    try {
      await MVM_GasOracle.setPrice(1000, {
        gasLimit: 8999999,
        gasPrice: 0})
      await MVM_GasOracle.transferSetter(l1Wallet.address, {
        gasLimit: 8999999,
        gasPrice: 0})
      console.log(await MVM_GasOracle)
    } catch (e) {
      console.error(e)
    }
  })
})
