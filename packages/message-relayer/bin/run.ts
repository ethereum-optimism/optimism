import { Wallet } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { MessageRelayerService } from '../src/service/message-relayer.service'

const main = async () => {
  const l2Provider = new JsonRpcProvider('')
  const l1Provider = new JsonRpcProvider('')

  const wallet = new Wallet('0x' + 'd0'.repeat(64), l1Provider)

  const service = new MessageRelayerService({
    l1RpcProvider: l1Provider,
    l2RpcProvider: l2Provider,
    stateCommitmentChainAddress: '0xcdf217480D0802DE475Bd531093c53033F2CDCfE',
    l1CrossDomainMessengerAddress: '0x1e3aa06079fDa5F395E663474ec5f7207A131bD2',
    l2CrossDomainMessengerAddress: '0x39405fcB4fcc295DcFD16b08bD6f0bAd9D34E438',
    l2ToL1MessagePasserAddress: '0x4200000000000000000000000000000000000000',
    pollingInterval: 5000,
    relaySigner: wallet,
    l2ChainStartingHeight: 63,
    blockOffset: 1,
  })

  await service.start()
}

main()
