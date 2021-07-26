"use strict";

const { dfn } = require('./JS');
const {
  encodeParameters,
  etherBalance,
  etherMantissa,
  etherUnsigned,
  mergeInterface
} = require('./Ethereum');
const BigNumber = require('bignumber.js');

async function makeComptroller(opts = {}) {
  const {
    root = saddle.account,
    kind = 'unitroller'
  } = opts || {};

  if (kind == 'bool') {
    return await deploy('BoolComptroller');
  }

  if (kind == 'false-marker') {
    return await deploy('FalseMarkerMethodComptroller');
  }

  if (kind == 'v1-no-proxy') {
    const comptroller = await deploy('ComptrollerHarness');
    const priceOracle = opts.priceOracle || await makePriceOracle(opts.priceOracleOpts);
    const closeFactor = etherMantissa(dfn(opts.closeFactor, .051));

    await send(comptroller, '_setCloseFactor', [closeFactor]);
    await send(comptroller, '_setPriceOracle', [priceOracle._address]);

    return Object.assign(comptroller, { priceOracle });
  }

  if (kind == 'unitroller-g2') {
    const unitroller = opts.unitroller || await deploy('Unitroller');
    const comptroller = await deploy('ComptrollerScenarioG2');
    const priceOracle = opts.priceOracle || await makePriceOracle(opts.priceOracleOpts);
    const closeFactor = etherMantissa(dfn(opts.closeFactor, .051));
    const maxAssets = etherUnsigned(dfn(opts.maxAssets, 10));
    const liquidationIncentive = etherMantissa(1);

    await send(unitroller, '_setPendingImplementation', [comptroller._address]);
    await send(comptroller, '_become', [unitroller._address]);
    mergeInterface(unitroller, comptroller);
    await send(unitroller, '_setLiquidationIncentive', [liquidationIncentive]);
    await send(unitroller, '_setCloseFactor', [closeFactor]);
    await send(unitroller, '_setMaxAssets', [maxAssets]);
    await send(unitroller, '_setPriceOracle', [priceOracle._address]);

    return Object.assign(unitroller, { priceOracle });
  }

  if (kind == 'unitroller-g3') {
    const unitroller = opts.unitroller || await deploy('Unitroller');
    const comptroller = await deploy('ComptrollerScenarioG3');
    const priceOracle = opts.priceOracle || await makePriceOracle(opts.priceOracleOpts);
    const closeFactor = etherMantissa(dfn(opts.closeFactor, .051));
    const maxAssets = etherUnsigned(dfn(opts.maxAssets, 10));
    const liquidationIncentive = etherMantissa(1);
    const compRate = etherUnsigned(dfn(opts.compRate, 1e18));
    const compMarkets = opts.compMarkets || [];
    const otherMarkets = opts.otherMarkets || [];

    await send(unitroller, '_setPendingImplementation', [comptroller._address]);
    await send(comptroller, '_become', [unitroller._address, compRate, compMarkets, otherMarkets]);
    mergeInterface(unitroller, comptroller);
    await send(unitroller, '_setLiquidationIncentive', [liquidationIncentive]);
    await send(unitroller, '_setCloseFactor', [closeFactor]);
    await send(unitroller, '_setMaxAssets', [maxAssets]);
    await send(unitroller, '_setPriceOracle', [priceOracle._address]);

    return Object.assign(unitroller, { priceOracle });
  }

  if (kind == 'unitroller-g6') {
    const unitroller = opts.unitroller || await deploy('Unitroller');
    const comptroller = await deploy('ComptrollerScenarioG6');
    const priceOracle = opts.priceOracle || await makePriceOracle(opts.priceOracleOpts);
    const closeFactor = etherMantissa(dfn(opts.closeFactor, .051));
    const liquidationIncentive = etherMantissa(1);
    const comp = opts.comp || await deploy('Comp', [opts.compOwner || root]);
    const compRate = etherUnsigned(dfn(opts.compRate, 1e18));

    await send(unitroller, '_setPendingImplementation', [comptroller._address]);
    await send(comptroller, '_become', [unitroller._address]);
    mergeInterface(unitroller, comptroller);
    await send(unitroller, '_setLiquidationIncentive', [liquidationIncentive]);
    await send(unitroller, '_setCloseFactor', [closeFactor]);
    await send(unitroller, '_setPriceOracle', [priceOracle._address]);
    await send(unitroller, '_setCompRate', [compRate]);
    await send(unitroller, 'setCompAddress', [comp._address]); // harness only

    return Object.assign(unitroller, { priceOracle, comp });
  }

  if (kind == 'unitroller') {
    const unitroller = opts.unitroller || await deploy('Unitroller');
    const comptroller = await deploy('ComptrollerHarness');
    const priceOracle = opts.priceOracle || await makePriceOracle(opts.priceOracleOpts);
    const closeFactor = etherMantissa(dfn(opts.closeFactor, .051));
    const liquidationIncentive = etherMantissa(1);
    const comp = opts.comp || await deploy('Comp', [opts.compOwner || root]);
    const compRate = etherUnsigned(dfn(opts.compRate, 1e18));

    await send(unitroller, '_setPendingImplementation', [comptroller._address]);
    await send(comptroller, '_become', [unitroller._address]);
    mergeInterface(unitroller, comptroller);
    await send(unitroller, '_setLiquidationIncentive', [liquidationIncentive]);
    await send(unitroller, '_setCloseFactor', [closeFactor]);
    await send(unitroller, '_setPriceOracle', [priceOracle._address]);
    await send(unitroller, 'setCompAddress', [comp._address]); // harness only
    await send(unitroller, 'harnessSetCompRate', [compRate]);

    return Object.assign(unitroller, { priceOracle, comp });
  }
}

