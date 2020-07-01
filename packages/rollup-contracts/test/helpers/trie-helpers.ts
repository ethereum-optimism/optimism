import * as rlp from 'rlp'
import * as seedbytes from 'random-bytes-seed'
import * as seedfloat from 'seedrandom'
import { BaseTrie } from 'merkle-patricia-tree'

interface UpdateTest {
  proof: string
  key: string
  val: string
  oldRoot: string
  newRoot: string
}

interface ProofTest {
  proof: string
  key: string
  val: string
  root: string
}

interface TrieNode {
  key: string
  val: string
}

/**
 * Utility; converts a buffer or string into a '0x'-prefixed string.
 * @param buf Element to convert.
 * @returns Converted element.
 */
const toHexString = (buf: Buffer | string): string => {
  return '0x' + Buffer.from(buf).toString('hex')
}

/**
 * Utility; generates a random integer.
 * @param seed Seed to the random number generator.
 * @param min Minimum for the RNG.
 * @param max Maximum for the RNG.
 * @returns Random integer between minimum and maximum.
 */
const randomInt = (seed: string, min: number, max: number): number => {
  const randomFloat = seedfloat(seed)
  min = Math.ceil(min)
  max = Math.floor(max)
  return Math.floor(randomFloat() * (max - min + 1)) + min
}

/**
 * Utility; creates a trie object from a list of nodes.
 * @param nodes Nodes to seed the trie with.
 * @returns Trie corresponding to the given nodes.
 */
const makeTrie = async (nodes: TrieNode[]): Promise<BaseTrie> => {
  const trie = new BaseTrie()

  for (const node of nodes) {
    await trie.put(Buffer.from(node.key), Buffer.from(node.val))
  }

  return trie
}

/**
 * Utility; generates random nodes.
 * @param germ Seed to the random number generator.
 * @param count Number of nodes to generate.
 * @param keySize Size of the key for each node in bytes.
 * @param valSize Size of the value for each node in bytes.
 * @returns List of randomly generated nodes.
 */
const makeRandomNodes = (
  germ: string,
  count: number,
  keySize: number = 32,
  valSize: number = 32
): TrieNode[] => {
  const randomBytes = seedbytes(germ)
  const nodes: TrieNode[] = Array(count)
    .fill({})
    .map(() => {
      return {
        key: randomBytes(keySize).toString('hex'),
        val: randomBytes(valSize).toString('hex'),
      }
    })
  return nodes
}

/**
 * Generates inclusion/exclusion proof test parameters.
 * @param nodes Nodes of the trie, or the trie itself.
 * @param key Key to prove inclusion/exclusion for.
 * @param val Value to prove inclusion/exclusion for.
 * @returns Proof test parameters.
 */
export const makeProofTest = async (
  nodes: TrieNode[] | BaseTrie,
  key: string,
  val?: string
): Promise<ProofTest> => {
  const trie = nodes instanceof BaseTrie ? nodes : await makeTrie(nodes)

  const proof = await BaseTrie.prove(trie, Buffer.from(key))
  const ret = val
    ? Buffer.from(val)
    : await BaseTrie.verifyProof(trie.root, Buffer.from(key), proof)

  return {
    proof: toHexString(rlp.encode(proof)),
    key: toHexString(key),
    val: toHexString(ret),
    root: toHexString(trie.root),
  }
}

/**
 * Automatically generates all possible leaf node inclusion proof tests.
 * @param nodes Nodes to generate tests for.
 * @returns All leaf node tests for the given nodes.
 */
export const makeAllProofTests = async (
  nodes: TrieNode[]
): Promise<ProofTest[]> => {
  const trie = await makeTrie(nodes)
  const tests: ProofTest[] = []

  for (const node of nodes) {
    tests.push(await makeProofTest(trie, node.key))
  }

  return tests
}

/**
 * Generates a random inclusion proof test.
 * @param germ Seed to the random number generator.
 * @param count Number of nodes to create.
 * @param keySize Key size in bytes.
 * @param valSize Value size in bytes.
 * @return Proof test parameters for the randomly generated nodes.
 */
export const makeRandomProofTest = async (
  germ: string,
  count: number,
  keySize: number = 32,
  valSize: number = 32
): Promise<ProofTest> => {
  const nodes = makeRandomNodes(germ, count, keySize, valSize)
  return makeProofTest(nodes, nodes[randomInt(germ, 0, count)].key)
}

/**
 * Generates update test parameters.
 * @param nodes Nodes in the trie.
 * @param key Key to update.
 * @param val Value to update.
 * @returns Update test parameters.
 */
export const makeUpdateTest = async (
  nodes: TrieNode[],
  key: string,
  val: string
): Promise<UpdateTest> => {
  const trie = await makeTrie(nodes)

  const proof = await BaseTrie.prove(trie, Buffer.from(key))
  const oldRoot = Buffer.from(trie.root)

  await trie.put(Buffer.from(key), Buffer.from(val))

  return {
    proof: toHexString(rlp.encode(proof)),
    key: toHexString(key),
    val: toHexString(val),
    oldRoot: toHexString(oldRoot),
    newRoot: toHexString(trie.root),
  }
}

/**
 * Generates a random update test.
 * @param germ Seed to the random number generator.
 * @param count Number of nodes to create.
 * @param keySize Key size in bytes.
 * @param valSize Value size in bytes.
 * @return Update test parameters for the randomly generated nodes.
 */
export const makeRandomUpdateTest = async (
  germ: string,
  count: number,
  keySize: number = 32,
  valSize: number = 32
): Promise<UpdateTest> => {
  const nodes = makeRandomNodes(germ, count, keySize, valSize)
  const randomBytes = seedbytes(germ)
  const newKey = randomBytes(keySize).toString('hex')
  const newVal = randomBytes(valSize).toString('hex')
  return makeUpdateTest(nodes, newKey, newVal)
}
