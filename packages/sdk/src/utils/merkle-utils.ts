/* Imports: External */
import { ethers, BigNumber } from 'ethers'
import {
  fromHexString,
  toHexString,
  toRpcHexString,
} from '@eth-optimism/core-utils'
import { MerkleTree } from 'merkletreejs'

/**
 * Generates a Merkle proof (using the particular scheme we use within Lib_MerkleTree).
 *
 * @param leaves Leaves of the merkle tree.
 * @param index Index to generate a proof for.
 * @returns Merkle proof sibling leaves, as hex strings.
 */
export const makeMerkleTreeProof = (
  leaves: string[],
  index: number
): string[] => {
  // Our specific Merkle tree implementation requires that the number of leaves is a power of 2.
  // If the number of given leaves is less than a power of 2, we need to round up to the next
  // available power of 2. We fill the remaining space with the hash of bytes32(0).
  const correctedTreeSize = Math.pow(2, Math.ceil(Math.log2(leaves.length)))
  const parsedLeaves = []
  for (let i = 0; i < correctedTreeSize; i++) {
    if (i < leaves.length) {
      parsedLeaves.push(leaves[i])
    } else {
      parsedLeaves.push(ethers.utils.keccak256('0x' + '00'.repeat(32)))
    }
  }

  // merkletreejs prefers things to be Buffers.
  const bufLeaves = parsedLeaves.map(fromHexString)
  const tree = new MerkleTree(bufLeaves, (el: Buffer | string): Buffer => {
    return fromHexString(ethers.utils.keccak256(el))
  })

  const proof = tree.getProof(bufLeaves[index], index).map((element: any) => {
    return toHexString(element.data)
  })

  return proof
}

/**
 * Generates a Merkle-Patricia trie proof for a given account and storage slot.
 *
 * @param provider RPC provider attached to an EVM-compatible chain.
 * @param blockNumber Block number to generate the proof at.
 * @param address Address to generate the proof for.
 * @param slot Storage slot to generate the proof for.
 * @returns Account proof and storage proof.
 */
export const makeStateTrieProof = async (
  provider: ethers.providers.JsonRpcProvider,
  blockNumber: number,
  address: string,
  slot: string
): Promise<{
  accountProof: string[]
  storageProof: string[]
  storageValue: BigNumber
  storageRoot: string
}> => {
  const proof = await provider.send('eth_getProof', [
    address,
    [slot],
    toRpcHexString(blockNumber),
  ])

  return {
    accountProof: proof.accountProof,
    storageProof: proof.storageProof[0].proof,
    storageValue: BigNumber.from(proof.storageProof[0].value),
    storageRoot: proof.storageHash,
  }
}
