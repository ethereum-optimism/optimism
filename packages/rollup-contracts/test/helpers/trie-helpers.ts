import * as rlp from 'rlp'
import * as seedbytes from 'random-bytes-seed'
import * as seedfloat from 'seedrandom'
import { BaseTrie } from 'merkle-patricia-tree'
import { encode } from 'punycode'

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

interface AccountStorageProofTest {
  address: string
  key: string
  val: string
  stateTrieWitness: string
  storageTrieWitness: string
  stateTrieRoot: string
}

interface AccountStorageUpdateTest extends AccountStorageProofTest {
  newStateTrieRoot: string
}

interface TrieNode {
  key: string
  val: string
}

interface StateTrieNode {
  nonce: number
  balance: number
  storageRoot: string
  codeHash: string
}

interface StateTrieMap {
  [address: string]: {
    state: StateTrieNode
    storage: TrieNode[]
  }
}

interface StateTrie {
  trie: BaseTrie
  storage: {
    [address: string]: BaseTrie
  }
}

/**
 * Utility; converts a buffer or string into a '0x'-prefixed string.
 * @param buf Element to convert.
 * @returns Converted element.
 */
const toHexString = (buf: Buffer | string | null): string => {
  return '0x' + toHexBuffer(buf).toString('hex')
}

const toHexBuffer = (buf: Buffer | string): Buffer => {
  if (typeof buf === 'string' && buf.startsWith('0x')) {
    return Buffer.from(buf.slice(2), 'hex')
  }

  return Buffer.from(buf)
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
    await trie.put(toHexBuffer(node.key), toHexBuffer(node.val))
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

  const proof = await BaseTrie.prove(trie, toHexBuffer(key))
  const ret = val
    ? toHexBuffer(val)
    : await BaseTrie.verifyProof(trie.root, toHexBuffer(key), proof)

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

  const proof = await BaseTrie.prove(trie, toHexBuffer(key))
  const oldRoot = toHexBuffer(trie.root)

  await trie.put(toHexBuffer(key), toHexBuffer(val))

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

const encodeAccountState = (state: StateTrieNode): Buffer => {
  return rlp.encode([
    state.nonce,
    state.balance,
    state.storageRoot,
    state.codeHash
  ])
}

const decodeAccountState = (state: Buffer): StateTrieNode => {
  const decoded = rlp.decode(state) as any
  return {
    nonce: decoded[0].length ? parseInt(toHexString(decoded[0]), 16) : 0,
    balance: decoded[1].length ? parseInt(toHexString(decoded[1]), 16) : 0,
    storageRoot: decoded[2].length ? toHexString(decoded[2]) : null,
    codeHash: decoded[3].length ? toHexString(decoded[3]) : null,
  }
}

const makeStateTrie = async (
  state: StateTrieMap
): Promise<StateTrie> => {
  const stateTrie = new BaseTrie();
  const accountTries: { [address: string]: BaseTrie } = {};

  for (const address of Object.keys(state)) {
    const account = state[address]
    accountTries[address] = await makeTrie(account.storage);
    account.state.storageRoot = toHexString(accountTries[address].root);
    await stateTrie.put(toHexBuffer(address), encodeAccountState(account.state));
  }

  return {
    trie: stateTrie,
    storage: accountTries
  }
}

export const makeAccountStorageProofTest = async (
  state: StateTrieMap,
  target: string,
  key: string,
  val?: string
): Promise<AccountStorageProofTest> => {
  const stateTrie = await makeStateTrie(state)
  
  const storageTrie = stateTrie.storage[target]
  const storageTrieWitness = await BaseTrie.prove(storageTrie, toHexBuffer(key))
  const ret = val || await BaseTrie.verifyProof(storageTrie.root, toHexBuffer(key), storageTrieWitness)

  const stateTrieWitness = await BaseTrie.prove(stateTrie.trie, toHexBuffer(target));

  return {
    address: target,
    key: toHexString(key),
    val: toHexString(ret),
    stateTrieWitness: toHexString(rlp.encode(stateTrieWitness)),
    storageTrieWitness: toHexString(rlp.encode(storageTrieWitness)),
    stateTrieRoot: toHexString(stateTrie.trie.root)
  }
}

export const makeAccountStorageUpdateTest = async (
  state: StateTrieMap,
  target: string,
  key: string,
  val: string,
  accountState?: StateTrieNode,
): Promise<AccountStorageUpdateTest> => {
  const stateTrie = await makeStateTrie(state)
  
  const storageTrie = stateTrie.storage[target]
  const storageTrieWitness = await BaseTrie.prove(storageTrie, toHexBuffer(key))
  const stateTrieWitness = await BaseTrie.prove(stateTrie.trie, toHexBuffer(target))

  if (!accountState) {
    await storageTrie.put(toHexBuffer(key), toHexBuffer(val))
    const encodedAccountState = await stateTrie.trie.get(toHexBuffer(target))
    accountState = decodeAccountState(encodedAccountState)
    accountState.storageRoot = toHexString(storageTrie.root)
  }

  const oldStateTrieRoot = toHexString(stateTrie.trie.root)
  await stateTrie.trie.put(toHexBuffer(target), encodeAccountState(accountState))

  return {
    address: target,
    key: toHexString(key),
    val: toHexString(val),
    stateTrieWitness: toHexString(rlp.encode(stateTrieWitness)),
    storageTrieWitness: toHexString(rlp.encode(storageTrieWitness)),
    stateTrieRoot: oldStateTrieRoot,
    newStateTrieRoot: toHexString(stateTrie.trie.root)
  }
}

export const printTestParameters = (test: any): void => {
  console.log(Object.values(test).join(', '))
}