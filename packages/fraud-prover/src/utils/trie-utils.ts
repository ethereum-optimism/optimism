import { BaseTrie } from 'merkle-patricia-tree'
import { fromHexString } from '@eth-optimism/core-utils'

export const makeTrieFromProofs = (proofs: string[][]): Promise<BaseTrie> => {
  if (
    proofs.length === 0 ||
    proofs.every((proof) => {
      return proof.length === 0
    })
  ) {
    return BaseTrie.fromProof([])
  }

  const nodes = proofs.reduce(
    // tslint:disable-next-line
    (nodes, proof) => {
      if (proof.length > 1) {
        return nodes.concat(proof.slice(1))
      } else {
        return nodes
      }
    },
    [proofs[0][0]]
  )

  return BaseTrie.fromProof(
    nodes.map((node) => {
      return fromHexString(node)
    })
  )
}
