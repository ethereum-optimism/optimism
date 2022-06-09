// Script for generating an inclusion proof for use in testing
// Intended for use with forge test --ffi, accepts abi encoded input and returns
// only the storageTrieWitness.
import { generateMockWithdrawalProof } from '../helpers'

let args = process.argv.slice(2)[0]

args = args
  .replace('0x', '')
  .split('')
  .filter((char) => '0123456789abcdef'.includes(char))
  .join('')

const main = async () => {
  const proof = await generateMockWithdrawalProof('0x' + args)
  console.log(proof.storageTrieWitness.slice(2))
}
main()
