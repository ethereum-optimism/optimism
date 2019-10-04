import '../setup'

/* Internal Imports */
import {
  Transition,
  generateNTransitions,
  makeRepeatedBytes,
  makePaddedBytes,
  makePaddedUint,
  ZERO_BYTES32,
  ZERO_ADDRESS,
  ZERO_UINT32,
  ZERO_SIGNATURE,
  getSlot,
  getAmount,
  getAddress,
  getSignature,
  getStateRoot,
  UNISWAP_ADDRESS,
  UNISWAP_STORAGE_SLOT,
} from '../helpers'

/* External Imports */
import { ethers } from 'ethers'
import {
  createMockProvider,
  deployContract,
  link,
  getWallets,
} from 'ethereum-waffle'
import {
  keccak256,
  abi,
  hexStrToBuf,
  bufToHexString,
  BigNumber,
  DefaultSignatureProvider,
  SignatureProvider,
  SignatureVerifier,
  DefaultSignatureVerifier,
} from '@pigi/core'
import {
  SwapTransition,
  TransferTransition,
  CreateAndTransferTransition,
  abiEncodeTransition,
  Transfer,
  Swap,
  abiEncodeTransaction,
  Address,
} from '@pigi/wallet'

/* Logging */
import debug from 'debug'
const log = debug('test:info:unipig-transition-evaluator')

/* Contract Imports */
import * as UnipigTransitionEvaluator from '../../build/UnipigTransitionEvaluator.json'

