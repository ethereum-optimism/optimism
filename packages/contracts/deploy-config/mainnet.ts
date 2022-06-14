const config = {
  numDeployConfirmations: 4,
  gasPrice: 150_000_000_000,
  l1BlockTimeSeconds: 15,
  l2BlockGasLimit: 15_000_000,
  l2ChainId: 10,
  ctcL2GasDiscountDivisor: 32,
  ctcEnqueueGasCost: 60_000,
  sccFaultProofWindowSeconds: 604800,
  sccSequencerPublishWindowSeconds: 12592000,
  ovmSequencerAddress: '0x6887246668a3b87F54DeB3b94Ba47a6f63F32985',
  ovmProposerAddress: '0x473300df21D047806A082244b417f96b32f13A33',
  ovmBlockSignerAddress: '0x00000398232E2064F896018496b4b44b3D62751F',
  ovmFeeWalletAddress: '0x391716d440c151c42cdf1c95c1d83a5427bca52c',
  ovmAddressManagerOwner: '0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A',
  ovmGasPriceOracleOwner: '0x7107142636C85c549690b1Aca12Bdb8052d26Ae6',
  ovmWhitelistOwner: '0x648E3e8101BFaB7bf5997Bd007Fb473786019159',
}

export default config
