import { BigNumber, utils, constants } from 'ethers'
import {
  decodeVersionedNonce,
  hashCrossDomainMessage,
  DepositTx,
  SourceHashDomain,
  encodeCrossDomainMessage,
  hashWithdrawal,
  hashOutputRootProof,
} from '@eth-optimism/core-utils'
import { SecureTrie } from '@ethereumjs/trie'
import { Account, Address, toBuffer, bufferToHex } from '@ethereumjs/util'

import { predeploys } from '../src'

const { hexZeroPad, keccak256 } = utils

const args = process.argv.slice(2)
const command = args[0]

;(async () => {
  switch (command) {
    case 'decodeVersionedNonce': {
      const input = BigNumber.from(args[1])
      const { nonce, version } = decodeVersionedNonce(input)

      const output = utils.defaultAbiCoder.encode(
        ['uint256', 'uint256'],
        [nonce.toHexString(), version.toHexString()]
      )
      process.stdout.write(output)
      break
    }
    case 'encodeCrossDomainMessage': {
      const nonce = BigNumber.from(args[1])
      const sender = args[2]
      const target = args[3]
      const value = BigNumber.from(args[4])
      const gasLimit = BigNumber.from(args[5])
      const data = args[6]

      const encoding = encodeCrossDomainMessage(
        nonce,
        sender,
        target,
        value,
        gasLimit,
        data
      )

      const output = utils.defaultAbiCoder.encode(['bytes'], [encoding])
      process.stdout.write(output)
      break
    }
    case 'hashCrossDomainMessage': {
      const nonce = BigNumber.from(args[1])
      const sender = args[2]
      const target = args[3]
      const value = BigNumber.from(args[4])
      const gasLimit = BigNumber.from(args[5])
      const data = args[6]

      const hash = hashCrossDomainMessage(
        nonce,
        sender,
        target,
        value,
        gasLimit,
        data
      )
      const output = utils.defaultAbiCoder.encode(['bytes32'], [hash])
      process.stdout.write(output)
      break
    }
    case 'hashDepositTransaction': {
      // The solidity transaction hash computation currently only works with
      // user deposits. System deposit transaction hashing is not supported.
      const l1BlockHash = args[1]
      const logIndex = BigNumber.from(args[2])
      const from = args[3]
      const to = args[4]
      const mint = BigNumber.from(args[5])
      const value = BigNumber.from(args[6])
      const gas = BigNumber.from(args[7])
      const data = args[8]

      const tx = new DepositTx({
        l1BlockHash,
        logIndex,
        from,
        to,
        mint,
        value,
        gas,
        data,
        isSystemTransaction: false,
        domain: SourceHashDomain.UserDeposit,
      })

      const digest = tx.hash()
      const output = utils.defaultAbiCoder.encode(['bytes32'], [digest])
      process.stdout.write(output)
      break
    }
    case 'encodeDepositTransaction': {
      const from = args[1]
      const to = args[2]
      const value = BigNumber.from(args[3])
      const mint = BigNumber.from(args[4])
      const gasLimit = BigNumber.from(args[5])
      const isCreate = args[6] === 'true' ? true : false
      const data = args[7]
      const l1BlockHash = args[8]
      const logIndex = BigNumber.from(args[9])

      const tx = new DepositTx({
        from,
        to: isCreate ? null : to,
        value,
        mint,
        gas: gasLimit,
        data,
        l1BlockHash,
        logIndex,
        domain: SourceHashDomain.UserDeposit,
      })

      const raw = tx.encode()
      const output = utils.defaultAbiCoder.encode(['bytes'], [raw])
      process.stdout.write(output)
      break
    }
    case 'hashWithdrawal': {
      const nonce = BigNumber.from(args[1])
      const sender = args[2]
      const target = args[3]
      const value = BigNumber.from(args[4])
      const gas = BigNumber.from(args[5])
      const data = args[6]

      const hash = hashWithdrawal(nonce, sender, target, value, gas, data)
      const output = utils.defaultAbiCoder.encode(['bytes32'], [hash])
      process.stdout.write(output)
      break
    }
    case 'hashOutputRootProof': {
      const version = hexZeroPad(BigNumber.from(args[1]).toHexString(), 32)
      const stateRoot = hexZeroPad(BigNumber.from(args[2]).toHexString(), 32)
      const messagePasserStorageRoot = hexZeroPad(
        BigNumber.from(args[3]).toHexString(),
        32
      )
      const latestBlockhash = hexZeroPad(
        BigNumber.from(args[4]).toHexString(),
        32
      )

      const hash = hashOutputRootProof({
        version,
        stateRoot,
        messagePasserStorageRoot,
        latestBlockhash,
      })
      const output = utils.defaultAbiCoder.encode(['bytes32'], [hash])
      process.stdout.write(output)
      break
    }
    case 'getProveWithdrawalTransactionInputs': {
      const nonce = BigNumber.from(args[1])
      const sender = args[2]
      const target = args[3]
      const value = BigNumber.from(args[4])
      const gas = BigNumber.from(args[5])
      const data = args[6]

      // Compute the withdrawalHash
      const withdrawalHash = hashWithdrawal(
        nonce,
        sender,
        target,
        value,
        gas,
        data
      )

      // Compute the storage slot the withdrawalHash will be stored in
      const slot = utils.defaultAbiCoder.encode(
        ['bytes32', 'bytes32'],
        [withdrawalHash, utils.hexZeroPad('0x', 32)]
      )
      const key = keccak256(slot)

      // Create the account storage trie
      const storage = new SecureTrie()
      // Put a bool "true" into storage
      await storage.put(toBuffer(key), toBuffer('0x01'))

      // Put the storage root into the L2ToL1MessagePasser storage
      const address = Address.fromString(predeploys.L2ToL1MessagePasser)
      const account = Account.fromAccountData({
        nonce: 0,
        balance: 0,
        stateRoot: storage.root,
      })

      const world = new SecureTrie()
      await world.put(address.toBuffer(), account.serialize())

      const proof = await SecureTrie.createProof(storage, toBuffer(key))

      const outputRoot = hashOutputRootProof({
        version: constants.HashZero,
        stateRoot: bufferToHex(world.root),
        messagePasserStorageRoot: bufferToHex(storage.root),
        latestBlockhash: constants.HashZero,
      })

      const output = utils.defaultAbiCoder.encode(
        ['bytes32', 'bytes32', 'bytes32', 'bytes32', 'bytes[]'],
        [world.root, storage.root, outputRoot, withdrawalHash, proof]
      )
      process.stdout.write(output)
      break
    }
  }
})().catch((err: Error) => {
  console.error(err)
  process.stdout.write('')
})
