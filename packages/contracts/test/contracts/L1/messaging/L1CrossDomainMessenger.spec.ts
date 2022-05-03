import { ethers } from 'hardhat'
import { Contract, BigNumber } from 'ethers'
import { smock, FakeContract, MockContract } from '@defi-wonderland/smock'
import { toHexString, applyL1ToL2Alias } from '@eth-optimism/core-utils'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import {
  NON_NULL_BYTES32,
  NON_ZERO_ADDRESS,
  DUMMY_BATCH_HEADERS,
  DUMMY_BATCH_PROOFS,
  L2_GAS_DISCOUNT_DIVISOR,
  ENQUEUE_GAS_COST,
  TrieTestGenerator,
  getNextBlockNumber,
  encodeXDomainCalldata,
  getEthTime,
  setEthTime,
  deploy,
} from '../../../helpers'
import { predeploys } from '../../../../src'

const MAX_GAS_LIMIT = 8_000_000

describe('L1CrossDomainMessenger', () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let Fake__TargetContract: FakeContract
  let Fake__L2CrossDomainMessenger: FakeContract
  let Fake__StateCommitmentChain: FakeContract
  before(async () => {
    Fake__TargetContract = await smock.fake<Contract>('TestERC20')
    Fake__L2CrossDomainMessenger = await smock.fake<Contract>(
      'L2CrossDomainMessenger',
      {
        address: predeploys.L2CrossDomainMessenger,
      }
    )
    Fake__StateCommitmentChain = await smock.fake<Contract>(
      'StateCommitmentChain'
    )
  })

  let AddressManager: Contract
  let CanonicalTransactionChain: Contract
  before(async () => {
    AddressManager = await deploy('Lib_AddressManager')

    await AddressManager.setAddress(
      'L2CrossDomainMessenger',
      Fake__L2CrossDomainMessenger.address
    )

    await AddressManager.setAddress(
      'StateCommitmentChain',
      Fake__StateCommitmentChain.address
    )

    CanonicalTransactionChain = await deploy('CanonicalTransactionChain', {
      args: [
        AddressManager.address,
        MAX_GAS_LIMIT,
        L2_GAS_DISCOUNT_DIVISOR,
        ENQUEUE_GAS_COST,
      ],
    })

    const batches = await deploy('ChainStorageContainer', {
      args: [AddressManager.address, 'CanonicalTransactionChain'],
    })

    await AddressManager.setAddress(
      'ChainStorageContainer-CTC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'CanonicalTransactionChain',
      CanonicalTransactionChain.address
    )
  })

  let L1CrossDomainMessenger: Contract
  beforeEach(async () => {
    const xDomainMessengerImpl = await deploy('L1CrossDomainMessenger')

    await AddressManager.setAddress(
      'L1CrossDomainMessenger',
      xDomainMessengerImpl.address
    )

    const proxy = await deploy('Lib_ResolvedDelegateProxy', {
      args: [AddressManager.address, 'L1CrossDomainMessenger'],
    })

    L1CrossDomainMessenger = xDomainMessengerImpl.attach(proxy.address)

    await L1CrossDomainMessenger.initialize(AddressManager.address)
  })

  describe('pause', () => {
    describe('when called by the current owner', () => {
      it('should pause the contract', async () => {
        await L1CrossDomainMessenger.pause()

        expect(await L1CrossDomainMessenger.paused()).to.be.true
      })
    })

    describe('when called by account other than the owner', () => {
      it('should not pause the contract', async () => {
        await expect(
          L1CrossDomainMessenger.connect(signer2).pause()
        ).to.be.revertedWith('Ownable: caller is not the owner')
      })
    })
  })

  describe('sendMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const gasLimit = 100_000

    it('should be able to send a single message', async () => {
      await expect(
        L1CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted

      const calldata = encodeXDomainCalldata(
        target,
        signer1.address,
        message,
        0
      )
      const transactionHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ['address', 'address', 'uint256', 'bytes'],
          [
            applyL1ToL2Alias(L1CrossDomainMessenger.address),
            Fake__L2CrossDomainMessenger.address,
            gasLimit,
            calldata,
          ]
        )
      )
      const queueLength = await CanonicalTransactionChain.getQueueLength()
      const queueElement = await CanonicalTransactionChain.getQueueElement(
        queueLength - 1
      )
      expect(queueElement[0]).to.equal(transactionHash)
    })

    it('should be able to send the same message twice', async () => {
      await L1CrossDomainMessenger.sendMessage(target, message, gasLimit)

      await expect(
        L1CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted
    })
  })

  describe('replayMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const oldGasLimit = 100_000
    const newGasLimit = 200_000

    let queueIndex: number
    beforeEach(async () => {
      await L1CrossDomainMessenger.connect(signer1).sendMessage(
        target,
        message,
        oldGasLimit
      )

      const queueLength = await CanonicalTransactionChain.getQueueLength()
      queueIndex = queueLength - 1
    })

    describe('when giving some incorrect input value', async () => {
      it('should revert if given the wrong target', async () => {
        await expect(
          L1CrossDomainMessenger.replayMessage(
            ethers.constants.AddressZero, // Wrong target
            signer1.address,
            message,
            queueIndex,
            oldGasLimit,
            newGasLimit
          )
        ).to.be.revertedWith('Provided message has not been enqueued.')
      })

      it('should revert if given the wrong sender', async () => {
        await expect(
          L1CrossDomainMessenger.replayMessage(
            target,
            ethers.constants.AddressZero, // Wrong sender
            message,
            queueIndex,
            oldGasLimit,
            newGasLimit
          )
        ).to.be.revertedWith('Provided message has not been enqueued.')
      })

      it('should revert if given the wrong message', async () => {
        await expect(
          L1CrossDomainMessenger.replayMessage(
            target,
            signer1.address,
            '0x', // Wrong message
            queueIndex,
            oldGasLimit,
            newGasLimit
          )
        ).to.be.revertedWith('Provided message has not been enqueued.')
      })

      it('should revert if given the wrong queue index', async () => {
        await expect(
          L1CrossDomainMessenger.replayMessage(
            target,
            signer1.address,
            message,
            queueIndex - 1, // Wrong queue index
            oldGasLimit,
            newGasLimit
          )
        ).to.be.revertedWith('Provided message has not been enqueued.')
      })

      it('should revert if given the wrong old gas limit', async () => {
        await expect(
          L1CrossDomainMessenger.replayMessage(
            target,
            signer1.address,
            message,
            queueIndex,
            oldGasLimit + 1, // Wrong gas limit
            newGasLimit
          )
        ).to.be.revertedWith('Provided message has not been enqueued.')
      })
    })

    describe('when all input values are the same as the existing message', () => {
      it('should succeed', async () => {
        await expect(
          L1CrossDomainMessenger.replayMessage(
            target,
            signer1.address,
            message,
            queueIndex,
            oldGasLimit,
            newGasLimit
          )
        ).to.not.be.reverted
      })

      it('should emit the TransactionEnqueued event', async () => {
        const newQueueIndex = await CanonicalTransactionChain.getQueueLength()
        const newTimestamp = (await getEthTime(ethers.provider)) + 100
        await setEthTime(ethers.provider, newTimestamp)

        await expect(
          L1CrossDomainMessenger.replayMessage(
            target,
            signer1.address,
            message,
            queueIndex,
            oldGasLimit,
            newGasLimit
          )
        )
          .to.emit(CanonicalTransactionChain, 'TransactionEnqueued')
          .withArgs(
            applyL1ToL2Alias(L1CrossDomainMessenger.address),
            Fake__L2CrossDomainMessenger.address,
            newGasLimit,
            encodeXDomainCalldata(target, signer1.address, message, queueIndex),
            newQueueIndex,
            newTimestamp
          )
      })
    })

    it('should succeed if all inputs are the same as the existing message', async () => {
      await L1CrossDomainMessenger.sendMessage(target, message, oldGasLimit)
      const queueLength = await CanonicalTransactionChain.getQueueLength()

      await expect(
        L1CrossDomainMessenger.replayMessage(
          target,
          signer1.address,
          message,
          queueLength - 1,
          oldGasLimit,
          newGasLimit
        )
      ).to.not.be.reverted
    })
  })

  describe('xDomainMessageSender', () => {
    let Mock__L1CrossDomainMessenger: MockContract<Contract>
    before(async () => {
      Mock__L1CrossDomainMessenger = await (
        await smock.mock('L1CrossDomainMessenger')
      ).deploy()
    })

    it('should return the xDomainMsgSender address', async () => {
      await Mock__L1CrossDomainMessenger.setVariable(
        'xDomainMsgSender',
        NON_ZERO_ADDRESS
      )

      expect(
        await Mock__L1CrossDomainMessenger.xDomainMessageSender()
      ).to.equal(NON_ZERO_ADDRESS)
    })
  })

  const generateMockRelayMessageProof = async (
    target: string,
    sender: string,
    message: string,
    messageNonce: number = 0
  ): Promise<{
    calldata: string
    proof: any
  }> => {
    const calldata = encodeXDomainCalldata(
      target,
      sender,
      message,
      messageNonce
    )

    const storageKey = ethers.utils.keccak256(
      ethers.utils.hexConcat([
        ethers.utils.keccak256(
          ethers.utils.hexConcat([
            calldata,
            Fake__L2CrossDomainMessenger.address,
          ])
        ),
        ethers.constants.HashZero,
      ])
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
          codeHash: ethers.utils.keccak256('0x1234'),
          storageRoot: toHexString(storageGenerator._trie.root),
        },
      ],
      secure: true,
    })

    const proof = {
      stateRoot: toHexString(generator._trie.root),
      stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
      stateRootProof: DUMMY_BATCH_PROOFS[0],
      stateTrieWitness: (
        await generator.makeAccountProofTest(predeploys.OVM_L2ToL1MessagePasser)
      ).accountTrieWitness,
      storageTrieWitness: (
        await storageGenerator.makeInclusionProofTest(storageKey)
      ).proof,
    }

    return {
      calldata,
      proof,
    }
  }

  describe('relayMessage', () => {
    let target: string
    let message: string
    let proof: any
    let calldata: string
    before(async () => {
      target = Fake__TargetContract.address
      message = Fake__TargetContract.interface.encodeFunctionData('mint', [
        NON_ZERO_ADDRESS,
        ethers.utils.parseEther('1'),
      ])
      ;({ proof, calldata } = await generateMockRelayMessageProof(
        target,
        signer1.address,
        message
      ))
    })

    beforeEach(() => {
      Fake__StateCommitmentChain.verifyStateCommitment.returns(true)
      Fake__StateCommitmentChain.insideFraudProofWindow.returns(false)
    })

    it('should revert if still inside the fraud proof window', async () => {
      Fake__StateCommitmentChain.insideFraudProofWindow.returns(true)

      const proof1 = {
        stateRoot: ethers.constants.HashZero,
        stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
        stateRootProof: DUMMY_BATCH_PROOFS[0],
        stateTrieWitness: '0x',
        storageTrieWitness: '0x',
      }

      await expect(
        L1CrossDomainMessenger.relayMessage(
          target,
          signer1.address,
          message,
          0,
          proof1
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should revert if attempting to relay a message sent to an L1 system contract', async () => {
      const maliciousProof = await generateMockRelayMessageProof(
        CanonicalTransactionChain.address,
        signer1.address,
        message
      )

      await expect(
        L1CrossDomainMessenger.relayMessage(
          CanonicalTransactionChain.address,
          signer1.address,
          message,
          0,
          maliciousProof.proof
        )
      ).to.be.revertedWith(
        'Cannot send L2->L1 messages to L1 system contracts.'
      )
    })

    it('should revert if provided an invalid state root proof', async () => {
      Fake__StateCommitmentChain.verifyStateCommitment.returns(false)

      const proof1 = {
        stateRoot: ethers.constants.HashZero,
        stateRootBatchHeader: DUMMY_BATCH_HEADERS[0],
        stateRootProof: DUMMY_BATCH_PROOFS[0],
        stateTrieWitness: '0x',
        storageTrieWitness: '0x',
      }

      await expect(
        L1CrossDomainMessenger.relayMessage(
          target,
          signer1.address,
          message,
          0,
          proof1
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should revert if provided an invalid storage trie witness', async () => {
      await expect(
        L1CrossDomainMessenger.relayMessage(
          target,
          signer1.address,
          message,
          0,
          {
            ...proof,
            storageTrieWitness: '0x',
          }
        )
      ).to.be.reverted
    })

    it('should revert if provided an invalid state trie witness', async () => {
      await expect(
        L1CrossDomainMessenger.relayMessage(
          target,
          signer1.address,
          message,
          0,
          {
            ...proof,
            stateTrieWitness: '0x',
          }
        )
      ).to.be.reverted
    })

    it('should send a successful call to the target contract', async () => {
      const blockNumber = await getNextBlockNumber(ethers.provider)

      await L1CrossDomainMessenger.relayMessage(
        target,
        signer1.address,
        message,
        0,
        proof
      )

      expect(
        await L1CrossDomainMessenger.successfulMessages(
          ethers.utils.keccak256(calldata)
        )
      ).to.equal(true)

      expect(
        await L1CrossDomainMessenger.relayedMessages(
          ethers.utils.keccak256(
            ethers.utils.hexConcat([
              calldata,
              signer1.address,
              ethers.utils.hexZeroPad(
                BigNumber.from(blockNumber).toHexString(),
                32
              ),
            ])
          )
        )
      ).to.equal(true)
    })

    it('the xDomainMessageSender is reset to the original value', async () => {
      await expect(
        L1CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')

      await L1CrossDomainMessenger.relayMessage(
        target,
        signer1.address,
        message,
        0,
        proof
      )

      await expect(
        L1CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')
    })

    it('should revert if trying to send the same message twice', async () => {
      await L1CrossDomainMessenger.relayMessage(
        target,
        signer1.address,
        message,
        0,
        proof
      )

      await expect(
        L1CrossDomainMessenger.relayMessage(
          target,
          signer1.address,
          message,
          0,
          proof
        )
      ).to.be.revertedWith('Provided message has already been received.')
    })

    it('should revert if paused', async () => {
      await L1CrossDomainMessenger.pause()

      await expect(
        L1CrossDomainMessenger.relayMessage(
          target,
          signer1.address,
          message,
          0,
          proof
        )
      ).to.be.revertedWith('Pausable: paused')
    })

    describe('blockMessage and allowMessage', () => {
      it('should revert if called by an account other than the owner', async () => {
        const L1CrossDomainMessenger2 = L1CrossDomainMessenger.connect(signer2)

        await expect(
          L1CrossDomainMessenger2.blockMessage(ethers.utils.keccak256(calldata))
        ).to.be.revertedWith('Ownable: caller is not the owner')

        await expect(
          L1CrossDomainMessenger2.allowMessage(ethers.utils.keccak256(calldata))
        ).to.be.revertedWith('Ownable: caller is not the owner')
      })

      it('should revert if the message is blocked', async () => {
        await L1CrossDomainMessenger.blockMessage(
          ethers.utils.keccak256(calldata)
        )

        await expect(
          L1CrossDomainMessenger.relayMessage(
            target,
            signer1.address,
            message,
            0,
            proof
          )
        ).to.be.revertedWith('Provided message has been blocked.')
      })

      it('should succeed if the message is blocked, then unblocked', async () => {
        await L1CrossDomainMessenger.blockMessage(
          ethers.utils.keccak256(calldata)
        )

        await expect(
          L1CrossDomainMessenger.relayMessage(
            target,
            signer1.address,
            message,
            0,
            proof
          )
        ).to.be.revertedWith('Provided message has been blocked.')

        await L1CrossDomainMessenger.allowMessage(
          ethers.utils.keccak256(calldata)
        )

        await expect(
          L1CrossDomainMessenger.relayMessage(
            target,
            signer1.address,
            message,
            0,
            proof
          )
        ).to.not.be.reverted
      })
    })
  })
})
