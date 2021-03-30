import { expect } from '../../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'
import { toHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  makeAddressManager,
  NON_ZERO_ADDRESS,
  NON_NULL_BYTES32,
  DUMMY_BATCH_HEADERS,
  DUMMY_BATCH_PROOFS,
} from '../../../../helpers'

describe('OVM_L1MultiMessageRelayer', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let Factory__OVM_L1MultiMessageRelayer: ContractFactory
  let Mock__OVM_L1CrossDomainMessenger: MockContract
  let messages: any[]

  before(async () => {
    // We do all the 'reusable setup' in here, ie. creating factories, mocks and setting addresses
    // for everything but the contract under test
    AddressManager = await makeAddressManager()

    // create a mock for the L1CrossDomainMessenger implementation
    Mock__OVM_L1CrossDomainMessenger = await smockit(
      await ethers.getContractFactory('OVM_L1CrossDomainMessenger')
    )

    // set the address of the mock contract to target
    await AddressManager.setAddress(
      // 'Proxy__OVM_L1CrossDomainMessenger' is the string used by the contract under test to lookup
      // the target contract. On mainnet the target is a proxy which points to the implementation of
      // the L1CrossDomainMessenger.
      // In order to keep the tests simple, we skip the proxy here, and point directly to the impl.
      'Proxy__OVM_L1CrossDomainMessenger',
      Mock__OVM_L1CrossDomainMessenger.address
    )

    // set the signer as the address required by access control
    await AddressManager.setAddress(
      'OVM_L2BatchMessageRelayer',
      signer.getAddress()
    )

    // define a dummy proof to satisfy the abi
    const dummyProof = {
      stateRoot: NON_NULL_BYTES32,
      stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
      stateRootProof: DUMMY_BATCH_PROOFS[0],
      stateTrieWitness: toHexString('some bytes'),
      storageTrieWitness: toHexString('some more bytes'),
    }

    // create a few dummy messages to relay
    const m1 = {
      target: '0x1100000000000000000000000000000000000000',
      message: NON_NULL_BYTES32,
      sender: '0x2200000000000000000000000000000000000000',
      messageNonce: 1,
      proof: dummyProof,
    }

    const m2 = {
      target: '0x1100000000000000000000000000000000000000',
      message: NON_NULL_BYTES32,
      sender: '0x2200000000000000000000000000000000000000',
      messageNonce: 2,
      proof: dummyProof,
    }

    const m3 = {
      target: '0x1100000000000000000000000000000000000000',
      message: NON_NULL_BYTES32,
      sender: '0x2200000000000000000000000000000000000000',
      messageNonce: 2,
      proof: dummyProof,
    }

    messages = [m1, m2, m3]
  })

  let OVM_L1MultiMessageRelayer: Contract

  beforeEach(async () => {
    // setup a factory and deploy a new test-contract for each unit test
    Factory__OVM_L1MultiMessageRelayer = await ethers.getContractFactory(
      'OVM_L1MultiMessageRelayer'
    )
    OVM_L1MultiMessageRelayer = await Factory__OVM_L1MultiMessageRelayer.deploy(
      AddressManager.address
    )

    // set the address of the OVM_L1MultiMessageRelayer, which the OVM_L1CrossDomainMessenger will
    // check in its onlyRelayer modifier.
    // The string currently used in the AddressManager is 'OVM_L2MessageRelayer'
    await AddressManager.setAddress(
      'OVM_L2MessageRelayer',
      OVM_L1MultiMessageRelayer.address
    )
    // set the mock return value
    Mock__OVM_L1CrossDomainMessenger.smocked.relayMessage.will.return()
  })

  describe('batchRelayMessages', () => {
    it('Successfully relay multiple messages', async () => {
      await OVM_L1MultiMessageRelayer.batchRelayMessages(messages)
      await expect(
        Mock__OVM_L1CrossDomainMessenger.smocked.relayMessage.calls.length
      ).to.deep.equal(messages.length)
    })

    it('should revert if called by the wrong account', async () => {
      // set the wrong address to use for ACL
      await AddressManager.setAddress(
        'OVM_L2BatchMessageRelayer',
        NON_ZERO_ADDRESS
      )

      await expect(
        OVM_L1MultiMessageRelayer.batchRelayMessages(messages)
      ).to.be.revertedWith(
        'OVM_L1MultiMessageRelayer: Function can only be called by the OVM_L2BatchMessageRelayer'
      )
    })
  })
})