async function makeCToken(opts = {}) {
  const {
    root = saddle.account,
    kind = 'cerc20'
  } = opts || {};

  const comptroller = opts.comptroller || await makeComptroller(opts.comptrollerOpts);
  const interestRateModel = opts.interestRateModel || await makeInterestRateModel(opts.interestRateModelOpts);
  const exchangeRate = etherMantissa(dfn(opts.exchangeRate, 1));
  const decimals = etherUnsigned(dfn(opts.decimals, 8));
  const symbol = opts.symbol || (kind === 'cether' ? 'cETH' : 'cOMG');
  const name = opts.name || `CToken ${symbol}`;
  const admin = opts.admin || root;

  let cToken, underlying;
  let cDelegator, cDelegatee, cDaiMaker;

  switch (kind) {
    case 'cether':
      cToken = await deploy('CEtherHarness',
        [
          comptroller._address,
          interestRateModel._address,
          exchangeRate,
          name,
          symbol,
          decimals,
          admin
        ])
      break;

    case 'cdai':
      cDaiMaker  = await deploy('CDaiDelegateMakerHarness');
      underlying = cDaiMaker;
      cDelegatee = await deploy('CDaiDelegateHarness');
      cDelegator = await deploy('CErc20Delegator',
        [
          underlying._address,
          comptroller._address,
          interestRateModel._address,
          exchangeRate,
          name,
          symbol,
          decimals,
          admin,
          cDelegatee._address,
          encodeParameters(['address', 'address'], [cDaiMaker._address, cDaiMaker._address])
        ]
      );
      cToken = await saddle.getContractAt('CDaiDelegateHarness', cDelegator._address);
      break;

    case 'ccomp':
      underlying = await deploy('Comp', [opts.compHolder || root]);
      cDelegatee = await deploy('CCompLikeDelegate');
      cDelegator = await deploy('CErc20Delegator',
        [
          underlying._address,
          comptroller._address,
          interestRateModel._address,
          exchangeRate,
          name,
          symbol,
          decimals,
          admin,
          cDelegatee._address,
          "0x0"
        ]
      );
      cToken = await saddle.getContractAt('CCompLikeDelegate', cDelegator._address);
      break;

    case 'cerc20':
    default:
      underlying = opts.underlying || await makeToken(opts.underlyingOpts);
      cDelegatee = await deploy('CErc20DelegateHarness');
      cDelegator = await deploy('CErc20Delegator',
        [
          underlying._address,
          comptroller._address,
          interestRateModel._address,
          exchangeRate,
          name,
          symbol,
          decimals,
          admin,
          cDelegatee._address,
          "0x0"
        ]
      );
      cToken = await saddle.getContractAt('CErc20DelegateHarness', cDelegator._address);
      break;
  }

  if (opts.supportMarket) {
    await send(comptroller, '_supportMarket', [cToken._address]);
  }

  if (opts.addCompMarket) {
    await send(comptroller, '_addCompMarket', [cToken._address]);
  }

  if (opts.underlyingPrice) {
    const price = etherMantissa(opts.underlyingPrice);
    await send(comptroller.priceOracle, 'setUnderlyingPrice', [cToken._address, price]);
  }

  if (opts.collateralFactor) {
    const factor = etherMantissa(opts.collateralFactor);
    expect(await send(comptroller, '_setCollateralFactor', [cToken._address, factor])).toSucceed();
  }

  return Object.assign(cToken, { name, symbol, underlying, comptroller, interestRateModel });
}

