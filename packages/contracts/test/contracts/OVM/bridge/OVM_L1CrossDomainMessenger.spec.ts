import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Signer, ContractFactory, Contract, BigNumber } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  NON_NULL_BYTES32,
  ZERO_ADDRESS,
  NON_ZERO_ADDRESS,
  NULL_BYTES32,
  DUMMY_BATCH_HEADERS,
  DUMMY_BATCH_PROOFS,
  TrieTestGenerator,
  toHexString,
  getNextBlockNumber,
  remove0x,
} from '../../../helpers'
import { getContractInterface } from '../../../../src'
import { keccak256 } from 'ethers/lib/utils'

const getXDomainCalldata = (
  sender: string,
  target: string,
  message: string,
  messageNonce: number
): string => {
  return getContractInterface(
    'OVM_L2CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [target, sender, message, messageNonce])
}

const deployProxyXDomainMessenger = async (
  addressManager: Contract,
  l1XDomainMessenger: Contract
): Promise<Contract> => {
  await addressManager.setAddress(
    'OVM_L1CrossDomainMessenger',
    l1XDomainMessenger.address
  )
  const proxy = await (
    await ethers.getContractFactory('Lib_ResolvedDelegateProxy')
  ).deploy(addressManager.address, 'OVM_L1CrossDomainMessenger')
  return l1XDomainMessenger.attach(proxy.address)
}

describe('OVM_L1CrossDomainMessenger', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__TargetContract: MockContract
  let Mock__OVM_L2CrossDomainMessenger: MockContract
  let Mock__OVM_CanonicalTransactionChain: MockContract
  let Mock__OVM_StateCommitmentChain: MockContract
  before(async () => {
    Mock__TargetContract = smockit(
      await ethers.getContractFactory('Helper_SimpleProxy')
    )
    Mock__OVM_L2CrossDomainMessenger = smockit(
      await ethers.getContractFactory('OVM_L2CrossDomainMessenger')
    )
    Mock__OVM_CanonicalTransactionChain = smockit(
      await ethers.getContractFactory('OVM_CanonicalTransactionChain')
    )
    Mock__OVM_StateCommitmentChain = smockit(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    await AddressManager.setAddress(
      'OVM_L2CrossDomainMessenger',
      Mock__OVM_L2CrossDomainMessenger.address
    )

    await setProxyTarget(
      AddressManager,
      'OVM_CanonicalTransactionChain',
      Mock__OVM_CanonicalTransactionChain
    )
    await setProxyTarget(
      AddressManager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain
    )
  })

  let Factory__OVM_L1CrossDomainMessenger: ContractFactory
  before(async () => {
    Factory__OVM_L1CrossDomainMessenger = await ethers.getContractFactory(
      'OVM_L1CrossDomainMessenger'
    )
  })

  let OVM_L1CrossDomainMessenger: Contract
  beforeEach(async () => {
    const xDomainMessenerImpl = await Factory__OVM_L1CrossDomainMessenger.deploy()
    // We use an upgradable proxy for the XDomainMessenger--deploy & set up the proxy.
    OVM_L1CrossDomainMessenger = await deployProxyXDomainMessenger(
      AddressManager,
      xDomainMessenerImpl
    )
    await OVM_L1CrossDomainMessenger.initialize(AddressManager.address)
  })

  describe('sendMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const gasLimit = 100_000

    it('should be able to send a single message', async () => {
      await expect(
        OVM_L1CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted

      expect(
        Mock__OVM_CanonicalTransactionChain.smocked.enqueue.calls[0]
      ).to.deep.equal([
        Mock__OVM_L2CrossDomainMessenger.address,
        BigNumber.from(gasLimit),
        getXDomainCalldata(await signer.getAddress(), target, message, 0),
      ])
    })

    it('should be able to send the same message twice', async () => {
      await OVM_L1CrossDomainMessenger.sendMessage(target, message, gasLimit)

      await expect(
        OVM_L1CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted
    })
  })

  describe('replayMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const gasLimit = 100_000

    it('should revert if the message does not exist', async () => {
      await expect(
        OVM_L1CrossDomainMessenger.replayMessage(
          target,
          await signer.getAddress(),
          message,
          0,
          gasLimit
        )
      ).to.be.revertedWith('Provided message has not already been sent.')
    })

    it('should succeed if the message exists', async () => {
      await OVM_L1CrossDomainMessenger.sendMessage(target, message, gasLimit)

      await expect(
        OVM_L1CrossDomainMessenger.replayMessage(
          target,
          await signer.getAddress(),
          message,
          0,
          gasLimit
        )
      ).to.not.be.reverted
    })
  })

  describe('relayMessage', () => {
    let target: string
    let message: string
    let sender: string
    let proof: any
    let calldata: string
    before(async () => {
      target = Mock__TargetContract.address
      message = Mock__TargetContract.interface.encodeFunctionData('setTarget', [
        NON_ZERO_ADDRESS,
      ])
      sender = await signer.getAddress()

      calldata = getXDomainCalldata(sender, target, message, 0)

      const precompile = '0x4200000000000000000000000000000000000000'

      const storageKey = keccak256(
        keccak256(
          calldata + remove0x(Mock__OVM_L2CrossDomainMessenger.address)
        ) + '00'.repeat(32)
      )
      const storageGenerator = await TrieTestGenerator.fromNodes({
        nodes: [
          {
            key: storageKey,
            val: '0x' + '01'.padStart(64, '0'),
          },
        ],
        secure: true,
      })

      const generator = await TrieTestGenerator.fromAccounts({
        accounts: [
          {
            address: precompile,
            nonce: 0,
            balance: 0,
            codeHash: keccak256('0x1234'),
            storageRoot: toHexString(storageGenerator._trie.root),
          },
        ],
        secure: true,
      })

      proof = {
        stateRoot: toHexString(generator._trie.root),
        stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
        stateRootProof: DUMMY_BATCH_PROOFS[0],
        stateTrieWitness: (await generator.makeAccountProofTest(precompile))
          .accountTrieWitness,
        storageTrieWitness: (
          await storageGenerator.makeInclusionProofTest(storageKey)
        ).proof,
      }
    })

    beforeEach(async () => {
      Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
        true
      )
      Mock__OVM_StateCommitmentChain.smocked.insideFraudProofWindow.will.return.with(
        false
      )
    })

    it('should revert if still inside the fraud proof window', async () => {
      Mock__OVM_StateCommitmentChain.smocked.insideFraudProofWindow.will.return.with(
        true
      )

      const proof = {
        stateRoot: NULL_BYTES32,
        stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
        stateRootProof: DUMMY_BATCH_PROOFS[0],
        stateTrieWitness: '0x',
        storageTrieWitness: '0x',
      }

      await expect(
        OVM_L1CrossDomainMessenger.relayMessage(
          target,
          sender,
          message,
          0,
          proof
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should revert if provided an invalid state root proof', async () => {
      Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
        false
      )

      const proof = {
        stateRoot: NULL_BYTES32,
        stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
        stateRootProof: DUMMY_BATCH_PROOFS[0],
        stateTrieWitness: '0x',
        storageTrieWitness: '0x',
      }

      await expect(
        OVM_L1CrossDomainMessenger.relayMessage(
          target,
          sender,
          message,
          0,
          proof
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should revert if provided an invalid storage trie witness', async () => {
      await expect(
        OVM_L1CrossDomainMessenger.relayMessage(target, sender, message, 0, {
          ...proof,
          storageTrieWitness: '0x',
        })
      ).to.be.reverted
    })

    it('should revert if provided an invalid state trie witness', async () => {
      await expect(
        OVM_L1CrossDomainMessenger.relayMessage(target, sender, message, 0, {
          ...proof,
          stateTrieWitness: '0x',
        })
      ).to.be.reverted
    })

    it('should send a successful call to the target contract', async () => {
      const blockNumber = await getNextBlockNumber(ethers.provider)

      await OVM_L1CrossDomainMessenger.relayMessage(
        target,
        sender,
        message,
        0,
        proof
      )

      expect(
        await OVM_L1CrossDomainMessenger.successfulMessages(keccak256(calldata))
      ).to.equal(true)

      expect(
        await OVM_L1CrossDomainMessenger.relayedMessages(
          keccak256(
            calldata +
              remove0x(await signer.getAddress()) +
              remove0x(BigNumber.from(blockNumber).toHexString()).padStart(
                64,
                '0'
              )
          )
        )
      ).to.equal(true)
    })

    it('should revert if trying to send the same message twice', async () => {
      await OVM_L1CrossDomainMessenger.relayMessage(
        target,
        sender,
        message,
        0,
        proof
      )

      await expect(
        OVM_L1CrossDomainMessenger.relayMessage(
          target,
          sender,
          message,
          0,
          proof
        )
      ).to.be.revertedWith('Provided message has already been received.')
    })
  })
})
