import type { Account, Chain, Client, Transport } from 'viem'
import type { WalletActionsL2 as OpWalletActionsL2 } from 'viem/op-stack'
import { walletActionsL2 as opWalletActionsL2 } from 'viem/op-stack'

import type {
  RelayCrossDomainMessageParameters,
  RelayCrossDomainMessageReturnType,
  SendCrossDomainMessageParameters,
  SendCrossDomainMessageReturnType,
  SendETHContractReturnType,
  SendETHParameters,
  SendSuperchainERC20Parameters,
  SendSuperchainERC20ReturnType,
} from '@/actions/interop/index.js'
import {
  relayCrossDomainMessage,
  sendCrossDomainMessage,
  sendETH,
  sendSuperchainERC20,
} from '@/actions/interop/index.js'

export type WalletInteropActionsL2<
  TChain extends Chain | undefined = Chain | undefined,
  TAccount extends Account | undefined = Account | undefined,
> = {
  sendCrossDomainMessage: <chainOverride extends Chain | undefined = undefined>(
    parameters: SendCrossDomainMessageParameters<
      TChain,
      TAccount,
      chainOverride
    >,
  ) => Promise<SendCrossDomainMessageReturnType>
  relayCrossDomainMessage: <
    chainOverride extends Chain | undefined = undefined,
  >(
    parameters: RelayCrossDomainMessageParameters<
      TChain,
      TAccount,
      chainOverride
    >,
  ) => Promise<RelayCrossDomainMessageReturnType>
  sendSuperchainERC20: <chainOverride extends Chain | undefined = undefined>(
    parameters: SendSuperchainERC20Parameters<TChain, TAccount, chainOverride>,
  ) => Promise<SendSuperchainERC20ReturnType>
  sendETH: <chainOverride extends Chain | undefined = undefined>(
    parameters: SendETHParameters<TChain, TAccount, chainOverride>,
  ) => Promise<SendETHContractReturnType>
}

export type WalletActionsL2<
  chain extends Chain | undefined = Chain | undefined,
  account extends Account | undefined = Account | undefined,
> = OpWalletActionsL2<chain, account> & {
  /** interop actions scoped under this member */
  interop: WalletInteropActionsL2<chain, account>
}

export function walletActionsL2() {
  return <
    transport extends Transport,
    chain extends Chain | undefined = Chain | undefined,
    account extends Account | undefined = Account | undefined,
  >(
    client: Client<transport, chain, account>,
  ): WalletActionsL2<chain, account> => {
    return {
      ...opWalletActionsL2(),
      interop: {
        sendCrossDomainMessage: (args) => sendCrossDomainMessage(client, args),
        relayCrossDomainMessage: (args) =>
          relayCrossDomainMessage(client, args),
        sendSuperchainERC20: (args) => sendSuperchainERC20(client, args),
        sendETH: (args) => sendETH(client, args),
      },
    } as WalletActionsL2<chain, account>
  }
}
