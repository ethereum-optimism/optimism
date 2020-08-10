import * as rlp from 'rlp'
import * as seedbytes from 'random-bytes-seed'
import * as seedfloat from 'seedrandom'
import { SecureTrie, BaseTrie } from 'merkle-patricia-tree'

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

export interface AccountStorageProofTest {
  address: string
  key: string
  val: string
  stateTrieWitness: string
  storageTrieWitness: string
  stateTrieRoot: string
}

export interface AccountStorageUpdateTest extends AccountStorageProofTest {
  newStateTrieRoot: string
}

export interface StateTrieProofTest {
  address: string
  encodedAccountState: string
  stateTrieWitness: string
  stateTrieRoot: string
}

export interface StateTrieUpdateTest extends StateTrieProofTest {
  newStateTrieRoot: string
}

export interface TrieNode {
  key: string
  val: string
}

export interface StateTrieNode {
  nonce: number
  balance: number
  storageRoot: string
  codeHash: string
}

export interface StateTrieMap {
  [address: string]: {
    state: StateTrieNode
    storage: TrieNode[]
  }
}

interface StateTrie {
  trie: SecureTrie
  storage: {
    [address: string]: SecureTrie | BaseTrie
  }
}

const getTrieType = (secure: boolean): any => {
  return secure ? SecureTrie : BaseTrie
}

/**
 * Utility; converts a buffer or string into a '0x'-prefixed string.
 * @param buf Element to convert.
 * @returns Converted element.
 */
export const toHexString = (buf: Buffer | string | null): string => {
  return '0x' + toHexBuffer(buf).toString('hex')
}

/**
 * Utility; converts a buffer or string into a non '0x' prefixed buffer.
 * @param buf Element to convert.
 * @returns Converted element.
 */