/* Begin tests */
describe.only('UnipigTransitionEvaluator', () => {
  const provider = createMockProvider()
  const [wallet1] = getWallets(provider)
  let unipigEvaluator
  let signatureProvider: SignatureProvider
  let signatureVerifier: SignatureVerifier

  /* Prep a signature provider and verifier for validating sends */
  before(async () => {
    signatureProvider = new DefaultSignatureProvider()
    signatureVerifier = DefaultSignatureVerifier.instance()
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    unipigEvaluator = await deployContract(
      wallet1,
      UnipigTransitionEvaluator,
      [],
      {
        gasLimit: 6700000,
      }
    )
  })
  /*
   * Test Signature Utils
   */
  describe('recoverSigner()', async () => {
    it('should recover the correct signer', async () => {
      const messageToSign: string = '0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
      const signature: string = await signatureProvider.sign(messageToSign)
      const contractRecoveredSigner = await unipigEvaluator.recoverSigner(messageToSign, signature)
      contractRecoveredSigner.should.equal(await signatureProvider.getAddress())
    })
  })
  /*
   * Test inferTransitionType()
   */
  describe('inferTransitionType() ', async () => {
    const txTypes = {
      TRANSFER_NEW_ACCOUNT_TYPE: 0,
      TRANSFER_STORED_ACCOUNT_TYPE: 1,
      SWAP_TYPE: 2,
    }

    it('should infer a transfer new account transaction', async () => {
      // Create a transaction which we will infer the type of
      const createAndTransfer: CreateAndTransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex: 2,
        recipientSlotIndex: 2,
        createdAccountPubkey: getAddress('01'),
        tokenType: 0,
        amount: 1,
        signature: getSignature('01'),
      }
      // Encode!
      const encoded = abiEncodeTransition(createAndTransfer)
      // Attempt to infer the transaction type
      const res = await unipigEvaluator.inferTransitionType(encoded)
      // // Check that it's the correct type
      res.should.equal(txTypes.TRANSFER_NEW_ACCOUNT_TYPE)
    })

    it('should infer a transfer stored account transaction', async () => {
      // Create a transaction which we will infer the type of
      const transfer: TransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex: 2,
        recipientSlotIndex: 2,
        tokenType: 0,
        amount: 1,
        signature: getSignature('01'),
      }
      // Encode!
      const encoded = abiEncodeTransition(transfer)
      // Attempt to infer the transaction type
      const res = await unipigEvaluator.inferTransitionType(encoded)
      // Check that it's the correct type
      res.should.equal(txTypes.TRANSFER_STORED_ACCOUNT_TYPE)
    })

    it('should infer a swap transaction', async () => {
      // Create a transition which we will infer the type of
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex: 2,
        uniswapSlotIndex: 2,
        tokenType: 1,
        inputAmount: 1,
        minOutputAmount: 3,
        timeout: 6,
        signature: getSignature('0'),
      }
      // Encode!
      const encoded = abiEncodeTransition(swap)
      // Attempt to infer the transaction type
      const res = await unipigEvaluator.inferTransitionType(encoded)
      // Check that it's the correct type
      res.should.equal(txTypes.SWAP_TYPE)
    })

    it('should revert if a tx has the wrong number of bytes', async () => {
      try {
        // Attempt to infer a faulty tx type
        const res = await unipigEvaluator.inferTransitionType('0x1234')
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error('Revert expected on invalid tx type length')
    })
  })

  /*
   * Test getTransitionStateRootAndAccessList()
   */
  describe('getTransitionStateRootAndAccessList() ', async () => {
    const stateRoot = getStateRoot('ab')
    const accessList = [1, 2]

    it('should return the expected storage slots for createAndTransferTransition', async () => {
      // Create a transition which we will decode
      const createAndTransfer: CreateAndTransferTransition = {
        stateRoot,
        senderSlotIndex: accessList[0],
        recipientSlotIndex: accessList[1],
        createdAccountPubkey: getAddress('01'),
        tokenType: 0,
        amount: 1,
        signature: getSignature('01'),
      }
      // Encode!
      const encoded = abiEncodeTransition(createAndTransfer)
      // Attempt to decode the transition
      const res = await unipigEvaluator.getTransitionStateRootAndAccessList(
        encoded
      )
      // Check that it returned the expected state root & access list
      res.should.deep.equal([stateRoot, accessList])
    })

    it('should return the expected storage slots for transferTransition', async () => {
      // Create a transition which we will decode
      const transfer: TransferTransition = {
        stateRoot,
        senderSlotIndex: accessList[0],
        recipientSlotIndex: accessList[1],
        tokenType: 0,
        amount: 1,
        signature: getSignature('01'),
      }
      // Encode!
      const encoded = abiEncodeTransition(transfer)
      // Attempt to decode the transition
      const res = await unipigEvaluator.getTransitionStateRootAndAccessList(
        encoded
      )
      // Check that it returned the expected state root & access list
      res.should.deep.equal([stateRoot, accessList])
    })

    it('should return the expected storage slots for swapTransition', async () => {
      // Create a transition which we will decode
      const swap: SwapTransition = {
        stateRoot,
        senderSlotIndex: accessList[0],
        uniswapSlotIndex: accessList[1],
        tokenType: 1,
        inputAmount: 1,
        minOutputAmount: 3,
        timeout: 6,
        signature: getSignature('01'),
      }
      // Encode!
      const encoded = abiEncodeTransition(swap)
      // Attempt to decode the transition
      const res = await unipigEvaluator.getTransitionStateRootAndAccessList(
        encoded
      )
      // Check that it returned the expected state root & access list
      res.should.deep.equal([stateRoot, accessList])
    })

    it('should throw if we put in some random bytes as the transition', async () => {
      const badEncoding = '0xdeadbeefdeadbeefdeadbeef'
      try {
        // Attempt to decode the transition
        const res = await unipigEvaluator.getTransitionStateRootAndAccessList(
          badEncoding
        )
        // It should have failed!
      } catch (err) {
        // Success! It threw!
        return
      }
      throw new Error('Expected bad encoding to fail!')
    })
  })

  /*
   * Test applyTransferTransition()
   */
  describe('applyTransferTransition() ', async () => {
    // TODO complete here next
    it('should return the correct storage slots after a successful send', async () => {
      // Set initialization variables
      const sentAmount = 5
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = await signatureProvider.getAddress() as Address
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      const tokenType = 0
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: recipientAddress,
          balances: initialBalances,
        },
      }

      // Create and sign the transaction
      const transaction: Transfer = {
        sender: senderAddress,
        recipient: recipientAddress,
        tokenType,
        amount: sentAmount
      }
      const signature = await signatureProvider.sign(
        abiEncodeTransaction(transaction)
      )
      // Create the transition  
      const transition: TransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        tokenType,
        amount: sentAmount,
        signature,
      }
      // Attempt to apply the transaction
      const res = await unipigEvaluator.applyTransferTransition(transition, [
        senderStorageSlot,
        recipientStorageSlot,
      ])
      // Check the sender's balance decremented
      res[0].balances.should.deep.equal([995, 1000])
      // Check the recipient's balance incremented
      res[1].balances.should.deep.equal([1005, 1000])
      // Success!
    })

    it('should throw if the sender does not have enough money', async () => {
      // Set initialization variables
      const sentAmount = 1100
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = getAddress('48')
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: recipientAddress,
          balances: initialBalances,
        },
      }

      // Create a transaction
      const transition: TransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        tokenType: 0,
        amount: sentAmount,
        signature: getSignature('9'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applyTransferTransition(transition, [
          senderStorageSlot,
          recipientStorageSlot,
        ])
      } catch (err) {
        // Success!
        return
      }
      throw new Error('Expected to fail due to insufficient balance!')
    })

    // TODO: Enable this test once we add real signature verification to the contract
    it('should throw if the signature for the transfer is invalid', async () => {
      // Set initialization variables
      const sentAmount = 900
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = getAddress('48')
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: recipientAddress,
          balances: initialBalances,
        },
      }

      // Create the transition
      const transition: TransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        tokenType: 0,
        amount: sentAmount,
        signature: getSignature('9'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applyTransferTransition(transition, [
          senderStorageSlot,
          recipientStorageSlot,
        ])
      } catch (err) {
        // Success!
        return
      }
      throw new Error('Expected to fail due to infalid signature!')
    })
  })

  /*
   * Test applyCreateAndTransferTransition()
   */
  describe('applyCreateAndTransferTransition() ', async () => {
    it('should succeed if the recipient storage is empty and it is a successful send', async () => {
      // Set initialization variables
      const sentAmount = 5
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = await signatureProvider.getAddress()
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      const tokenType = 0
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: getAddress('00'),
          balances: [0, 0],
        },
      }
      // Create and sign the transaction
      const transaction: Transfer = {
        sender: senderAddress,
        recipient: recipientAddress,
        tokenType,
        amount: sentAmount
      }
      const signature = await signatureProvider.sign(
        abiEncodeTransaction(transaction)
      )
      // Create a transition
      const transition: CreateAndTransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        createdAccountPubkey: recipientAddress,
        tokenType,
        amount: sentAmount,
        signature,
      }
      // Attempt to apply the transition
      const res = await unipigEvaluator.applyCreateAndTransferTransition(
        transition,
        [senderStorageSlot, recipientStorageSlot]
      )
      // Check the sender's balance decremented
      res[0].balances.should.deep.equal([995, 1000])
      // Check the recipient's balance incremented
      res[1].balances.should.deep.equal([5, 0])
      // Also make sure the new storage slot has the recipient's pubkey
      res[1].pubkey.should.equal(recipientAddress)
      // Success!
    })

    it('should throw if the recipient storage is NOT empty and it is a successful send', async () => {
      // Set initialization variables
      const sentAmount = 5
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = getAddress('48')
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: recipientAddress,
          balances: [10, 10],
        },
      }

      // Create a transition
      const transition: CreateAndTransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        createdAccountPubkey: recipientAddress,
        tokenType: 0,
        amount: sentAmount,
        signature: getSignature('9'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applyCreateAndTransferTransition(
          transition,
          [senderStorageSlot, recipientStorageSlot]
        )
      } catch (err) {
        // Success!
        return
      }
      throw new Error('Expected to fail due to a non-empty storage slot!')
    })
  })

  /*
   * Test applySwapTransition()
   */
  describe('applySwapTransition() ', async () => {
    const senderSlotIndex = 3428942
    const timeout = +new Date()
    const tokenType = 1

    it('should return the correct storage slots after a successful swap', async () => {
      // Set initialization variables
      const senderAddress = await signatureProvider.getAddress()
      const inputAmount = 5
      const minOutputAmount = 4
      const initialBalances = [1000, 1000]
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const uniswapStorageSlot = {
        slotIndex: UNISWAP_STORAGE_SLOT,
        value: {
          pubkey: UNISWAP_ADDRESS,
          balances: initialBalances,
        },
      }
      // Create and sign the transaction
      const transaction: Swap = {
        sender: senderAddress,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
      }
      const signature = await signatureProvider.sign(
        abiEncodeTransaction(transaction)
      )
      // Create a swap transition
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex,
        uniswapSlotIndex: UNISWAP_STORAGE_SLOT,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
        signature: signature,
      }
      // Attempt to apply the transaction
      const res = await unipigEvaluator.applySwapTransition(swap, [
        senderStorageSlot,
        uniswapStorageSlot,
      ])
      // Check to see that the result was successful
      // The sender should have 4 tokens (their min)
      res[0].balances.should.deep.equal([1004, 995])
      // The sender address should be correct
      res[0].pubkey.should.equal(senderAddress)
      // The uniswap should have gained a token from the fee
      res[1].balances.should.deep.equal([996, 1005])
      // Uniswap's address should be correct
      res[1].pubkey.should.equal(UNISWAP_ADDRESS)
      // Success!
    })

    it('should throw if the min output amount is too high', async () => {
      // Set initialization variables
      const senderAddress = await signatureProvider.getAddress()
      const inputAmount = 5
      const minOutputAmount = 5
      // We're setting the output amount to 5 -- this won't be what is returned because we have a fee!
      // 1000 + (5 + FEE) = 1000 - OUTPUT          -- note here OUTPUT will have to be less than 5 if FEE > 0.
      const initialBalances = [1000, 1000]
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const uniswapStorageSlot = {
        slotIndex: UNISWAP_STORAGE_SLOT,
        value: {
          pubkey: UNISWAP_ADDRESS,
          balances: initialBalances,
        },
      }
      // Create a swap transition
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex,
        uniswapSlotIndex: UNISWAP_STORAGE_SLOT,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
        signature: getSignature('aa'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applySwapTransition(swap, [
          senderStorageSlot,
          uniswapStorageSlot,
        ])
        // We shouldn't get here!
      } catch (err) {
        // Success!
        return
      }
      throw new Error(
        'Expected minOutputAmount to be too high & for this to throw!'
      )
    })

    it('should throw if the sender cannot aford the swap', async () => {
      // Set initialization variables
      const senderAddress = await signatureProvider.getAddress()
      const inputAmount = 5000
      const minOutputAmount = 4
      const initialBalances = [1000, 1000]
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const uniswapStorageSlot = {
        slotIndex: UNISWAP_STORAGE_SLOT,
        value: {
          pubkey: UNISWAP_ADDRESS,
          balances: initialBalances,
        },
      }
      // Create a swap transition
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex,
        uniswapSlotIndex: UNISWAP_STORAGE_SLOT,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
        signature: getSignature('aa'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applySwapTransition(swap, [
          senderStorageSlot,
          uniswapStorageSlot,
        ])
        // We shouldn't get here!
      } catch (err) {
        // Success!
        return
      }
      throw new Error(
        'Expected sender not being able to afford the tx to cause this to throw!'
      )
    })

    it('should throw if the 2nd storage slot doesnt match the Uniswap address', async () => {
      // These are NOT the uniswap storage slot & address
      const senderAddress = await signatureProvider.getAddress()
      const NOT_UNISWAP_STORAGE_SLOT = 10
      const NOT_UNISWAP_ADDRESS = getAddress('99')
      // Set initialization variables
      const inputAmount = 5000
      const minOutputAmount = 4
      const initialBalances = [1000, 1000]
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const uniswapStorageSlot = {
        slotIndex: NOT_UNISWAP_STORAGE_SLOT,
        value: {
          pubkey: NOT_UNISWAP_ADDRESS,
          balances: initialBalances,
        },
      }
      // Create a swap transition
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex,
        uniswapSlotIndex: NOT_UNISWAP_STORAGE_SLOT,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
        signature: getSignature('aa'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applySwapTransition(swap, [
          senderStorageSlot,
          uniswapStorageSlot,
        ])
        // We shouldn't get here!
      } catch (err) {
        // Success!
        return
      }
      throw new Error(
        'Swapping with a user other than Uniswap should cause the tx to throw!'
      )
    })

    it('should throw if the signature for the swap does not match the sender', async () => {
      // Set initialization variables
      const senderAddress = await signatureProvider.getAddress()
      const inputAmount = 5
      const minOutputAmount = 4
      const initialBalances = [1000, 1000]
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const uniswapStorageSlot = {
        slotIndex: UNISWAP_STORAGE_SLOT,
        value: {
          pubkey: UNISWAP_ADDRESS,
          balances: initialBalances,
        },
      }
      // Create a swap transition
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex,
        uniswapSlotIndex: UNISWAP_STORAGE_SLOT,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
        signature: getSignature('aa'),
      }
      try {
        // Attempt to apply the transaction
        const res = await unipigEvaluator.applySwapTransition(swap, [
          senderStorageSlot,
          uniswapStorageSlot,
        ])
      } catch (err) {
        // Success!
        return
      }
      throw new Error('Expected to fail due to infalid signature!')
    })
  })

  /*
   * Test evaluateTransition()
   */
  describe('evaluateTransition() ', async () => {
    it('should evaluate a transfer transition without failing', async () => {
      // Set initialization variables
      const sentAmount = 5
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = await signatureProvider.getAddress()
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      const tokenType = 0
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: recipientAddress,
          balances: initialBalances,
        },
      }
      // Create and sign the transaction
      const transaction: Transfer = {
        sender: senderAddress,
        recipient: recipientAddress,
        tokenType,
        amount: sentAmount
      }
      const signature = await signatureProvider.sign(
        abiEncodeTransaction(transaction)
      )

      // Create a transaction which we will infer the type of
      const transfer: TransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        tokenType,
        amount: sentAmount,
        signature,
      }
      // Encode!
      const encoded = abiEncodeTransition(transfer)
      // Attempt to apply the transaction
      const res = await unipigEvaluator.evaluateTransition(encoded, [
        senderStorageSlot,
        recipientStorageSlot,
      ])
      log('Transfer Transition successfully returned:', res)
      // Success!
    })

    it('should evaluate a CreateAndTransferTransition without throwing', async () => {
      // Set initialization variables
      const sentAmount = 5
      const initialBalances = [1000, 1000]
      const senderSlotIndex = 50
      const senderAddress = await signatureProvider.getAddress()
      const recipientAddress = getAddress('38')
      const recipientSlotIndex = 100
      const tokenType = 0
      // Create the storage slots
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const recipientStorageSlot = {
        slotIndex: recipientSlotIndex,
        value: {
          pubkey: getAddress('00'),
          balances: [0, 0],
        },
      }
      // Create and sign the transaction
      const transaction: Transfer = {
        sender: senderAddress,
        recipient: recipientAddress,
        tokenType,
        amount: sentAmount
      }
      const signature = await signatureProvider.sign(
        abiEncodeTransaction(transaction)
      )
      // Create a transition
      const createAndTransfer: CreateAndTransferTransition = {
        stateRoot: getStateRoot('ab'),
        senderSlotIndex,
        recipientSlotIndex,
        createdAccountPubkey: recipientAddress,
        tokenType,
        amount: sentAmount,
        signature,
      }
      // Encode!
      const encoded = abiEncodeTransition(createAndTransfer)
      // Attempt to apply the transaction
      const res = await unipigEvaluator.evaluateTransition(encoded, [
        senderStorageSlot,
        recipientStorageSlot,
      ])
      log('Create and Transfer Transition successfully returned:', res)
      // Success!
    })

    it('should evaluate a swap transition without failing', async () => {
      // Set initialization variables
      const senderAddress = await signatureProvider.getAddress()
      const senderSlotIndex = 3428942
      const timeout = +new Date()
      const tokenType = 1
      const inputAmount = 5
      const minOutputAmount = 4
      const initialBalances = [1000, 1000]
      const senderStorageSlot = {
        slotIndex: senderSlotIndex,
        value: {
          pubkey: senderAddress,
          balances: initialBalances,
        },
      }
      const uniswapStorageSlot = {
        slotIndex: UNISWAP_STORAGE_SLOT,
        value: {
          pubkey: UNISWAP_ADDRESS,
          balances: initialBalances,
        },
      }
      // Create and sign the transaction
      const transaction: Swap = {
        sender: senderAddress,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
      }
      const signature = await signatureProvider.sign(
        abiEncodeTransaction(transaction)
      )
      // Create a swap transition
      const swap: SwapTransition = {
        stateRoot: getStateRoot('cd'),
        senderSlotIndex,
        uniswapSlotIndex: UNISWAP_STORAGE_SLOT,
        tokenType,
        inputAmount,
        minOutputAmount,
        timeout,
        signature,
      }
      const encoded = abiEncodeTransition(swap)
      // Attempt to apply the transaction
      const res = await unipigEvaluator.evaluateTransition(encoded, [
        senderStorageSlot,
        uniswapStorageSlot,
      ])
      log('Swap Transition successfully returned:', res)
      // Success!
    })
  })
})
