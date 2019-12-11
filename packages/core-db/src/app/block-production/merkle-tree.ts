/* External Imports */
import {
  BIG_ENDIAN,
  BigNumber,
  getLogger,
  HashFunction,
  keccak256,
  ONE,
  runInDomain,
  TWO,
  ZERO,
} from '@pigi/core-utils'
import * as assert from 'assert'
import * as AsyncLock from 'async-lock'
import * as domain from 'domain'

/* Internal Imports */
import {
  MerkleTreeInclusionProof,
  MerkleTreeNode,
  MerkleUpdate,
  SparseMerkleTree,
} from '../../types/block-production'
import {
  Batch,
  DB,
  DEL_BATCH_TYPE,
  DelBatch,
  PUT_BATCH_TYPE,
  PutBatch,
} from '../../types/db/db.interface'

const log = getLogger('merkle-tree')

/**
 * SparseMerkleTree implementation assuming a 256-bit hash algorithm is used.
 */
export class SparseMerkleTreeImpl implements SparseMerkleTree {
  public static readonly emptyBuffer: Buffer = Buffer.alloc(32).fill('\x00')
  public static readonly siblingBuffer: Buffer = Buffer.alloc(1).fill('\x00')

  private root: MerkleTreeNode
  private zeroHashes: Buffer[]
  private static readonly lockKey: string = 'lock'
  private readonly treeLock: AsyncLock = new AsyncLock({
    domainReentrant: true,
  })
  private readonly hashFunction: (Buffer) => Buffer
  private readonly hashBuffer: Buffer = Buffer.alloc(64)

  private readonly nodeIDsToDelete: Buffer[]

  public static async create(
    db: DB,
    rootHash?: Buffer,
    height: number = 160,
    hashFunction = keccak256
  ): Promise<SparseMerkleTreeImpl> {
    assert(!rootHash || rootHash.length === 32, 'Root hash must be 32 bytes')

    const tree = new SparseMerkleTreeImpl(db, height, hashFunction)

    await tree.init(rootHash)
    return tree
  }

  private constructor(
    private db: DB,
    private height: number = 160,
    hashFunction: HashFunction = keccak256
  ) {
    assert(height > 0, 'SMT height needs to be > 0')

    // TODO: Hack for now -- change everything to string if/when it makes sense
    this.hashFunction = (buff: Buffer) =>
      Buffer.from(hashFunction(buff.toString('hex')), 'hex')

    this.nodeIDsToDelete = []
  }

  private async init(rootHash?: Buffer): Promise<void> {
    await this.populateZeroHashesAndRoot(rootHash)
  }

  public getHeight(): number {
    return this.height
  }

  public async getRootHash(): Promise<Buffer> {
    const copy: Buffer = Buffer.alloc(this.root.hash.length)
    log.debug(
      `Root Hash: [${this.root.hash.toString('hex')}], value: [${
        !!this.root.value ? this.root.value.toString('hex') : 'undefined'
      }]`
    )
    this.root.hash.copy(copy)
    return copy
  }

  public async purgeOldNodes(): Promise<void> {
    const batch: DelBatch[] = this.nodeIDsToDelete.map((id) => {
      return {
        type: DEL_BATCH_TYPE,
        key: id,
      }
    })
    await this.db.batch(batch)
    this.nodeIDsToDelete.length = 0
  }

  public async getLeaf(leafKey: BigNumber, rootHash?: Buffer): Promise<Buffer> {
    log.debug(`Trying to get leaf [${leafKey.toString(10)}]`)
    return this.treeLock.acquire(SparseMerkleTreeImpl.lockKey, async () => {
      if (!!rootHash && !rootHash.equals(this.root.hash)) {
        log.debug(
          `Cannot get Leaf [${leafKey.toString(
            10
          )}] because root hash does not match.`
        )
        return undefined
      }

      const nodesInPath: MerkleTreeNode[] = await this.getNodesInPath(leafKey)
      if (!nodesInPath || nodesInPath.length !== this.height) {
        log.debug(
          `Cannot get Leaf [${leafKey.toString(
            10
          )}] because nodes in path does not equal tree height. Expected [${
            this.height
          }], found [${nodesInPath.length}]`
        )
        return undefined
      }
      const leaf: MerkleTreeNode = nodesInPath[nodesInPath.length - 1]

      // Will only match if we were able to traverse all the way to the leaf
      if (!leaf.key.equals(leafKey)) {
        log.debug(
          `Cannot get Leaf because leaf key does not match. Path: [${leafKey.toString(
            10
          )}], leaf key: ${leaf.key.toString(10)}.`
        )
        return undefined
      }

      log.debug(`Returning leaf value: [${leaf.value.toString()}].`)
      return leaf.value
    })
  }

