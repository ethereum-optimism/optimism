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
import { Signer } from '@ethersproject/abstract-signer'
import { ethers } from 'ethers'

const ethersJsonRpcProvider = makeValidator<ethers.providers.JsonRpcProvider>(
  (input) => {
    const parsed = url()._parse(input)
    return new ethers.providers.JsonRpcProvider(parsed)
  }
)

const ethersStaticJsonRpcProvider =
  makeValidator<ethers.providers.StaticJsonRpcProvider>((input) => {
    const parsed = url()._parse(input)
    return new ethers.providers.StaticJsonRpcProvider(parsed)
  })

const ethersJsonRpcBatchProvider =
  makeValidator<ethers.providers.JsonRpcBatchProvider>((input) => {
    const parsed = url()._parse(input)
    return new ethers.providers.JsonRpcBatchProvider(parsed)
  })

const ethersWebSocketProvider =
  makeValidator<ethers.providers.WebSocketProvider>((input) => {
    const parsed = url()._parse(input)
    return new ethers.providers.WebSocketProvider(parsed)
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
  provider: ethersJsonRpcProvider,
  ethersJsonRpcProvider,
  ethersJsonRpcBatchProvider,
  ethersStaticJsonRpcProvider,
  ethersWebSocketProvider,
}
