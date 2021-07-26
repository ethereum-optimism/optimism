import { injectL2Context } from '@eth-optimism/core-utils'
import { getContractInterface, getContractFactory } from '@eth-optimism/contracts'
import {
  Contract,
  Wallet,
  constants,
  providers,
  BigNumber,
  utils,
} from 'ethers'

import * as request from "request-promise-native";
import { expectEvent } from '@openzeppelin/test-helpers'

require('dotenv').config()

export const GWEI = BigNumber.from(0)

if(!process.env.L1_NODE_WEB3_URL) {
  console.log(`!!You did not set process.env.L1_NODE_WEB3_URL!!`)
  console.log(`Setting to default value of http://localhost:9545`)
} else {
  console.log(`process.env.L1_NODE_WEB3_URL set to:`,process.env.L1_NODE_WEB3_URL)
}

export const L1_NODE_WEB3_URL = process.env.L1_NODE_WEB3_URL || 'http://localhost:9545'

if(!process.env.L2_NODE_WEB3_URL) {
  console.log(`!!You did not set process.env.L2_NODE_WEB3_URL!!`)
  console.log(`Setting to default value of http://localhost:8545`)
} else {
  console.log(`process.env.L2_NODE_WEB3_URL set to:`,process.env.L2_NODE_WEB3_URL)
}

export const L2_NODE_WEB3_URL = process.env.L2_NODE_WEB3_URL || 'http://localhost:8545'

// The hardhat instance
export const l1Provider = new providers.JsonRpcProvider(L1_NODE_WEB3_URL)
export const l2Provider = new providers.JsonRpcProvider(L2_NODE_WEB3_URL)

// An account for testing which is funded on L1
export const bobl1Wallet = new Wallet(process.env.TEST_PRIVATE_KEY_1,l1Provider)
export const bobl2Wallet = bobl1Wallet.connect(l2Provider)

// The second test user with some eth
export const alicel1Wallet = new Wallet(process.env.TEST_PRIVATE_KEY_2).connect(l1Provider)
export const alicel2Wallet = new Wallet(process.env.TEST_PRIVATE_KEY_2).connect(l2Provider)

// The third test user with some eth
export const katel1Wallet = new Wallet(process.env.TEST_PRIVATE_KEY_3).connect(l1Provider)
export const katel2Wallet = new Wallet(process.env.TEST_PRIVATE_KEY_3).connect(l2Provider)

// Predeploys
export const PROXY_SEQUENCER_ENTRYPOINT_ADDRESS = '0x4200000000000000000000000000000000000004'
export const OVM_ETH_ADDRESS = '0x4200000000000000000000000000000000000006'
export const Proxy__OVM_L2CrossDomainMessenger = '0x4200000000000000000000000000000000000007'

export const DEPLOYER = process.env.URL || 'http://127.0.0.1:8080/addresses.json'
export const OMGX_URL = process.env.OMGX_URL || 'http://127.0.0.1:8078/addresses.json'

if(!process.env.URL) {
  console.log(`!!You did not set process.env.URL!!`)
  console.log(`Setting to default value of http://127.0.0.1:8080/addresses.json`)
} else {
  console.log(`process.env.URL set to:`,process.env.URL)
}

if(!process.env.OMGX_URL) {
  console.log(`!!You did not set process.env.OMGX_URL!!`)
  console.log(`Setting to default value of http://127.0.0.1:8078/addresses.json`)
} else {
  console.log(`process.env.OMGX_URL set to:`,process.env.OMGX_URL)
}

export let addressManagerAddress = process.env.ADDRESS_MANAGER_ADDRESS
export const getAddressManager = async (provider: any) => {
   //console.log(addressManagerAddress)
   if (addressManagerAddress){
     console.log(`ADDRESS_MANAGER_ADDRESS var was set`)
     return getContractFactory('Lib_AddressManager')
    .connect(provider)
    .attach(addressManagerAddress) as any
   } else {
     console.log(`ADDRESS_MANAGER_ADDRESS var was left unset. Using {$DEPLOYER} response`)
     addressManagerAddress = (await getDeployerAddresses()).AddressManager
     return getContractFactory('Lib_AddressManager')
     .connect(provider)
     .attach(addressManagerAddress) as any
  }
}

export const getDeployerAddresses = async () => {
  var options = {
    uri: DEPLOYER,
  }
  const result = await request.get(options)
  return JSON.parse(result)
 }

export const getOMGXDeployerAddresses = async () => {
  var options = {
    uri: OMGX_URL,
  }
  const result = await request.get(options)
  return JSON.parse(result)
}

export const expectLogs = async (receipt, emitterAbi, emitterAddress, eventName, eventArgs = {}) => {
  let eventABI = emitterAbi.filter(x => x.type === 'event' && x.name === eventName);
  if (eventABI.length === 0) {
    throw new Error(`No ABI entry for event '${eventName}'`);
  } else if (eventABI.length > 1) {
    throw new Error(`Multiple ABI entries for event '${eventName}', only uniquely named events are supported`);
  }

  eventABI = eventABI[0];
  const eventSignature = `${eventName}(${eventABI.inputs.map(input => input.type).join(',')})`;
  const eventTopic = utils.keccak256(utils.toUtf8Bytes(eventSignature))
  const logs = receipt.logs
  const abiCoder = new utils.AbiCoder();
  const filterdLogs = logs.filter(log => log.topics.length > 0 && log.topics[0] === eventTopic && (!emitterAddress || log.address === emitterAddress))
  .map(log => abiCoder.decode(eventABI.inputs, log.data, log.topics.slice(1)))
  .map(decoded => ({ event: eventName, args: decoded }));

  return expectEvent.inLogs(filterdLogs, eventName, eventArgs);

}

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
