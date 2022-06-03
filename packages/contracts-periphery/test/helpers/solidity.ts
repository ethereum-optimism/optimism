import { ethers } from 'ethers'

export const decodeSolidityRevert = (revert: string) => {
  const iface = new ethers.utils.Interface([
    {
      inputs: [
        {
          internalType: 'string',
          name: 'message',
          type: 'string',
        },
      ],
      name: 'Error',
      outputs: [],
      stateMutability: 'nonpayable',
      type: 'function',
    },
  ])

  return iface.decodeFunctionData('Error', revert)[0]
}