async function makeInterestRateModel(opts = {}) {
  const {
    root = saddle.account,
    kind = 'harnessed'
  } = opts || {};

  if (kind == 'harnessed') {
    const borrowRate = etherMantissa(dfn(opts.borrowRate, 0));
    return await deploy('InterestRateModelHarness', [borrowRate]);
  }

  if (kind == 'false-marker') {
    const borrowRate = etherMantissa(dfn(opts.borrowRate, 0));
    return await deploy('FalseMarkerMethodInterestRateModel', [borrowRate]);
  }

  if (kind == 'white-paper') {
    const baseRate = etherMantissa(dfn(opts.baseRate, 0));
    const multiplier = etherMantissa(dfn(opts.multiplier, 1e-18));
    return await deploy('WhitePaperInterestRateModel', [baseRate, multiplier]);
  }

  if (kind == 'jump-rate') {
    const baseRate = etherMantissa(dfn(opts.baseRate, 0));
    const multiplier = etherMantissa(dfn(opts.multiplier, 1e-18));
    const jump = etherMantissa(dfn(opts.jump, 0));
    const kink = etherMantissa(dfn(opts.kink, 0));
    return await deploy('JumpRateModel', [baseRate, multiplier, jump, kink]);
  }
}

async function makePriceOracle(opts = {}) {
  const {
    root = saddle.account,
    kind = 'simple'
  } = opts || {};

  if (kind == 'simple') {
    return await deploy('SimplePriceOracle');
  }
}

async function makeToken(opts = {}) {
  const {
    root = saddle.account,
    kind = 'erc20'
  } = opts || {};

  if (kind == 'erc20') {
    const quantity = etherUnsigned(dfn(opts.quantity, 1e25));
    const decimals = etherUnsigned(dfn(opts.decimals, 18));
    const symbol = opts.symbol || 'OMG';
    const name = opts.name || `Erc20 ${symbol}`;
    return await deploy('ERC20Harness', [quantity, name, decimals, symbol]);
  }
}

async function balanceOf(token, account) {
  return etherUnsigned(await call(token, 'balanceOf', [account]));
}

async function totalSupply(token) {
  return etherUnsigned(await call(token, 'totalSupply'));
}

async function borrowSnapshot(cToken, account) {
  const { principal, interestIndex } = await call(cToken, 'harnessAccountBorrows', [account]);
  return { principal: etherUnsigned(principal), interestIndex: etherUnsigned(interestIndex) };
}

async function totalBorrows(cToken) {
  return etherUnsigned(await call(cToken, 'totalBorrows'));
}

async function totalReserves(cToken) {
  return etherUnsigned(await call(cToken, 'totalReserves'));
}

async function enterMarkets(cTokens, from) {
  return await send(cTokens[0].comptroller, 'enterMarkets', [cTokens.map(c => c._address)], { from });
}

async function fastForward(cToken, blocks = 5) {
  return await send(cToken, 'harnessFastForward', [blocks]);
}

async function setBalance(cToken, account, balance) {
  return await send(cToken, 'harnessSetBalance', [account, balance]);
}

async function setEtherBalance(cEther, balance) {
  const current = await etherBalance(cEther._address);
  const root = saddle.account;
  expect(await send(cEther, 'harnessDoTransferOut', [root, current])).toSucceed();
  expect(await send(cEther, 'harnessDoTransferIn', [root, balance], { value: balance })).toSucceed();
}

async function getBalances(cTokens, accounts) {
  const balances = {};
  for (let cToken of cTokens) {
    const cBalances = balances[cToken._address] = {};
    for (let account of accounts) {
      cBalances[account] = {
        eth: await etherBalance(account),
        cash: cToken.underlying && await balanceOf(cToken.underlying, account),
        tokens: await balanceOf(cToken, account),
        borrows: (await borrowSnapshot(cToken, account)).principal
      };
    }
    cBalances[cToken._address] = {
      eth: await etherBalance(cToken._address),
      cash: cToken.underlying && await balanceOf(cToken.underlying, cToken._address),
      tokens: await totalSupply(cToken),
      borrows: await totalBorrows(cToken),
      reserves: await totalReserves(cToken)
    };
  }
  return balances;
}

