const config = {
  numDeployConfirmations: 1,
  l1BlockTimeSeconds: 15,
  l2BlockGasLimit: 15_000_000,
  l2ChainId: 420,
  ctcL2GasDiscountDivisor: 32,
  ctcEnqueueGasCost: 60_000,
  sccFaultProofWindowSeconds: 10,
  sccSequencerPublishWindowSeconds: 12592000,
  ovmSequencerAddress: '0x7431310e026B69BFC676C0013E12A1A11411EEc9',
  ovmProposerAddress: '0x02b1786A85Ec3f71fBbBa46507780dB7cF9014f6',
  ovmBlockSignerAddress: '0x27770a9694e4B4b1E130Ab91Bc327C36855f612E',
  ovmFeeWalletAddress: '0xfd1D2e729aE8eEe2E146c033bf4400fE75284301',
  ovmAddressManagerOwner: '0xf80267194936da1E98dB10bcE06F3147D580a62e',
  ovmGasPriceOracleOwner: '0xa693B8f8207FF043F6bbC2E2120bbE4C2251Efe9',
}

export default config
