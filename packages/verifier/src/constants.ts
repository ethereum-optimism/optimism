export const PREDICATE_ABI = [
  {
    constant: true,
    inputs: [
      { name: '_oldState', type: 'bytes' },
      { name: '_newState', type: 'bytes' },
      { name: '_witness', type: 'bytes' },
    ],
    name: 'validStateTransition',
    outputs: [{ name: '', type: 'bool' }],
    payable: false,
    stateMutability: 'view',
    type: 'function',
  },
]

export const ACCOUNT = {
  address: '0xa75abbCA2a01847F54FC4E5F4D73F8F6bA9E13F5',
  privateKey: Buffer.from(
    '5733f2b332c443b839196cd14635342ffa046c402007e82bf16fb60d56a53aa6',
    'hex'
  ),
}