export const toHexBuffer = (buf: Buffer | string): Buffer => {
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
const makeTrie = async (nodes: TrieNode[], secure: boolean): Promise<BaseTrie | SecureTrie> => {
  const TrieType = getTrieType(secure)
  const trie = new TrieType()

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
  nodes: TrieNode[] | BaseTrie | SecureTrie,
  key: string,
  val?: string,
  secure = true
): Promise<ProofTest> => {
  const TrieType = getTrieType(secure)
  const trie = (nodes instanceof SecureTrie || nodes instanceof BaseTrie) ? nodes : await makeTrie(nodes, secure)

  const proof = await TrieType.prove(trie, toHexBuffer(key))
  const ret = val
    ? toHexBuffer(val)
    : await TrieType.verifyProof(trie.root, toHexBuffer(key), proof)

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
  nodes: TrieNode[],
  secure = true
): Promise<ProofTest[]> => {
  const trie = await makeTrie(nodes, secure)
  const tests: ProofTest[] = []

  for (const node of nodes) {
    tests.push(await makeProofTest(trie, node.key, undefined, secure))
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
  valSize: number = 32,
  secure = true
): Promise<ProofTest> => {
  const nodes = makeRandomNodes(germ, count, keySize, valSize)
  return makeProofTest(nodes, nodes[randomInt(germ, 0, count)].key, undefined, secure)
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
  val: string,
  secure = true
): Promise<UpdateTest> => {
  const TrieType = getTrieType(secure)
  const trie = await makeTrie(nodes, secure)

  const proof = await TrieType.prove(trie, toHexBuffer(key))
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
  valSize: number = 32,
  secure = true
): Promise<UpdateTest> => {
  const nodes = makeRandomNodes(germ, count, keySize, valSize)
  const randomBytes = seedbytes(germ)
  const newKey = randomBytes(keySize).toString('hex')
  const newVal = randomBytes(valSize).toString('hex')
  return makeUpdateTest(nodes, newKey, newVal, secure)
}

const encodeAccountState = (state: StateTrieNode): Buffer => {
  return rlp.encode([
    state.nonce,
    state.balance,
    state.storageRoot,
    state.codeHash,
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

const makeStateTrie = async (state: StateTrieMap, secure = true): Promise<StateTrie> => {
  const TrieType = getTrieType(secure)
  const stateTrie = new TrieType()
  const accountTries: { [address: string]: SecureTrie | BaseTrie } = {}

  for (const address of Object.keys(state)) {
    const account = state[address]
    accountTries[address] = await makeTrie(account.storage, secure)
    account.state.storageRoot = toHexString(accountTries[address].root)
    await stateTrie.put(toHexBuffer(address), encodeAccountState(account.state))
  }

  return {
    trie: stateTrie,
    storage: accountTries,
  }
}

export const makeAccountStorageProofTest = async (
  state: StateTrieMap,
  target: string,
  key: string,
  val?: string,
  secure = true
): Promise<AccountStorageProofTest> => {
  const TrieType = getTrieType(secure)
  const stateTrie = await makeStateTrie(state, secure)

  const storageTrie = stateTrie.storage[target]
  const storageTrieWitness = await TrieType.prove(
    storageTrie,
    toHexBuffer(key)
  )
  const ret =
    val ||
    (await SecureTrie.verifyProof(
      storageTrie.root,
      toHexBuffer(key),
      storageTrieWitness
    ))

  const stateTrieWitness = await SecureTrie.prove(
    stateTrie.trie,
    toHexBuffer(target)
  )

  return {
    address: target,
    key: toHexString(key),
    val: toHexString(ret),
    stateTrieWitness: toHexString(rlp.encode(stateTrieWitness)),
    storageTrieWitness: toHexString(rlp.encode(storageTrieWitness)),
    stateTrieRoot: toHexString(stateTrie.trie.root),
  }
}

export const makeAccountStorageUpdateTest = async (
  state: StateTrieMap,
  target: string,
  key: string,
  val: string,
  accountState?: StateTrieNode,
  secure = true
): Promise<AccountStorageUpdateTest> => {
  const TrieType = getTrieType(secure)
  const stateTrie = await makeStateTrie(state, secure)

  const storageTrie = stateTrie.storage[target]
  const storageTrieWitness = await TrieType.prove(
    storageTrie,
    toHexBuffer(key)
  )
  const stateTrieWitness = await TrieType.prove(
    stateTrie.trie,
    toHexBuffer(target)
  )

  if (!accountState) {
    await storageTrie.put(toHexBuffer(key), toHexBuffer(val))
    const encodedAccountState = await stateTrie.trie.get(toHexBuffer(target))
    accountState = decodeAccountState(encodedAccountState)
    accountState.storageRoot = toHexString(storageTrie.root)
  }

  const oldStateTrieRoot = toHexString(stateTrie.trie.root)
  await stateTrie.trie.put(
    toHexBuffer(target),
    encodeAccountState(accountState)
  )

  return {
    address: target,
    key: toHexString(key),
    val: toHexString(val),
    stateTrieWitness: toHexString(rlp.encode(stateTrieWitness)),
    storageTrieWitness: toHexString(rlp.encode(storageTrieWitness)),
    stateTrieRoot: oldStateTrieRoot,
    newStateTrieRoot: toHexString(stateTrie.trie.root),
  }
}

export const makeStateTrieProofTest = async (
  state: StateTrieMap,
  address: string,
  secure = true
): Promise<StateTrieProofTest> => {
  const TrieType = getTrieType(secure)
  const stateTrie = await makeStateTrie(state, secure)

  const stateTrieWitness = await TrieType.prove(
    stateTrie.trie,
    toHexBuffer(address)
  )

  const ret = await TrieType.verifyProof(
    stateTrie.trie.root,
    toHexBuffer(address),
    stateTrieWitness
  )

  return {
    address,
    encodedAccountState: toHexString(ret),
    stateTrieWitness: toHexString(rlp.encode(stateTrieWitness)),
    stateTrieRoot: toHexString(stateTrie.trie.root),
  }
}

export const makeStateTrieUpdateTest = async (
  state: StateTrieMap,
  address: string,
  accountState: StateTrieNode,
  secure = true
): Promise<StateTrieUpdateTest> => {
  const TrieType = getTrieType(secure)
  const stateTrie = await makeStateTrie(state, secure)

  const stateTrieWitness = await TrieType.prove(
    stateTrie.trie,
    toHexBuffer(address)
  )

  const oldStateTrieRoot = toHexString(stateTrie.trie.root)
  await stateTrie.trie.put(
    toHexBuffer(address),
    encodeAccountState(accountState)
  )

  return {
    address,
    encodedAccountState: toHexString(encodeAccountState(accountState)),
    stateTrieWitness: toHexString(rlp.encode(stateTrieWitness)),
    stateTrieRoot: oldStateTrieRoot,
    newStateTrieRoot: toHexString(stateTrie.trie.root),
  }
}

export const printTestParameters = (test: any): void => {
  console.log(Object.values(test).join(', '))
}
