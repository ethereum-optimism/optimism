const {
  etherUnsigned,
  etherMantissa,
  etherExp,
} = require('./Utils/Ethereum');

const {
  makeComptroller,
  makeCToken,
  preApprove,
  preSupply,
  quickRedeem,
} = require('./Utils/Compound');

async function compBalance(comptroller, user) {
  return etherUnsigned(await call(comptroller.comp, 'balanceOf', [user]))
}

async function compAccrued(comptroller, user) {
  return etherUnsigned(await call(comptroller, 'compAccrued', [user]));
}

async function fastForwardPatch(patch, comptroller, blocks) {
  if (patch == 'unitroller') {
    return await send(comptroller, 'harnessFastForward', [blocks]);
  } else {
    return await send(comptroller, 'fastForward', [blocks]);
  }
}

const fs = require('fs');
const util = require('util');
const diffStringsUnified = require('jest-diff').default;


async function preRedeem(
  cToken,
  redeemer,
  redeemTokens,
  redeemAmount,
  exchangeRate
) {
  await preSupply(cToken, redeemer, redeemTokens);
  await send(cToken.underlying, 'harnessSetBalance', [
    cToken._address,
    redeemAmount
  ]);
}

const sortOpcodes = (opcodesMap) => {
  return Object.values(opcodesMap)
    .map(elem => [elem.fee, elem.name])
    .sort((a, b) => b[0] - a[0]);
};

const getGasCostFile = name => {
  try {
    const jsonString = fs.readFileSync(name);
    return JSON.parse(jsonString);
  } catch (err) {
    console.log(err);
    return {};
  }
};

const recordGasCost = (totalFee, key, filename, opcodes = {}) => {
  let fileObj = getGasCostFile(filename);
  const newCost = {fee: totalFee, opcodes: opcodes};
  console.log(diffStringsUnified(fileObj[key], newCost));
  fileObj[key] = newCost;
  fs.writeFileSync(filename, JSON.stringify(fileObj, null, ' '), 'utf-8');
};

async function mint(cToken, minter, mintAmount, exchangeRate) {
  expect(await preApprove(cToken, minter, mintAmount, {})).toSucceed();
  return send(cToken, 'mint', [mintAmount], { from: minter });
}

async function claimComp(comptroller, holder) {
  return send(comptroller, 'claimComp', [holder], { from: holder });
}

/// GAS PROFILER: saves a digest of the gas prices of common CToken operations
/// transiently fails, not sure why

