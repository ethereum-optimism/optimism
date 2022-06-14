const config = {
  numDeployConfirmations: 1,
  gasPrice: 5_000_000_000,
  l1BlockTimeSeconds: 15,
  l2BlockGasLimit: 15_000_000,
  l2ChainId: 69,
  ctcL2GasDiscountDivisor: 32,
  ctcEnqueueGasCost: 60_000,
  sccFaultProofWindowSeconds: 10,
  sccSequencerPublishWindowSeconds: 12592000,
  ovmSequencerAddress: '0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244',
  ovmProposerAddress: '0x9A2F243c605e6908D96b18e21Fb82Bf288B19EF3',
  ovmBlockSignerAddress: '0x00000398232E2064F896018496b4b44b3D62751F',
  ovmFeeWalletAddress: '0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244',
  ovmAddressManagerOwner: '0x18394B52d3Cb931dfA76F63251919D051953413d',
  ovmGasPriceOracleOwner: '0x84f70449f90300997840eCb0918873745Ede7aE6',
}

export default config