  public async verifyAndStore(
    inclusionProof: MerkleTreeInclusionProof
  ): Promise<boolean> {
    // There should be one sibling for every node except the root.
    if (inclusionProof.siblings.length !== this.height - 1) {
      log.info(
        `verifyAndStore: inclusion proof siblings length [${
          inclusionProof.siblings.length
        }] != tree height -1 [${this.height - 1}]`
      )
      return false
    }

    return this.treeLock.acquire(SparseMerkleTreeImpl.lockKey, async () => {
      const leafHash: Buffer = this.hashFunction(inclusionProof.value)
      if (!!(await this.getNode(leafHash, inclusionProof.key))) {
        return true
      }

      let child: MerkleTreeNode = this.createNode(
        leafHash,
        inclusionProof.value,
        inclusionProof.key
      )

      let siblingIndex = 0
      let parent: MerkleTreeNode = child
      const nodesToStore: MerkleTreeNode[] = [child]
      for (let parentDepth = this.height - 2; parentDepth >= 0; parentDepth--) {
        child = parent

        const childDepth: number = parentDepth + 1
        // Since there's no root sibling, each sibling is one index lower
        const childSiblingHash: Buffer = inclusionProof.siblings[siblingIndex++]
        parent = this.calculateParentNode(
          child,
          childSiblingHash,
          inclusionProof.key,
          parentDepth
        )
        nodesToStore.push(parent)

        // Store sibling node, but don't overwrite it if it's in the db.
        const siblingNode: MerkleTreeNode = await this.createProofSiblingNodeIfDoesntExist(
          childSiblingHash,
          inclusionProof.key,
          childDepth
        )
        if (!!siblingNode) {
          nodesToStore.push(siblingNode)
        }
      }

      if (!parent.hash.equals(this.root.hash)) {
        log.info(
          `verifyAndStore: Parent hash [${parent.hash.toString(
            'hex'
          )}] does not equal root hash [${this.root.hash.toString('hex')}].`
        )
        return false
      }

      const toDelete: MerkleTreeNode[] = await this.getNodesInPath(
        inclusionProof.key
      )

      await this.db.batch(this.getNodePutBatch(nodesToStore))

      // Root hash will not change, but it might have gone from a shortcut to regular node.
      this.root = parent

      this.nodeIDsToDelete.push(...toDelete.map((n) => this.getNodeID(n)))
      return true
    })
  }

