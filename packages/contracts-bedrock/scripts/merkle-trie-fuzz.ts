import { ethers } from 'ethers'
import { Trie } from '@ethereumjs/trie'

////////////////////////////////////////////////////////////////
//                         INPUT GEN                          //
////////////////////////////////////////////////////////////////

const genValidFuzzInput = async (): Promise<void> => {
  // Create an empty trie and determine a random number
  // of key-value pairs to insert into it.
  const trie = new Trie()
  const nElem = randRange(2, 1024)
  const randElem = randRange(0, nElem)

  let key: Buffer, value: Buffer

  // Insert random key-value pairs into our trie
  for (let i = 0; i < nElem; i++) {
    let a = Buffer.from(ethers.utils.randomBytes(32))
    let b = Buffer.from(ethers.utils.randomBytes(32))

    // Randomly select a key / value pair to create a
    // proof of inclusion for.
    if (i == randElem) {
      key = a
      value = b
    }

    // Insert our randomly generated key / value pair
    await trie.put(a, b)
  }

  // Grab the trie's root
  const root = bufferToHex(trie.root)

  // Grab the proof of inclusion for `key` in `trie`
  const proof = (await Trie.createProof(trie, key!)).map(bufferToHex)

  // Print the abi-encoded test case
  process.stdout.write(
    encodeOutput(root, bufferToHex(key!), bufferToHex(value!), proof)
  )
}

////////////////////////////////////////////////////////////////
//                          HELPERS                           //
////////////////////////////////////////////////////////////////

/// Encode a root, key, value, and proof for consumption by the test suite
const encodeOutput = (
  root: string,
  key: string,
  val: string,
  proof: string[]
): string => {
  return ethers.utils.defaultAbiCoder.encode(
    ['bytes32', 'bytes', 'bytes', 'bytes[]'],
    [root, key, val, proof]
  )
}

/// Convert a Buffer to an abi-encodable hex string
const bufferToHex = (b: Buffer): string => {
  return `0x${b.toString('hex')}`
}

/// Generate a random number within a given range
const randRange = (max: number, min: number): number => {
  return Math.floor(Math.random() * (max - min) + min)
}

// Run program
genValidFuzzInput()
