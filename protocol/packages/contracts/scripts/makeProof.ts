// Script for generating an inclusion proof for use in testing.
// Meant for manual usage, ie.
// ts-node scripts/makeProof.ts 1 0x0000000000000000000000000000000000000002 0x0000000000000000000000000000000000000003 4 500000 0x06
import { generateMockWithdrawalProof } from '../helpers'

const args = process.argv.slice(2)

const [nonce, sender, target, value, gasLimit, data] = args

const main = async () => {
  const proof = await generateMockWithdrawalProof({
    nonce: +nonce,
    sender,
    target,
    value: +value,
    gasLimit: +gasLimit,
    data,
  })
  console.log(proof)
}
main()