  public async update(
    leafKey: BigNumber,
    leafValue: Buffer,
    d?: domain.Domain
  ): Promise<boolean> {
    return runInDomain(d, async () => {
      return this.treeLock.acquire(SparseMerkleTreeImpl.lockKey, async () => {
        let nodesToUpdate: MerkleTreeNode[] = await this.getNodesInPath(leafKey)

        if (!nodesToUpdate) {
          log.info(
            `Unable to update tree leaf ${leafKey.toString(
              'hex'
            )} to ${leafValue.toString(
              'hex'
            )} because there are no nodes in the leaf path.`
          )
          return false
        } else if (nodesToUpdate.length !== this.height) {
          log.debug(`nodes to update length != height`)
          if (
            !(await this.verifyAndStorePartiallyEmptyPath(
              leafKey,
              nodesToUpdate.length
            ))
          ) {
            log.info(
              `Unable to update tree leaf ${leafKey.toString(
                'hex'
              )} to ${leafValue.toString(
                'hex'
              )} because unable to verify and store partially empty path.`
            )
            return false
          }
          nodesToUpdate = await this.getNodesInPath(leafKey)
        }

        const oldLeaf: MerkleTreeNode = nodesToUpdate[nodesToUpdate.length - 1]
        // This means the update is to the existing value -- just return
        if (!!oldLeaf && !!oldLeaf.value && leafValue.equals(oldLeaf.value)) {
          log.info(
            `Trying to update tree leaf ${leafKey.toString(
              'hex'
            )} to the same value it currently has ${leafValue.toString(
              'hex'
            )}. Returning.`
          )
          return true
        }

        const idsToDelete: Buffer[] = [this.getNodeID(oldLeaf)]

        let updatedChild: MerkleTreeNode = {
          key: leafKey,
          value: leafValue,
          hash: this.hashFunction(leafValue),
        }

        const nodesToSave: MerkleTreeNode[] = [updatedChild]
        let parentDepth: number = nodesToUpdate.length - 2 // -2 because this array also contains the leaf

        // Iteratively update all nodes from the leaf-pointer node up to the root
        for (; parentDepth >= 0; parentDepth--) {
          idsToDelete.push(this.getNodeID(nodesToUpdate[parentDepth]))
          updatedChild = this.updateNode(
            nodesToUpdate[parentDepth],
            updatedChild,
            leafKey,
            parentDepth
          )
          nodesToSave.push(updatedChild)
        }

        await this.db.batch(this.getNodePutBatch(nodesToSave))

        this.nodeIDsToDelete.push(...idsToDelete)

        this.root = updatedChild

        log.debug(
          `Tree leaf 
          ${leafKey.toString('hex')} updated to value
          ${leafValue.toString('hex')} resulting in new root hash: 
          ${this.root.hash.toString('hex')}, value: 
          ${this.root.value.toString('hex')}.`
        )

        return true
      })
    })
  }

  public async batchUpdate(updates: MerkleUpdate[]): Promise<boolean> {
    const d: domain.Domain = domain.create()

    return runInDomain(d, () => {
      return this.treeLock.acquire(SparseMerkleTreeImpl.lockKey, async () => {
        for (const update of updates) {
          if (
            !(await this.verifyAndStore({
              rootHash: this.root.hash,
              key: update.key,
              value: update.oldValue,
              siblings: update.oldValueProofSiblings,
            }))
          ) {
            return false
          }
        }

        for (const update of updates) {
          if (!(await this.update(update.key, update.newValue, d))) {
            throw Error(
              "Verify and Store worked but update didn't! This should never happen!"
            )
          }
        }

        return true
      })
    })
  }

  public async getMerkleProof(
    leafKey: BigNumber,
    leafValue: Buffer
  ): Promise<MerkleTreeInclusionProof> {
    const result: MerkleTreeInclusionProof = await this.treeLock.acquire(
      SparseMerkleTreeImpl.lockKey,
      async () => {
        if (!this.root || !this.root.hash) {
          return undefined
        }

        let node: MerkleTreeNode = this.root
        const siblings: Buffer[] = []
        for (
          let depth = 0;
          depth < this.height &&
          !!node &&
          !!node.value &&
          node.value.length === 64;
          depth++
        ) {
          siblings.push(this.getChildSiblingHash(node, depth, leafKey))
          node = await this.getChild(node, depth, leafKey)
        }

        if (siblings.length !== this.height - 1) {
          // TODO: A much better way of indicating this
          log.info(
            `Getting merkle proof for leaf that has not been stored. Storing siblings and leaf and creating proof.`
          )
          return {
            rootHash: undefined,
            key: undefined,
            value: undefined,
            siblings: undefined,
          }
        }

        const leafHash: Buffer = this.hashFunction(leafValue)
        if (!node.hash.equals(leafHash)) {
          log.info(
            `Node hash does not match leaf value hash for key [${leafKey.toNumber()}]. Passed leaf value: [${leafValue.toString()}], Passed leaf hash: ${leafHash.toString(
              'hex'
            )}, Node hash: ${node.hash.toString('hex')}`
          )
          // Provided leaf doesn't match stored leaf
          return undefined
        }

        return {
          rootHash: this.root.hash,
          key: leafKey,
          value: leafValue,
          siblings: siblings.reverse(),
        }
      }
    )

    if (!result || !!result.rootHash) {
      return result
    }

    // If this is for an empty leaf, we can store it and create a MerkleProof
    if (leafValue.equals(SparseMerkleTreeImpl.emptyBuffer)) {
      if (await this.verifyAndStorePartiallyEmptyPath(leafKey)) {
        return this.getMerkleProof(leafKey, leafValue)
      }
    }
    return undefined
  }

