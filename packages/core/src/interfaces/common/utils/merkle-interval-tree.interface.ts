export interface MerkleIntervalTreeLeafNode {
  start: BigNumber
  end: BigNumber
  data: Buffer
}

export interface MerkleIntervalTreeInternalNode {
  index: BigNumber
  hash: Buffer
}

export type MerkleIntervalTreeInclusionProof = MerkleIntervalTreeInternalNode[]
