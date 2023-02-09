import { str, bool, num, email, host, port, url, json } from 'envalid'
import { Provider } from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { ethers } from 'ethers'

import { LogLevel, logLevels } from '../common'

export type Validator<T> = (input: string | T) => T

export const validators = {
  str: (input: string) => {
    return str()._parse(input)
  },
  bool: (input: string | boolean) => {
    if (typeof input === 'boolean') {
      return input
    } else {
      return bool()._parse(input)
    }
  },
  num: (input: string | number) => {
    if (typeof input === 'number') {
      return input
    } else {
      return num()._parse(input)
    }
  },
  email: (input: string) => {
    return email()._parse(input)
  },
  host: (input: string) => {
    return host()._parse(input)
  },
  port: (input: string) => {
    return port()._parse(input)
  },
  url: (input: string) => {
    return url()._parse(input)
  },
  json: (input: string | object) => {
    if (typeof input === 'object') {
      return input
    } else {
      return json()._parse(input)
    }
  },
  Provider: (input: string | Provider) => {
    if (typeof input === 'string') {
      const parsed = url()._parse(input)
      return new ethers.providers.JsonRpcProvider(parsed)
    } else {
      return input
    }
  },
  JsonRpcProvider: (input: string | ethers.providers.JsonRpcProvider) => {
    if (typeof input === 'string') {
      const parsed = url()._parse(input)
      return new ethers.providers.JsonRpcProvider(parsed)
    } else {
      return input
    }
  },
  StaticJsonRpcProvider: (
    input: string | ethers.providers.StaticJsonRpcProvider
  ) => {
    if (typeof input === 'string') {
      const parsed = url()._parse(input)
      return new ethers.providers.StaticJsonRpcProvider(parsed)
    } else {
      return input
    }
  },
  Wallet: (input: string | Signer) => {
    if (typeof input === 'string') {
      if (!ethers.utils.isHexString(input)) {
        throw new Error(`expected wallet to be a hex string`)
      } else {
        return new ethers.Wallet(input)
      }
    } else {
      return input
    }
  },
  LogLevel: (input: LogLevel) => {
    if (!logLevels.includes(input as LogLevel)) {
      throw new Error(`expected log level to be one of ${logLevels.join(', ')}`)
    } else {
      return input as LogLevel
    }
  },
}
