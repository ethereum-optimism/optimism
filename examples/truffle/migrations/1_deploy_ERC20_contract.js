const ERC20 = artifacts.require('ERC20')

module.exports = function (deployer, accounts) {
  const tokenName = 'My Optimistic Coin'
  const tokenSymbol = 'OPT'
  const tokenDecimals = 1

  // deployment steps
  deployer.deploy(
    ERC20,
    10000,
    tokenName,
    tokenDecimals,
    tokenSymbol,
    { gasPrice: 0 }
  )
}
