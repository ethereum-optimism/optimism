const config = {
  numDeployConfirmations: 1,
  l1BlockTimeSeconds: 150,
  l2BlockGasLimit: 15_000_000,
  l2ChainId: 57000,
  ctcL2GasDiscountDivisor: 32,
  ctcEnqueueGasCost: 60_000,
  sccFaultProofWindowSeconds: 10,
  sccSequencerPublishWindowSeconds: 12592000,
  ovmSequencerAddress: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
  ovmProposerAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  ovmBlockSignerAddress: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
  ovmFeeWalletAddress: '0x749058367c48a10c728073dcc4613560d69e730d',
  ovmAddressManagerOwner: '0x48ab1cE92e1ea9713AdDeA668E146f575D60954e',
  ovmGasPriceOracleOwner: '0x48ab1cE92e1ea9713AdDeA668E146f575D60954e',
}

export default config
