import { ethers } from 'ethers'
import { toHexString } from '@eth-optimism/core-utils'

import { TrieTestGenerator } from './trie-test-generator'
import { predeploys } from './constants'

interface WithdrawalArgs {
  nonce: number
  sender: string
  target: string
  value: number
  gasLimit: number
  data: string
}

interface OutputRootProof {
  version: string
  stateRoot: string
  withdrawerStorageRoot: string
  latestBlockhash: string
}

export const deriveWithdrawalHash = (wd: WithdrawalArgs): string => {
  return ethers.utils.keccak256(
    ethers.utils.defaultAbiCoder.encode(
      ['uint256', 'address', 'address', 'uint256', 'uint256', 'bytes'],
      [wd.nonce, wd.sender, wd.target, wd.value, wd.gasLimit, wd.data]
    )
  )
}

export const generateMockWithdrawalProof = async (
  wd: WithdrawalArgs | string
): Promise<{
  outputRootProof: OutputRootProof
  storageTrieWitness: string
}> => {
  let withdrawalHash
  if (typeof wd == 'string') {
    // wd should be an abi encoded string
    withdrawalHash = ethers.utils.keccak256(wd)
  } else {
    withdrawalHash = deriveWithdrawalHash(wd as WithdrawalArgs)
  }

  const storageKey = ethers.utils.keccak256(
    ethers.utils.hexConcat([
      withdrawalHash,
      ethers.utils.hexZeroPad('0x01', 32),
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
        address: predeploys.L2ToL1MessagePasser,
        nonce: 0,
        balance: 0,
        codeHash: ethers.utils.keccak256('0x1234'),
        storageRoot: toHexString(storageGenerator._trie.root),
      },
    ],
    secure: true,
  })

  return {
    outputRootProof: {
      version: ethers.constants.HashZero,
      stateRoot: toHexString(generator._trie.root),
      withdrawerStorageRoot: toHexString(storageGenerator._trie.root),
      latestBlockhash: ethers.constants.HashZero,
    },
    storageTrieWitness: (
      await storageGenerator.makeInclusionProofTest(storageKey)
    ).proof,
  }
}

export const generateOutputRoot = (outputElements: {
  version: string
  stateRoot: string
  withdrawerStorageRoot: string
  latestBlockhash: string
}) => {
  const { version, stateRoot, withdrawerStorageRoot, latestBlockhash } =
    outputElements
  return ethers.utils.solidityKeccak256(
    ['bytes32', 'bytes32', 'bytes32', 'bytes32'],
    [version, stateRoot, withdrawerStorageRoot, latestBlockhash]
  )
}
