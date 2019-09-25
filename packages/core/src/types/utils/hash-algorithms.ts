export enum HashAlgorithm {
  MD5 = 'MD5',
  KECCAK256 = 'KECCAK256',
}

export type HashFunction = (string) => string
