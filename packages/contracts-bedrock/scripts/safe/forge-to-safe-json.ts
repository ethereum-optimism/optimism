import fs from 'fs'
import crypto from 'crypto'

interface ForgeTransaction {
  contractAddress: string
  transaction: {
    to: string
    value: string
    input: string
  }
}

interface ForgeTransactionFile {
  transactions: ForgeTransaction[]
  timestamp: number
  chain: number
}

interface SafeTransaction {
  to: string
  value: string
  data: string
  contractMethod: null
  contractInputsValues: null
}

interface SafeTransactionFile {
  version: string
  chainId: string
  createdAt: number
  meta: {
    name: string
    description: string
    txBuilderVersion: string
    createdFromSafeAddress: string
    createdFromOwnerAddress: string
    checksum: string
  }
  transactions: SafeTransaction[]
}

/**
 * Calculate the checksum for a safe transaction file.
 * @param data The safe transaction file.
 * @returns The checksum.
 */
const checksum = (data: SafeTransactionFile): string => {
  const hash = crypto.createHash('sha256')
  hash.update(JSON.stringify(data.transactions))
  return `0x${hash.digest('hex')}`
}

/**
 * Transform a forge transaction file into a safe transaction file.
 * @param forge Forge transaction file.
 * @param address Safe address.
 * @returns Safe transaction file.
 */
const transform = (
  forge: ForgeTransactionFile,
  address: string
): SafeTransactionFile => {
  const transactions = forge.transactions.map((tx) => ({
    to: tx.contractAddress,
    value: '0',
    data: tx.transaction.input,
    contractMethod: null,
    contractInputsValues: null,
  }))

  const safe: SafeTransactionFile = {
    version: '1.0',
    chainId: forge.chain.toString(),
    createdAt: forge.timestamp,
    meta: {
      name: 'Transactions Batch',
      description: '',
      txBuilderVersion: '1.16.5',
      createdFromSafeAddress: address,
      createdFromOwnerAddress: '',
      checksum: '',
    },
    transactions,
  }

  safe.meta.checksum = checksum(safe)

  return safe
}

/**
 * Get a required argument from the command line.
 * @param name The argument name.
 * @returns The argument value.
 */
const reqarg = (name: string) => {
  const value = process.argv.find((arg) => arg.startsWith(`--${name}=`))
  if (!value) {
    console.error(`Please provide --${name} argument`)
    process.exit(1)
  }
  return value.split('=')[1]
}

/**
 * Main function.
 */
const main = () => {
  const input = reqarg('input')
  const output = reqarg('output')
  const address = reqarg('safe')

  // Load the original forge transaction file.
  const forge: ForgeTransactionFile = JSON.parse(fs.readFileSync(input, 'utf8'))

  // Transform the forge transaction file into a safe transaction file.
  const safe = transform(forge, address)

  // Write the safe transaction file.
  fs.writeFileSync(output, JSON.stringify(safe, null, 2), 'utf8')
}

// Run the main function.
main()
