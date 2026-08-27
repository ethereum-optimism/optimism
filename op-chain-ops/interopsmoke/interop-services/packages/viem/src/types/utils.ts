import type { Account, Address, Chain, ChainContract } from 'viem'

export type Prettify<T> = {
  [K in keyof T]: T[K]
} & {}

export type UnionEvaluate<type> = type extends object ? Prettify<type> : type

export type IsUndefined<T> = [undefined] extends [T] ? true : false
export type ErrorType<name extends string = 'Error'> = Error & { name: name }

export type GetAccountParameter<
  account extends Account | undefined = Account | undefined,
  accountOverride extends Account | Address | undefined = Account | Address,
  required extends boolean = true,
> = IsUndefined<account> extends true
  ? required extends true
    ? { account: accountOverride | Account | Address }
    : { account?: accountOverride | Account | Address | undefined }
  : { account?: accountOverride | Account | Address | undefined }

export type GetContractAddressParameter<
  chain extends Chain | undefined,
  contractName extends string,
> =
  | (chain extends Chain
      ? Prettify<
          {
            targetChain: Prettify<TargetChain<chain, contractName>>
          } & {
            [_ in `${contractName}Address`]?: undefined
          }
        >
      : never)
  | Prettify<
      {
        targetChain?: undefined
      } & {
        [_ in `${contractName}Address`]: Address
      }
    >

export type TargetChain<
  chain extends Chain = Chain,
  contractName extends string = string,
> = {
  /** Required Properties of `Chain` */
  id: chain['id']
  name: chain['name']
  nativeCurrency: chain['nativeCurrency']
  rpcUrls: chain['rpcUrls']

  /** Enforce the specific contract */
  contracts: {
    [_ in contractName]: { [_ in chain['id']]: ChainContract }
  }
}
