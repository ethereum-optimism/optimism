import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'
import { formatAbiItem, toEventSelector, toFunctionSelector } from 'viem/utils'

import * as abis from '../src/abis'

const fileName = 'abi-signatures.md'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

function pascalCase(str: string): string {
  // Handle strings that are already camelCase by looking for capital letters
  return str
    .split(/(?=[A-Z])/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join('')
}

let mdContent =
  '# Contract ABI Signatures\n\n' +
  '> [!NOTE]  \n' +
  '> This document is auto-generated. Do not edit it manually.\n\n' +
  'Function signatures, event topics, and error selectors ' +
  'for all contracts in the viem package. Each signature is accompanied by its 4-byte selector (for functions and errors) ' +
  'or 32-byte topic hash (for events).\n\n'

Object.entries(abis).forEach(([abiName, abi]) => {
  const contractName = pascalCase(abiName.replace(/Abi$/, ''))
  mdContent += `## ${contractName}\n\n`

  // Functions
  const functions = abi.filter((item) => item.type === 'function')
  if (functions.length > 0) {
    mdContent += '### Functions\n\n```solidity\n'
    functions.forEach((func) => {
      const signature = formatAbiItem(func)
      const selector = toFunctionSelector(signature)
      mdContent += `${signature}\n${selector}\n\n`
    })
    mdContent += '```\n\n'
  }

  // Events
  const events = abi.filter((item) => item.type === 'event')
  if (events.length > 0) {
    mdContent += '### Events\n\n```solidity\n'
    events.forEach((event) => {
      const signature = formatAbiItem(event)
      const topic = toEventSelector(signature)
      mdContent += `${signature}\n${topic}\n\n`
    })
    mdContent += '```\n\n'
  }

  // Errors
  const errors = abi.filter((item) => item.type === 'error')
  if (errors.length > 0) {
    mdContent += '### Errors\n\n```solidity\n'
    errors.forEach((error) => {
      const signature = formatAbiItem(error)
      const selector = toFunctionSelector(signature)
      mdContent += `${signature}\n${selector}\n\n`
    })
    mdContent += '```\n\n'
  }
})

// Write to file
fs.writeFileSync(path.join(__dirname, '../docs', fileName), mdContent)

// eslint-disable-next-line no-console
console.log(`Generated ${fileName} documentation`)
