'use strict'

import * as L2CrossDomainMessengerArtifact from '../../build/ovm_artifacts/L2CrossDomainMessenger.json'
import * as CrossDomainSimpleStorageArtifact from '../../build/ovm_artifacts/CrossDomainSimpleStorage.json'
import { ethers, Wallet, ContractFactory, Contract } from 'ethers'

import {
  deployAndRegister,
  getContractInterface,
  getContractFactory,
} from '../'

const l1Url = process.env.L1_URL
const l2Url = process.env.L2_URL
const addressResolverContractAddress =
  process.env.L1_ADDRESS_RESOLVER_CONTRACT_ADDRESS
const l1Provider = new ethers.providers.JsonRpcProvider(l1Url)
const l2Provider = new ethers.providers.JsonRpcProvider(l2Url)

const l1Owner = new Wallet(process.env.L1_PRIVATE_KEY, l1Provider)

const l2Owner = new Wallet(process.env.L2_PRIVATE_KEY, l2Provider)

const godWalletAddress = process.env.GOD_ADDRESS

const deployMessengers = async () => {
  const AddressResolver = new Contract(
    addressResolverContractAddress,
    getContractInterface('AddressResolver'),
    l1Owner
  )

  const L1toL2TransactionQueueFactory = await getContractFactory(
    'L1ToL2TransactionQueue'
  )

  console.log(`deploying L1ToL2TransactionQueue...`)

  const L1ToL2TransactionQueue = await deployAndRegister(
    AddressResolver,
    'L1toL2TransactionQueue',
    {
      factory: L1toL2TransactionQueueFactory.connect(l1Owner),
      params: [AddressResolver.address],
      signer: l1Owner,
    }
  )

  console.log(
    `deployed L1ToL2TransactionQueue at:`,
    L1ToL2TransactionQueue.address
  )

  const L1CrossDomainMessengerFactory = await getContractFactory(
    'L1CrossDomainMessenger'
  )

  console.log(`deploying L1CrossDomainMessenger...`)

  const L1CrossDomainMessenger = await deployAndRegister(
    AddressResolver,
    'L1CrossDomainMessenger',
    {
      factory: L1CrossDomainMessengerFactory.connect(l1Owner),
      params: [AddressResolver.address],
      signer: l1Owner,
    }
  )
  console.log(
    `deployed L1CrossDomainMessenger at:`,
    L1CrossDomainMessenger.address
  )

  const L2CrossDomainMessengerFactory = new ContractFactory(
    L2CrossDomainMessengerArtifact.abi,
    L2CrossDomainMessengerArtifact.bytecode,
    l2Owner
  )

  console.log(`deploying L2CrossDomainMessenger...`)

  const L2CrossDomainMessenger = await L2CrossDomainMessengerFactory.connect(
    l2Owner
  ).deploy(
    '0x4200000000000000000000000000000000000001', //L1 message sender precompile
    '0x4200000000000000000000000000000000000000' //L2 To L1 Message Passer Precompile
  )
  console.log(
    `deployed L2CrossDomainMessenger at:`,
    L2CrossDomainMessenger.address
  )

  const l2SetTargetTx = await L2CrossDomainMessenger.connect(
    l2Owner
  ).setTargetMessengerAddress(L1CrossDomainMessenger.address)
  console.log(
    'Set L2 target address to ',
    L1CrossDomainMessenger.address,
    'w/ tx hash:',
    l2SetTargetTx.hash
  )

  const l1SetTargetTx = await L1CrossDomainMessenger.connect(
    l1Owner
  ).setTargetMessengerAddress(L2CrossDomainMessenger.address)
  await l1Provider.waitForTransaction(l1SetTargetTx.hash)
  console.log(
    'Set L1 target address to ',
    L2CrossDomainMessenger.address,
    'w/ tx hash:',
    l1SetTargetTx.hash
  )

  console.log('\n~~~~~~~~~~~~~~~ Time for some temp initing ~~~~~~~~~~~~~~~\n')

  // INIT L1TOL2TXQUEUE
  const tempInitQueue = await L1ToL2TransactionQueue.tempInit(
    L1CrossDomainMessenger.address
  )
  console.log(
    'tempInit-ed the L1ToL2TransactionQueue with L1crossdomainMessenger',
    L1CrossDomainMessenger.address,
    'txHash',
    tempInitQueue.hash,
    '\n'
  )
  await l1Provider.waitForTransaction(tempInitQueue.hash)

  // INIT L1 MESSENGER
  const tempInitL1Messenger = await L1CrossDomainMessenger.tempInit(
    L1ToL2TransactionQueue.address
  )
  console.log(
    'tempInit-ed the L1CrossDomainMessenger with L1ToL2TransactionQueue',
    L1ToL2TransactionQueue.address,
    'txHash',
    tempInitL1Messenger.hash,
    '\n'
  )
  await l1Provider.waitForTransaction(tempInitL1Messenger.hash)

  // INIT L2 MESSENGER
  const tempInitL2Messenger = await L2CrossDomainMessenger.tempInit(
    godWalletAddress
  )

  console.log(
    'tempInit-ed the L2CrossDomainMessenger with God Wallet Address',
    godWalletAddress,
    'txHash',
    tempInitL2Messenger.hash,
    '\n'
  )
  console.log(
    '\n~~~~~~~~~~~~~~~ Time to deploy L2 Contract and test a deploy! ~~~~~~~~~~~~~~~\n'
  )

  const CrossDomainSimpleStorageFactory = new ContractFactory(
    CrossDomainSimpleStorageArtifact.abi,
    CrossDomainSimpleStorageArtifact.bytecode,
    l2Owner
  )

  const CrossDomainSimpleStorage = await CrossDomainSimpleStorageFactory.connect(
    l2Owner
  ).deploy()
  console.log(
    `deployed CrossDomainSimpleStorage at:`,
    CrossDomainSimpleStorage.address
  )
  console.log()

  const xDomainTx = await L1CrossDomainMessenger.connect(l1Owner).sendMessage(
    CrossDomainSimpleStorage.address, //target address
    '0x0348299d42424242424242424242424242424242424242424242424242424242424242429999999999999999999999999999999999999999999999999999999999999999', //calldata
    1000000,
    { gasLimit: 1000000 }
  )
  await new Promise((r) => setTimeout(r, 10000))
  const xDomainMsgSender = await CrossDomainSimpleStorage.connect(
    l2Owner
  ).crossDomainMsgSender()
  console.log('got xdomain message sender', xDomainMsgSender)
}
;(async () => {
  await deployMessengers()
})()