  public async verifyAndStorePartiallyEmptyPath(
    leafKey: BigNumber,
    numExistingNodes?: number
  ): Promise<boolean> {
    if (numExistingNodes === undefined) {
      numExistingNodes = (await this.getNodesInPath(leafKey)).length
    }
    const existingChildren: number = Math.max(numExistingNodes - 1, 0)

    const siblings: Buffer[] = []
    let node: MerkleTreeNode = this.root
    for (let i = 0; i < this.height - 1; i++) {
      if (
        i > existingChildren ||
        (i === existingChildren && (!node.value || node.value.length !== 64))
      ) {
        siblings.push(...this.zeroHashes.slice(i + 1))
        break
      }

      siblings.push(this.getChildSiblingHash(node, i, leafKey))
      node = await this.getChild(node, i, leafKey)
    }

    return this.verifyAndStore({
      rootHash: this.root.hash,
      key: leafKey,
      value: SparseMerkleTreeImpl.emptyBuffer,
      siblings: siblings.reverse(),
    })
  }

  /**
   * Gets the provided parent node's child's sibling hash based on the provided
   * leafKey. If the leafKey path is through he left child, this will get the right
   * and vice-versa.
   *
   * @param parent The node whose child sibling this will get.
   * @param parentDepth The depth of the parent.
   * @param leafKey The leaf key helping determine the sibling.
   * @returns The child sibling hash.
   */
  private getChildSiblingHash(
    parent: MerkleTreeNode,
    parentDepth: number,
    leafKey: BigNumber
  ): Buffer {
    const isLeft: boolean = this.isLeft(leafKey, parentDepth)
    return isLeft ? parent.value.subarray(32) : parent.value.subarray(0, 32)
  }

  /**
   * Gets the provided parent node's child following the path specified by the
   *  provided leafKey.
   *
   * @param parent The node whose child this will get.
   * @param parentDepth The depth of the parent.
   * @param leafKey The leaf key specifying the path to the child.
   * @returns The child if one is present.
   */
  private async getChild(
    parent: MerkleTreeNode,
    parentDepth: number,
    leafKey: BigNumber
  ): Promise<MerkleTreeNode> {
    const childIndex: number = this.isLeft(leafKey, parentDepth) ? 0 : 32
    const childHash: Buffer = parent.value.subarray(childIndex, childIndex + 32)
    return this.getNode(childHash, this.getNodeKey(leafKey, ++parentDepth))
  }

  /**
   * Gets an array of MerkleTreeNodes starting at the root and iterating down to the leaf
   * following the path in the provided key. The returned array will omit any nodes that
   * are not persisted because they can be calculated from the leaf and the zeroHashes.
   *
   * NOTE: If the tree is modified in parallel with a call to this function,
   * results are non-deterministic.
   *
   * @param leafKey The key describing the path to the leaf in question
   * @returns The array of MerkleTreeNodes from root to leaf
   */
  private async getNodesInPath(leafKey: BigNumber): Promise<MerkleTreeNode[]> {
    if (!this.root || !this.root.hash) {
      return []
    }
    if (!this.root.value) {
      return [this.root]
    }

    let node: MerkleTreeNode = this.root
    const nodesToUpdate: MerkleTreeNode[] = [node]

    let depth
    for (depth = 0; depth < this.height - 1; depth++) {
      const childDepth: number = depth + 1
      if (node.value.length === 64) {
        // This is a standard node
        node = this.isLeft(leafKey, depth)
          ? await this.getNode(
              node.value.subarray(0, 32),
              this.getNodeKey(leafKey, childDepth)
            )
          : await this.getNode(
              node.value.subarray(32),
              this.getNodeKey(leafKey, childDepth)
            )
        if (node) {
          nodesToUpdate.push(node)
        } else {
          break
        }
      } else {
        // This is malformed or a disconnected sibling node
        break
      }
    }
    return nodesToUpdate
  }

