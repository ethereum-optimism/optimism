import { ethers } from 'ethers'

const errorABI = new ethers.utils.Interface([
  {
    type: 'function',
    inputs: [
      {
        type: 'string',
      },
    ],
    name: 'Error',
    stateMutability: 'pure',
  },
])

export const decodeSolidityError = (err: string): string => {
  return errorABI.decodeFunctionData('Error', err)[0]
}

export const encodeSolidityError = (message: string): string => {
  return errorABI.encodeFunctionData('Error', [message])
}
