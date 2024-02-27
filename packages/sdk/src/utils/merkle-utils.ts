/* Imports: External */
import { ethers, BigNumber } from 'ethers'
import {
  fromHexString,
  toHexString,
  toRpcHexString,
} from '@eth-optimism/core-utils'
import { MerkleTree } from 'merkletreejs'
import * as rlp from 'rlp'

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
 * Fix for the case where the final proof element is less than 32 bytes and the element exists
 * inside of a branch node. Current implementation of the onchain MPT contract can't handle this
 * natively so we instead append an extra proof element to handle it instead.
 *
 * @param key Key that the proof is for.
 * @param proof Proof to potentially modify.
 * @returns Modified proof.
 */
export const maybeAddProofNode = (key: string, proof: string[]) => {
  const modifiedProof = [...proof]
  const finalProofEl = modifiedProof[modifiedProof.length - 1]
  const finalProofElDecoded = rlp.decode(finalProofEl) as any
  if (finalProofElDecoded.length === 17) {
    for (const item of finalProofElDecoded) {
      // Find any nodes located inside of the branch node.
      if (Array.isArray(item)) {
        // Check if the key inside the node matches the key we're looking for. We remove the first
        // two characters (0x) and then we remove one more character (the first nibble) since this
        // is the identifier for the type of node we're looking at. In this case we don't actually
        // care what type of node it is because a branch node would only ever be the final proof
        // element if (1) it includes the leaf node we're looking for or (2) it stores the value
        // within itself. If (1) then this logic will work, if (2) then this won't find anything
        // and we won't append any proof elements, which is exactly what we would want.
        const suffix = toHexString(item[0]).slice(3)
        if (key.endsWith(suffix)) {
          modifiedProof.push(toHexString(rlp.encode(item)))
        }
      }
    }
  }

  // Return the modified proof.
  return modifiedProof
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

  proof.storageProof[0].proof = maybeAddProofNode(
    ethers.utils.keccak256(slot),
    proof.storageProof[0].proof
  )

  return {
    accountProof: proof.accountProof,
    storageProof: proof.storageProof[0].proof,
    storageValue: BigNumber.from(proof.storageProof[0].value),
    storageRoot: proof.storageHash,
  }
}