  /**
   * Updates the provided MerkleTreeNode based on the provided updated child node.
   *
   * @param node The node to update
   * @param updatedChild The child of the node to update that has changed
   * @param key The key for the updated leaf
   * @param depth the depth of the node
   * @returns The updated node
   */
  private updateNode(
    node: MerkleTreeNode,
    updatedChild: MerkleTreeNode,
    key: BigNumber,
    depth: number
  ): MerkleTreeNode {
    const value = Buffer.alloc(64, node.value)
    const isLeft: boolean = this.isLeft(key, depth)
    if (isLeft) {
      value.fill(updatedChild.hash, 0, 32)
    } else {
      value.fill(updatedChild.hash, 32)
    }

    return {
      key: node.key,
      value,
      hash: this.hashFunction(value),
    }
  }

  /**
   * Creates a Merkle Proof sibling node if a node with this hash has not already been stored
   * in the DB.
   *
   * NOTE: If the tree is modified in parallel with a call to this function,
   * results are non-deterministic.
   *
   * @param nodeHash The hash of the node to create if not already present.
   * @param leafKey The key detailing how to get to this node from the root
   * @param depth The depth of this node in the tree
   * @returns The created node if one was created or undefined if one already exists.
   */
  private async createProofSiblingNodeIfDoesntExist(
    nodeHash: Buffer,
    leafKey: BigNumber,
    depth: number
  ): Promise<MerkleTreeNode> {
    // Need to XOR with 1 because this represents a sibling.
    const nodeKey: BigNumber = this.getNodeKey(leafKey, depth).xor(ONE)
    const node: MerkleTreeNode = await this.getNode(nodeHash, nodeKey)
    if (!!node) {
      return undefined
    }
    return this.createNode(
      nodeHash,
      SparseMerkleTreeImpl.siblingBuffer,
      nodeKey
    )
  }

  /**
   * Gets the MerkleTreeNode with the provided hash from the DB, if one exists.
   *
   * @param nodeHash The node hash uniquely identifying the node
   * @param nodeKey The key identifying the location of the node in question
   * @returns The node, if one was found
   */
  private async getNode(
    nodeHash: Buffer,
    nodeKey: BigNumber
  ): Promise<MerkleTreeNode> {
    const value: Buffer = await this.db.get(
      this.getNodeIDFromHashAndKey(nodeHash, nodeKey)
    )
    if (!value) {
      return undefined
    }
    return this.createNode(nodeHash, value, nodeKey)
  }

  /**
   * Calculates the parent hash from the provided node and sibling hash, using the key and depth
   * to determine whether the node is the left node or the sibling is the left node.
   *
   * @param node The node whose hash is used as 1/2 input to parent calculation
   * @param siblingHash The sibling node hash used as 1/2 input to parent calculation
   * @param leafKey The key representing the path to a leaf from which we started
   * @param depth The depth of this node
   * @returns The parent node
   */
  private calculateParentNode(
    node: MerkleTreeNode,
    siblingHash: Buffer,
    leafKey: BigNumber,
    depth: number
  ): MerkleTreeNode {
    const value = Buffer.alloc(64)
    if (this.isLeft(leafKey, depth)) {
      this.hashBuffer
        .fill(node.hash, 0, 32)
        .fill(siblingHash, 32)
        .copy(value)
    } else {
      this.hashBuffer
        .fill(siblingHash, 0, 32)
        .fill(node.hash, 32)
        .copy(value)
    }

    return this.createNode(
      this.hashFunction(value),
      value,
      this.getNodeKey(leafKey, depth)
    )
  }

