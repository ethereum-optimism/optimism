import { DeployConfig } from '../src/deploy-config'

const config: DeployConfig = {
  network: 'goerli',
  l1BlockTimeSeconds: 15,
  l2BlockGasLimit: 15_000_000,
  l2ChainId: 420,
  ctcL2GasDiscountDivisor: 32,
  ctcEnqueueGasCost: 60_000,
  sccFaultProofWindowSeconds: 604800,
  sccSequencerPublishWindowSeconds: 12592000,
  ovmSequencerAddress: '0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244',
  ovmProposerAddress: '0x9A2F243c605e6908D96b18e21Fb82Bf288B19EF3',
  ovmBlockSignerAddress: '0x27770a9694e4B4b1E130Ab91Bc327C36855f612E',
  ovmFeeWalletAddress: '0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244',
  ovmAddressManagerOwner: '0x32b70c156302d28A9119445d2bbb9ab1cBD01671',
  ovmGasPriceOracleOwner: '0x84f70449f90300997840eCb0918873745Ede7aE6',
}

export default config