async function adjustBalances(balances, deltas) {
  for (let delta of deltas) {
    let cToken, account, key, diff;
    if (delta.length == 4) {
      ([cToken, account, key, diff] = delta);
    } else {
      ([cToken, key, diff] = delta);
      account = cToken._address;
    }

    balances[cToken._address][account][key] = new BigNumber(balances[cToken._address][account][key]).plus(diff);
  }
  return balances;
}


async function preApprove(cToken, from, amount, opts = {}) {
  if (dfn(opts.faucet, true)) {
    expect(await send(cToken.underlying, 'harnessSetBalance', [from, amount], { from })).toSucceed();
  }

  return send(cToken.underlying, 'approve', [cToken._address, amount], { from });
}

async function quickMint(cToken, minter, mintAmount, opts = {}) {
  // make sure to accrue interest
  await fastForward(cToken, 1);

  if (dfn(opts.approve, true)) {
    expect(await preApprove(cToken, minter, mintAmount, opts)).toSucceed();
  }
  if (dfn(opts.exchangeRate)) {
    expect(await send(cToken, 'harnessSetExchangeRate', [etherMantissa(opts.exchangeRate)])).toSucceed();
  }
  return send(cToken, 'mint', [mintAmount], { from: minter });
}


async function preSupply(cToken, account, tokens, opts = {}) {
  if (dfn(opts.total, true)) {
    expect(await send(cToken, 'harnessSetTotalSupply', [tokens])).toSucceed();
  }
  return send(cToken, 'harnessSetBalance', [account, tokens]);
}

async function quickRedeem(cToken, redeemer, redeemTokens, opts = {}) {
  await fastForward(cToken, 1);

  if (dfn(opts.supply, true)) {
    expect(await preSupply(cToken, redeemer, redeemTokens, opts)).toSucceed();
  }
  if (dfn(opts.exchangeRate)) {
    expect(await send(cToken, 'harnessSetExchangeRate', [etherMantissa(opts.exchangeRate)])).toSucceed();
  }
  return send(cToken, 'redeem', [redeemTokens], { from: redeemer });
}

async function quickRedeemUnderlying(cToken, redeemer, redeemAmount, opts = {}) {
  await fastForward(cToken, 1);

  if (dfn(opts.exchangeRate)) {
    expect(await send(cToken, 'harnessSetExchangeRate', [etherMantissa(opts.exchangeRate)])).toSucceed();
  }
  return send(cToken, 'redeemUnderlying', [redeemAmount], { from: redeemer });
}

async function setOraclePrice(cToken, price) {
  return send(cToken.comptroller.priceOracle, 'setUnderlyingPrice', [cToken._address, etherMantissa(price)]);
}

async function setBorrowRate(cToken, rate) {
  return send(cToken.interestRateModel, 'setBorrowRate', [etherMantissa(rate)]);
}

async function getBorrowRate(interestRateModel, cash, borrows, reserves) {
  return call(interestRateModel, 'getBorrowRate', [cash, borrows, reserves].map(etherUnsigned));
}

async function getSupplyRate(interestRateModel, cash, borrows, reserves, reserveFactor) {
  return call(interestRateModel, 'getSupplyRate', [cash, borrows, reserves, reserveFactor].map(etherUnsigned));
}

async function pretendBorrow(cToken, borrower, accountIndex, marketIndex, principalRaw, blockNumber = 2e7) {
  await send(cToken, 'harnessSetTotalBorrows', [etherUnsigned(principalRaw)]);
  await send(cToken, 'harnessSetAccountBorrows', [borrower, etherUnsigned(principalRaw), etherMantissa(accountIndex)]);
  await send(cToken, 'harnessSetBorrowIndex', [etherMantissa(marketIndex)]);
  await send(cToken, 'harnessSetAccrualBlockNumber', [etherUnsigned(blockNumber)]);
  await send(cToken, 'harnessSetBlockNumber', [etherUnsigned(blockNumber)]);
}

module.exports = {
  makeComptroller,
  makeCToken,
  makeInterestRateModel,
  makePriceOracle,
  makeToken,

  balanceOf,
  totalSupply,
  borrowSnapshot,
  totalBorrows,
  totalReserves,
  enterMarkets,
  fastForward,
  setBalance,
  setEtherBalance,
  getBalances,
  adjustBalances,

  preApprove,
  quickMint,

  preSupply,
  quickRedeem,
  quickRedeemUnderlying,

  setOraclePrice,
  setBorrowRate,
  getBorrowRate,
  getSupplyRate,
  pretendBorrow
};