  /**
   * Populates the zero-hash array for each level of the Sparse Merkle Tree
   * and stores the resulting root.
   *
   * @param rootHash The optional root hash to assign the tree
   */
  private async populateZeroHashesAndRoot(rootHash?: Buffer): Promise<void> {
    const hashes: Buffer[] = [
      this.hashFunction(SparseMerkleTreeImpl.emptyBuffer),
    ]

    for (let i = 1; i < this.height; i++) {
      hashes[i] = this.hashFunction(
        this.hashBuffer.fill(hashes[i - 1], 0, 32).fill(hashes[i - 1], 32)
      )
    }

    this.zeroHashes = hashes.reverse()

    if (!!rootHash) {
      log.info(
        `Attempting to initialize SMT with root hash ${rootHash.toString(
          'hex'
        )} and DB ID: ${this.getNodeIDFromHashAndKey(rootHash, ZERO).toString(
          'hex'
        )}`
      )

      this.root = await this.getNode(rootHash, ZERO)
    }

    if (!this.root) {
      this.root = this.createNode(
        rootHash || this.zeroHashes[0],
        undefined,
        ZERO
      )
      log.info(
        `Initialized Sparse Merkle Tree with root ${(
          rootHash || this.zeroHashes[0]
        ).toString('hex')}`
      )
    } else {
      log.info(
        `Initialized Sparse Merkle Tree with root hash [${this.root.hash.toString(
          'hex'
        )}] and root value: [${
          !!this.root.value ? this.root.value.toString('hex') : 'undefined'
        }]`
      )
    }
  }

  /**
   * Helper function to create a MerkleTreeNode from the provided hash, value, and key
   *
   * @param hash The hash
   * @param value The value
   * @param key The key that describes how to get to this node from the tree root
   * @returns The resulting MerkleTreeNode
   */
  private createNode(
    hash: Buffer,
    value: Buffer,
    key: BigNumber
  ): MerkleTreeNode {
    return { hash, value, key }
  }

  /**
   * Determines whether or not the key at the provided depth points to the left child or right child.
   *
   * @param key The key
   * @param depth The depth
   * @returns true if the key points to the left child at the provided depth, false if right
   */
  private isLeft(key: BigNumber, depth: number): boolean {
    return key
      .shiftLeft(depth)
      .shiftRight(this.height - 2)
      .mod(TWO)
      .equals(ZERO)
  }

  private getNodeKey(leafKey: BigNumber, depth: number): BigNumber {
    return leafKey.shiftRight(this.height - depth - 1)
  }

  /**
   * Gets the unique ID for the provided node used for lookup in the DB.
   *
   * @param node The node in question
   */
  private getNodeID(node: MerkleTreeNode): Buffer {
    return this.getNodeIDFromHashAndKey(node.hash, node.key)
  }

  /**
   * Convenience function for consistency -- calculates the node ID from the provided info.
   *
   * @param nodeHash The hash of the node for which the ID will be calculated.
   * @param nodeKey The key of the node for which the ID will be calculated.
   * @returns the node's ID
   */
  private getNodeIDFromHashAndKey(
    nodeHash: Buffer,
    nodeKey: BigNumber
  ): Buffer {
    return this.hashFunction(
      this.hashBuffer
        .fill(nodeHash, 0, 32)
        .fill(this.hashFunction(nodeKey.toBuffer(BIG_ENDIAN)), 32)
    )
  }

  /**
   * Gets the batch update for a collection of MerkleTreeNodes.
   *
   * @param nodes The nodes to be batched
   * @returns The PutBatch
   */
  private getNodePutBatch(nodes: MerkleTreeNode[]): PutBatch[] {
    return nodes.map((n) => {
      const nodeId: Buffer = this.getNodeID(n)
      log.debug(
        `Setting nodeID: ${nodeId.toString('hex')} to value: ${n.value.toString(
          'hex'
        )} (hash: ${this.hashFunction(n.value).toString('hex')}`
      )
      return {
        type: PUT_BATCH_TYPE,
        key: this.getNodeID(n),
        value: n.value,
      }
    })
  }
}
