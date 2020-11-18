import { expect } from '../setup'

import { ethers, Signer, Contract, ContractFactory, Wallet } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { getContractFactory, getContractInterface } from '@eth-optimism/contracts'

import { main } from '../../src'
import { ganache, wallets } from '../helpers/ganache'

const getStateRoot = async (provider: JsonRpcProvider): Promise<string> => {
  const proof = await provider.send('eth_getProof', [
    wallets[0].address,
    []
  ])

  return proof.stateRoot
}

const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}

const getMessageHash = (
  target: string,
  sender: string,
  message: string,
  messageNonce: number,
): string => {
  const iface = getContractInterface('OVM_L2CrossDomainMessenger')
  const encoded = iface.encodeFunctionData('relayMessage', [
    target,
    sender,
    message,
    messageNonce,
  ])

  return ethers.utils.keccak256(encoded)
}

describe('Message Relayer: basic tests', () => {
  let l1RpcProvider: JsonRpcProvider
  let l2RpcProvider: JsonRpcProvider
  before(async () => {
    const l1Server = ganache.server()
    const l2Server = ganache.server()

    await new Promise<void>((resolve) => {
      l1Server.listen(8545, null, null, () => {
        resolve()
      })
    })

    await new Promise<void>((resolve) => {
      l2Server.listen(8546, null, null, () => {
        resolve()
      })
    })

    l1RpcProvider = new JsonRpcProvider('http://localhost:8545')
    l2RpcProvider = new JsonRpcProvider('http://localhost:8546')
  })

  let l1DeployWallet: Wallet
  let l2DeployWallet: Wallet
  let l1RelayWallet: Wallet
  before(async () => {
    // L1 wallets.
    l1DeployWallet = wallets[0].connect(l1RpcProvider)
    l1RelayWallet = wallets[1].connect(l1RpcProvider)
  
    // L2 wallets.
    l2DeployWallet = wallets[2].connect(l2RpcProvider)
  })

  let Factory__Lib_AddressManager_L1: ContractFactory
  let Factory__Lib_AddressManager_L2: ContractFactory
  let Factory__OVM_BondManager: ContractFactory
  let Factory__OVM_CanonicalTransactionChain: ContractFactory
  let Factory__OVM_StateCommitmentChain: ContractFactory
  let Factory__OVM_L1CrossDomainMessenger: ContractFactory
  let Factory__OVM_L2CrossDomainMessenger: ContractFactory
  let Factory__OVM_L2ToL1MessagePasser: ContractFactory
  let Factory__OVM_L1MessageSender: ContractFactory
  before(async () => {
    // L1 factories.
    Factory__Lib_AddressManager_L1 = getContractFactory('Lib_AddressManager', l1DeployWallet)
    Factory__OVM_BondManager = getContractFactory('mockOVM_BondManager', l1DeployWallet)
    Factory__OVM_CanonicalTransactionChain = getContractFactory('mockOVM_CanonicalTransactionChain', l1DeployWallet)
    Factory__OVM_StateCommitmentChain = getContractFactory('OVM_StateCommitmentChain', l1DeployWallet)
    Factory__OVM_L1CrossDomainMessenger = getContractFactory('OVM_L1CrossDomainMessenger', l1DeployWallet)

    // L2 factories.
    Factory__Lib_AddressManager_L2 = getContractFactory('Lib_AddressManager', l2DeployWallet)
    Factory__OVM_L2CrossDomainMessenger = getContractFactory('OVM_L2CrossDomainMessenger', l2DeployWallet)
    Factory__OVM_L2ToL1MessagePasser = getContractFactory('OVM_L2ToL1MessagePasser', l2DeployWallet)
    Factory__OVM_L1MessageSender = getContractFactory('mockOVM_L1MessageSender', l2DeployWallet)
  })

  let Lib_AddressManager_L1: Contract
  let Lib_AddressManager_L2: Contract
  let OVM_BondManager: Contract
  let OVM_CanonicalTransactionChain: Contract
  let OVM_StateCommitmentChain: Contract
  let OVM_L1CrossDomainMessenger: Contract
  let OVM_L2CrossDomainMessenger: Contract
  let OVM_L2ToL1MessagePasser: Contract
  let OVM_L1MessageSender: Contract
  before(async () => {
    // L1 contract deployments.
    Lib_AddressManager_L1 = await Factory__Lib_AddressManager_L1.deploy()
    OVM_BondManager = await Factory__OVM_BondManager.deploy()
    OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy()
    OVM_StateCommitmentChain = await Factory__OVM_StateCommitmentChain.deploy(Lib_AddressManager_L1.address)
    OVM_L1CrossDomainMessenger = await Factory__OVM_L1CrossDomainMessenger.deploy()

    // L2 contract deployments.
    Lib_AddressManager_L2 = await Factory__Lib_AddressManager_L2.deploy()
    OVM_L2CrossDomainMessenger = await Factory__OVM_L2CrossDomainMessenger.deploy(Lib_AddressManager_L2.address)
    OVM_L2ToL1MessagePasser = await Factory__OVM_L2ToL1MessagePasser.deploy()
    OVM_L1MessageSender = await Factory__OVM_L1MessageSender.deploy()

    // L1 contract initializations.
    await OVM_StateCommitmentChain.init()
    await OVM_L1CrossDomainMessenger.initialize(Lib_AddressManager_L1.address)

    // L2 contract initializations.
    await OVM_L1MessageSender.setL1MessageSender(OVM_L1CrossDomainMessenger.address)
  })

  before(async () => {
    // L1 address manager initializations.
    await Lib_AddressManager_L1.setAddress('OVM_CanonicalTransactionChain', OVM_CanonicalTransactionChain.address)
    await Lib_AddressManager_L1.setAddress('OVM_StateCommitmentChain', OVM_StateCommitmentChain.address)
    await Lib_AddressManager_L1.setAddress('OVM_L1CrossDomainMessenger', OVM_L1CrossDomainMessenger.address)
    await Lib_AddressManager_L1.setAddress('OVM_L2CrossDomainMessenger', OVM_L2CrossDomainMessenger.address)
    await Lib_AddressManager_L1.setAddress('OVM_BondManager', OVM_BondManager.address)

    // L2 address manager initializations.
    await Lib_AddressManager_L2.setAddress('OVM_L1CrossDomainMessenger', OVM_L1CrossDomainMessenger.address)
    await Lib_AddressManager_L2.setAddress('OVM_L2ToL1MessagePasser', OVM_L2ToL1MessagePasser.address)
    await Lib_AddressManager_L2.setAddress('OVM_L1MessageSender', OVM_L1MessageSender.address)
  })

  before(async () => {
    /*
    main({
      l1RpcProvider: l1RpcProvider,
      l2RpcProvider: l2RpcProvider,
      stateCommitmentChainAddress: OVM_StateCommitmentChain.address,
      l1CrossDomainMessengerAddress: OVM_L1CrossDomainMessenger.address,
      l2CrossDomainMessengerAddress: OVM_L2CrossDomainMessenger.address,
      l2ChainStartingHeight: 0,
      pollingInterval: 15,
      relaySigner: relaySigner,
    })
    */
  })

  describe('basic complete tests', () => {
    it('should detect a single finalized transaction', async () => {
      await OVM_StateCommitmentChain.appendStateBatch(
        ['0x0000000000000000000000000000000000000000000000000000000000000000'],
        0
      )

      await OVM_L2CrossDomainMessenger.sendMessage(
        '0x0000000000000000000000000000000000000004',
        '0x1234123412341234',
        2000000,
        {
          from: l2DeployWallet.address,
        }
      )

      const root1 = await getStateRoot(l2RpcProvider)

      await OVM_StateCommitmentChain.appendStateBatch(
        [root1],
        1
      )

      await l2RpcProvider.send('evm_increaseTime', [864000])

      await sleep(10)

      expect(
        await OVM_L1CrossDomainMessenger.successfulMessages(
          getMessageHash(
            '0x0000000000000000000000000000000000000004',
            l2DeployWallet.address,
            '0x1234123412341234',
            0,
          )
        )
      ).to.be.true
    })
  })
})
