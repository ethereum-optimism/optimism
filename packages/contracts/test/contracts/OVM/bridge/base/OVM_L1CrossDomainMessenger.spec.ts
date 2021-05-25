import { expect } from '../../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, BigNumber } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'
import { remove0x, toHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  NON_NULL_BYTES32,
  NON_ZERO_ADDRESS,
  DUMMY_BATCH_HEADERS,
  DUMMY_BATCH_PROOFS,
  FORCE_INCLUSION_PERIOD_SECONDS,
  FORCE_INCLUSION_PERIOD_BLOCKS,
  TrieTestGenerator,
  getNextBlockNumber,
  encodeXDomainCalldata,
} from '../../../../helpers'
import { keccak256 } from 'ethers/lib/utils'
import { predeploys } from '../../../../../src'

const MAX_GAS_LIMIT = 8_000_000

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
  let signer2: Signer
  before(async () => {
    ;[signer, signer2] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__TargetContract: MockContract
  let Mock__OVM_L2CrossDomainMessenger: MockContract
  let Mock__OVM_StateCommitmentChain: MockContract

  let Factory__OVM_CanonicalTransactionChain: ContractFactory
  let Factory__OVM_ChainStorageContainer: ContractFactory
  let Factory__OVM_L1CrossDomainMessenger: ContractFactory

  let OVM_CanonicalTransactionChain: Contract
  before(async () => {
    Mock__TargetContract = await smockit(
      await ethers.getContractFactory('Helper_SimpleProxy')
    )
    Mock__OVM_L2CrossDomainMessenger = await smockit(
      await ethers.getContractFactory('OVM_L2CrossDomainMessenger')
    )
    Mock__OVM_StateCommitmentChain = await smockit(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    await AddressManager.setAddress(
      'OVM_L2CrossDomainMessenger',
      Mock__OVM_L2CrossDomainMessenger.address
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain
    )

    Factory__OVM_CanonicalTransactionChain = await ethers.getContractFactory(
      'OVM_CanonicalTransactionChain'
    )

    Factory__OVM_ChainStorageContainer = await ethers.getContractFactory(
      'OVM_ChainStorageContainer'
    )

    Factory__OVM_L1CrossDomainMessenger = await ethers.getContractFactory(
      'OVM_L1CrossDomainMessenger'
    )
    OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS,
      FORCE_INCLUSION_PERIOD_BLOCKS,
      MAX_GAS_LIMIT
    )

    const batches = await Factory__OVM_ChainStorageContainer.deploy(
      AddressManager.address,
      'OVM_CanonicalTransactionChain'
    )
    const queue = await Factory__OVM_ChainStorageContainer.deploy(
      AddressManager.address,
      'OVM_CanonicalTransactionChain'
    )

    await AddressManager.setAddress(
      'OVM_ChainStorageContainer-CTC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'OVM_ChainStorageContainer-CTC-queue',
      queue.address
    )

    await AddressManager.setAddress(
      'OVM_CanonicalTransactionChain',
      OVM_CanonicalTransactionChain.address
    )
  })

  let OVM_L1CrossDomainMessenger: Contract
  beforeEach(async () => {
    const xDomainMessengerImpl = await Factory__OVM_L1CrossDomainMessenger.deploy()
    // We use an upgradable proxy for the XDomainMessenger--deploy & set up the proxy.
    OVM_L1CrossDomainMessenger = await deployProxyXDomainMessenger(
      AddressManager,
      xDomainMessengerImpl
    )
    await OVM_L1CrossDomainMessenger.initialize(AddressManager.address)
  })

  describe('pause', () => {
    describe('when called by the current owner', () => {
      it('should pause the contract', async () => {
        await OVM_L1CrossDomainMessenger.pause()

        expect(await OVM_L1CrossDomainMessenger.paused()).to.be.true
      })
    })

    describe('when called by account other than the owner', () => {
      it('should not pause the contract', async () => {
        await expect(
          OVM_L1CrossDomainMessenger.connect(signer2).pause()
        ).to.be.revertedWith('Ownable: caller is not the owner')
      })
    })
  })

  const getTransactionHash = (
    sender: string,
    target: string,
    gasLimit: number,
    data: string
  ): string => {
    return keccak256(encodeQueueTransaction(sender, target, gasLimit, data))
  }

  const encodeQueueTransaction = (
    sender: string,
    target: string,
    gasLimit: number,
    data: string
  ): string => {
    return ethers.utils.defaultAbiCoder.encode(
      ['address', 'address', 'uint256', 'bytes'],
      [sender, target, gasLimit, data]
    )
  }
  describe('sendMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const gasLimit = 100_000

    it('should be able to send a single message', async () => {
      await expect(
        OVM_L1CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted

      const calldata = encodeXDomainCalldata(
        target,
        await signer.getAddress(),
        message,
        0
      )
      const transactionHash = getTransactionHash(
        OVM_L1CrossDomainMessenger.address,
        Mock__OVM_L2CrossDomainMessenger.address,
        gasLimit,
        calldata
      )

      const queueLength = await OVM_CanonicalTransactionChain.getQueueLength()
      const queueElement = await OVM_CanonicalTransactionChain.getQueueElement(
        queueLength - 1
      )
      expect(queueElement[0]).to.equal(transactionHash)
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

    it('should revert if given the wrong queue index', async () => {
      await OVM_L1CrossDomainMessenger.sendMessage(target, message, 100_001)

      const queueLength = await OVM_CanonicalTransactionChain.getQueueLength()
      await expect(
        OVM_L1CrossDomainMessenger.replayMessage(
          target,
          await signer.getAddress(),
          message,
          queueLength - 1,
          gasLimit
        )
      ).to.be.revertedWith('Provided message has not been enqueued.')
    })

    it('should succeed if the message exists', async () => {
      await OVM_L1CrossDomainMessenger.sendMessage(target, message, gasLimit)
      const queueLength = await OVM_CanonicalTransactionChain.getQueueLength()

      const calldata = encodeXDomainCalldata(
        target,
        await signer.getAddress(),
        message,
        queueLength - 1
      )
      await expect(
        OVM_L1CrossDomainMessenger.replayMessage(
          Mock__OVM_L2CrossDomainMessenger.address,
          await signer.getAddress(),
          calldata,
          queueLength - 1,
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

      calldata = encodeXDomainCalldata(target, sender, message, 0)

      const storageKey = keccak256(
        keccak256(
          calldata + remove0x(Mock__OVM_L2CrossDomainMessenger.address)
        ) + '00'.repeat(32)
      )
      const storageGenerator = await TrieTestGenerator.fromNodes({
        nodes: [
          {
            key: storageKey,
            val: '0x' + '01'.padStart(2, '0'),
          },
        ],
        secure: true,
      })

      const generator = await TrieTestGenerator.fromAccounts({
        accounts: [
          {
            address: predeploys.OVM_L2ToL1MessagePasser,
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
        stateTrieWitness: (
          await generator.makeAccountProofTest(
            predeploys.OVM_L2ToL1MessagePasser
          )
        ).accountTrieWitness,
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

      const proof1 = {
        stateRoot: ethers.constants.HashZero,
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
          proof1
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should revert if provided an invalid state root proof', async () => {
      Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
        false
      )

      const proof1 = {
        stateRoot: ethers.constants.HashZero,
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
          proof1
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

    it('the xDomainMessageSender is reset to the original value', async () => {
      await expect(
        OVM_L1CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')
      await OVM_L1CrossDomainMessenger.relayMessage(
        target,
        sender,
        message,
        0,
        proof
      )
      await expect(
        OVM_L1CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')
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

    it('should revert if paused', async () => {
      await OVM_L1CrossDomainMessenger.pause()

      await expect(
        OVM_L1CrossDomainMessenger.relayMessage(
          target,
          sender,
          message,
          0,
          proof
        )
      ).to.be.revertedWith('Pausable: paused')
    })

    describe('blockMessage and allowMessage', () => {
      it('should revert if called by an account other than the owner', async () => {
        const OVM_L1CrossDomainMessenger2 = OVM_L1CrossDomainMessenger.connect(
          signer2
        )
        await expect(
          OVM_L1CrossDomainMessenger2.blockMessage(keccak256(calldata))
        ).to.be.revertedWith('Ownable: caller is not the owner')

        await expect(
          OVM_L1CrossDomainMessenger2.allowMessage(keccak256(calldata))
        ).to.be.revertedWith('Ownable: caller is not the owner')
      })

      it('should revert if the message is blocked', async () => {
        await OVM_L1CrossDomainMessenger.blockMessage(keccak256(calldata))

        await expect(
          OVM_L1CrossDomainMessenger.relayMessage(
            target,
            sender,
            message,
            0,
            proof
          )
        ).to.be.revertedWith('Provided message has been blocked.')
      })

      it('should succeed if the message is blocked, then unblocked', async () => {
        await OVM_L1CrossDomainMessenger.blockMessage(keccak256(calldata))

        await expect(
          OVM_L1CrossDomainMessenger.relayMessage(
            target,
            sender,
            message,
            0,
            proof
          )
        ).to.be.revertedWith('Provided message has been blocked.')

        await OVM_L1CrossDomainMessenger.allowMessage(keccak256(calldata))

        await expect(
          OVM_L1CrossDomainMessenger.relayMessage(
            target,
            sender,
            message,
            0,
            proof
          )
        ).to.not.be.reverted
      })
    })

    describe('onlyRelayer', () => {
      it('when the OVM_L2MessageRelayer address is set, should revert if called by a different account', async () => {
        // set to a random NON-ZERO address
        await AddressManager.setAddress(
          'OVM_L2MessageRelayer',
          '0x1234123412341234123412341234123412341234'
        )

        await expect(
          OVM_L1CrossDomainMessenger.relayMessage(
            target,
            sender,
            message,
            0,
            proof
          )
        ).to.be.revertedWith(
          'Only OVM_L2MessageRelayer can relay L2-to-L1 messages.'
        )
      })
    })
  })
})
