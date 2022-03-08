import {
  str,
  bool,
  num,
  email,
  host,
  port,
  url,
  json,
  makeValidator,
} from 'envalid'
import { Provider } from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { ethers } from 'ethers'

const provider = makeValidator<Provider>((input) => {
  const parsed = url()._parse(input)
  return new ethers.providers.JsonRpcProvider(parsed)
})

const wallet = makeValidator<Signer>((input) => {
  if (!ethers.utils.isHexString(input)) {
    throw new Error(`expected wallet to be a hex string`)
  } else {
    return new ethers.Wallet(input)
  }
})

export const validators = {
  str,
  bool,
  num,
  email,
  host,
  port,
  url,
  json,
  wallet,
  provider,
}