describe('Gas report', () => {
  let root, minter, redeemer, accounts, cToken;
  const exchangeRate = 50e3;
  const preMintAmount = etherUnsigned(30e4);
  const mintAmount = etherUnsigned(10e4);
  const mintTokens = mintAmount.div(exchangeRate);
  const redeemTokens = etherUnsigned(10e3);
  const redeemAmount = redeemTokens.multipliedBy(exchangeRate);
  const filename = './gasCosts.json';

  describe('CToken', () => {
    beforeEach(async () => {
      [root, minter, redeemer, ...accounts] = saddle.accounts;
      cToken = await makeCToken({
        comptrollerOpts: { kind: 'bool'}, 
        interestRateModelOpts: { kind: 'white-paper'},
        exchangeRate
      });
    });

    it('first mint', async () => {
      await send(cToken, 'harnessSetAccrualBlockNumber', [40]);
      await send(cToken, 'harnessSetBlockNumber', [41]);

      const trxReceipt = await mint(cToken, minter, mintAmount, exchangeRate);
      recordGasCost(trxReceipt.gasUsed, 'first mint', filename);
    });

    it('second mint', async () => {
      await mint(cToken, minter, mintAmount, exchangeRate);

      await send(cToken, 'harnessSetAccrualBlockNumber', [40]);
      await send(cToken, 'harnessSetBlockNumber', [41]);

      const mint2Receipt = await mint(cToken, minter, mintAmount, exchangeRate);
      expect(Object.keys(mint2Receipt.events)).toEqual(['AccrueInterest', 'Transfer', 'Mint']);

      console.log(mint2Receipt.gasUsed);
      const opcodeCount = {};

      await saddle.trace(mint2Receipt, {
        execLog: log => {
          if (log.lastLog != undefined) {
            const key = `${log.op} @ ${log.gasCost}`;
            opcodeCount[key] = (opcodeCount[key] || 0) + 1;
          }
        }
      });

      recordGasCost(mint2Receipt.gasUsed, 'second mint', filename, opcodeCount);
    });

    it('second mint, no interest accrued', async () => {
      await mint(cToken, minter, mintAmount, exchangeRate);

      await send(cToken, 'harnessSetAccrualBlockNumber', [40]);
      await send(cToken, 'harnessSetBlockNumber', [40]);

      const mint2Receipt = await mint(cToken, minter, mintAmount, exchangeRate);
      expect(Object.keys(mint2Receipt.events)).toEqual(['Transfer', 'Mint']);
      recordGasCost(mint2Receipt.gasUsed, 'second mint, no interest accrued', filename);

      // console.log("NO ACCRUED");
      // const opcodeCount = {};
      // await saddle.trace(mint2Receipt, {
      //   execLog: log => {
      //     opcodeCount[log.op] = (opcodeCount[log.op] || 0) + 1;
      //   }
      // });
      // console.log(getOpcodeDigest(opcodeCount));
    });

    it('redeem', async () => {
      await preRedeem(cToken, redeemer, redeemTokens, redeemAmount, exchangeRate);
      const trxReceipt = await quickRedeem(cToken, redeemer, redeemTokens);
      recordGasCost(trxReceipt.gasUsed, 'redeem', filename);
    });

    it.skip('print mint opcode list', async () => {
      await preMint(cToken, minter, mintAmount, mintTokens, exchangeRate);
      const trxReceipt = await quickMint(cToken, minter, mintAmount);
      const opcodeCount = {};
      await saddle.trace(trxReceipt, {
        execLog: log => {
          opcodeCount[log.op] = (opcodeCount[log.op] || 0) + 1;
        }
      });
      console.log(getOpcodeDigest(opcodeCount));
    });
  });

  describe.each([
    ['unitroller-g6'],
    ['unitroller']
  ])('Comp claims %s', (patch) => {
    beforeEach(async () => {
      [root, minter, redeemer, ...accounts] = saddle.accounts;
      comptroller = await makeComptroller({ kind: patch });
      let interestRateModelOpts = {borrowRate: 0.000001};
      cToken = await makeCToken({comptroller, supportMarket: true, underlyingPrice: 2, interestRateModelOpts});
      if (patch == 'unitroller') {
        await send(comptroller, '_setCompSpeed', [cToken._address, etherExp(0.05)]);
      } else {
        await send(comptroller, '_addCompMarkets', [[cToken].map(c => c._address)]);
        await send(comptroller, 'setCompSpeed', [cToken._address, etherExp(0.05)]);
      }
      await send(comptroller.comp, 'transfer', [comptroller._address, etherUnsigned(50e18)], {from: root});
    });

    it(`${patch} second mint with comp accrued`, async () => {
      await mint(cToken, minter, mintAmount, exchangeRate);

      await fastForwardPatch(patch, comptroller, 10);

      console.log('Comp balance before mint', (await compBalance(comptroller, minter)).toString());
      console.log('Comp accrued before mint', (await compAccrued(comptroller, minter)).toString());
      const mint2Receipt = await mint(cToken, minter, mintAmount, exchangeRate);
      console.log('Comp balance after mint', (await compBalance(comptroller, minter)).toString());
      console.log('Comp accrued after mint', (await compAccrued(comptroller, minter)).toString());
      recordGasCost(mint2Receipt.gasUsed, `${patch} second mint with comp accrued`, filename);
    });

    it(`${patch} claim comp`, async () => {
      await mint(cToken, minter, mintAmount, exchangeRate);

      await fastForwardPatch(patch, comptroller, 10);

      console.log('Comp balance before claim', (await compBalance(comptroller, minter)).toString());
      console.log('Comp accrued before claim', (await compAccrued(comptroller, minter)).toString());
      const claimReceipt = await claimComp(comptroller, minter);
      console.log('Comp balance after claim', (await compBalance(comptroller, minter)).toString());
      console.log('Comp accrued after claim', (await compAccrued(comptroller, minter)).toString());
      recordGasCost(claimReceipt.gasUsed, `${patch} claim comp`, filename);
    });
  });
});
