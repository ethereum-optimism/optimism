import { BigNumber } from '@eth-optimism/core-utils'

export interface MerkleTreeNode {
  key: BigNumber
  hash: Buffer
  value: Buffer
}

export interface MerkleTreeInclusionProof {
  rootHash: Buffer
  key: BigNumber
  value: Buffer
  siblings: Buffer[]
}

export interface MerkleUpdate {
  key: BigNumber
  oldValue: Buffer
  oldValueProofSiblings: Buffer[]
  newValue: Buffer
}

export interface MerkleTree {
  /**
   * Gets the root hash for this tree.
   *
   * @returns The root hash.
   */
  getRootHash(): Promise<Buffer>

  /**
   * Updates the provided key in the Merkle Tree to have the value as data,
   * including all ancestors' hashes that result from this modification.
   *
   * @param leafKey The leaf key to update
   * @param leafValue The new value
   * @param purgeOldNodes Whether or not to delete nodes deprecated by this update
   * @return true if the update succeeded, false if we're missing the intermediate nodes / siblings required for this
   */
  update(
    leafKey: BigNumber,
    leafValue: Buffer,
    purgeOldNodes: boolean
  ): Promise<boolean>

  /**
   * Updates the provided keys in the Merkle Tree in an atomic fashion
   * including all ancestor hashes that result from these modifications.
   *
   * Note: It is known that applying one update invalidates the proof for the next
   * update, which should be accounted for within this method.
   *
   * @param updates The updates to execute
   * @return true if the update succeeded, false if we're missing the intermediate nodes / siblings required for this
   */
  batchUpdate(updates: MerkleUpdate[]): Promise<boolean>

  /**
   * Gets a Merkle proof for the provided leaf value at the provided key in the tree.
   *
   * @param leafKey The exact path from the root to the leaf value in question
   * @param leafValue The leaf data
   * @returns The MerkleTreeInclusionProof if one is possible, else undefined
   * @throws If data required to calculate the Merkle proof is missing
   */
  getMerkleProof(
    leafKey: BigNumber,
    leafValue: Buffer
  ): Promise<MerkleTreeInclusionProof>

  /**
   * Gets the leaf data at the provided key in the tree, if any exists.
   *
   * @param leafKey The key of the leaf to fetch
   * @param rootHash: The optional root hash if root verification is desired
   * @returns The value at the key if one exists, else undefined
   */
  getLeaf(leafKey: BigNumber, rootHash?: Buffer): Promise<Buffer>

  /**
   * Gets the height of the Merkle tree, including the root node.
   *
   * @returns the height
   */
  getHeight(): number

  /**
   * Purges old nodes that have been queued for deletion.
   *
   * Background: Since we may want recoverability across multiple tree operations,
   * we queue nodes for deletion instead of deleting them eagerly. Once the change
   * fully completes from the caller's perspective, they may call this to delete the
   * nodes that would make the tree recoverable from a previous state root.
   */
  purgeOldNodes(): Promise<void>
}

export interface SparseMerkleTree extends MerkleTree {
  /**
   * Verifies that the provided inclusion proof and stores the
   * associated siblings for future updates / calculations.
   *
   * @param inclusionProof The inclusion proof in question
   * @return true if the proof was valid (and thus stored), false otherwise
   */
  verifyAndStore(inclusionProof: MerkleTreeInclusionProof): Promise<boolean>

  /**
   * Verifies and stores an empty leaf from a partially non-existent path.
   *
   * @param leafKey The leaf to store
   * @param numExistingNodes The number of existing nodes, if known
   * @returns True if verified, false otherwise
   */
  verifyAndStorePartiallyEmptyPath(
    leafKey: BigNumber,
    numExistingNodes?: number
  ): Promise<boolean>
}
