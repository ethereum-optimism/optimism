const BigNum = require('bignumber.js');
const {Ganache} = require('eth-saddle/dist/config');
const {etherUnsigned} = require('../Utils/Ethereum');
const {
  makeInterestRateModel,
  getBorrowRate,
  getSupplyRate
} = require('../Utils/Compound');

const blocksPerYear = 2102400;
const secondsPerYear = 60 * 60 * 24 * 365;

function utilizationRate(cash, borrows, reserves) {
  return borrows ? borrows / (cash + borrows - reserves) : 0;
}

function baseRoofRateFn(dsr, duty, mkrBase, jump, kink, cash, borrows, reserves) {
  const assumedOneMinusReserveFactor = 0.95;
  const stabilityFeePerBlock = (duty + mkrBase - 1) * 15;
  const dsrPerBlock = (dsr - 1) * 15;
  const gapPerBlock = 0.04 / blocksPerYear;
  const jumpPerBlock = jump / blocksPerYear;

  let baseRatePerBlock = dsrPerBlock / assumedOneMinusReserveFactor, multiplierPerBlock;
  if (baseRatePerBlock < stabilityFeePerBlock) {
    multiplierPerBlock = (stabilityFeePerBlock - baseRatePerBlock + gapPerBlock) / kink;
  } else {
    multiplierPerBlock = gapPerBlock / kink;
  }

  const ur = utilizationRate(cash, borrows, reserves);

  if (ur <= kink) {
    return ur * multiplierPerBlock + baseRatePerBlock;
  } else {
    const excessUtil = ur - kink;
    return (excessUtil * jumpPerBlock) + (kink * multiplierPerBlock) + baseRatePerBlock;
  }
}

function daiSupplyRate(dsr, duty, mkrBase, jump, kink, cash, borrows, reserves, reserveFactor = 0.1) {
  const dsrPerBlock = (dsr - 1) * 15;
  const ur = utilizationRate(cash, borrows, reserves);
  const borrowRate = baseRoofRateFn(dsr, duty, mkrBase, jump, kink, cash, borrows, reserves);
  const underlying = cash + borrows - reserves;
  const lendingSupplyRate = borrowRate * (1 - reserveFactor) * ur;

  if (underlying == 0) {
    return lendingSupplyRate;
  }
  const cashSupplyRate = (new BigNum(cash)).times(new BigNum(dsrPerBlock)).div(underlying);
  return cashSupplyRate.plus(lendingSupplyRate).toNumber();
}

let fork = "https://kovan-eth.compound.finance/@14764778";

async function getKovanFork() {
  const kovan = new web3.constructor(
    Ganache.provider({
      allowUnlimitedContractSize: true,
      fork: fork,
      gasLimit: 20000000,
      gasPrice: '20000',
      port: 8546
    }));
  const [root, ...accounts] = await kovan.eth.getAccounts();

  return {kovan, root, accounts};
}

describe('DAIInterestRateModelV3', () => {
  describe("constructor", () => {
    it("sets jug and ilk address and pokes", async () => {
      // NB: Going back a certain distance requires an archive node, currently that add-on is $250/mo
      //  https://community.infura.io/t/error-returned-error-project-id-does-not-have-access-to-archive-state/847
      const {kovan, root, accounts} = await getKovanFork();

      // TODO: Get contract craz
      let {contract: model} = await saddle.deployFull('DAIInterestRateModelV3', [
        etherUnsigned(0.8e18),
        etherUnsigned(0.9e18),
        "0xea190dbdc7adf265260ec4da6e9675fd4f5a78bb",
        "0xcbb7718c9f39d05aeede1c472ca8bf804b2f1ead",
        "0xe3e07f4f3e2f5a5286a99b9b8deed08b8e07550b" // kovan timelock
      ], {gas: 20000000, gasPrice: 20000, from: root}, kovan);

      let args = [0.5e18, 0.45e18, 500].map(etherUnsigned);
      // let mult = await call(model, 'multiplierPerBlock');
      let sr = await call(model, 'getSupplyRate', [...args, etherUnsigned(0.1e18)]);
      // TODO: This doesn't check the return valie?
    });
  });

  describe('getBorrowRate', () => {
    [
      // Description of tests arrays:
      // duty + base = stability fee
      // [dsr, duty, base, cash, borrows, reserves]

      // 2% dsr, 5% duty, 0.5% base
      [0.02e27, 0.05e27, 0.005e27, 500, 100],
      [0.02e27, 0.05e27, 0.005e27, 1000, 900],
      [0.02e27, 0.05e27, 0.005e27, 1000, 950],
      [0.02e27, 0.05e27, 0.005e27, 500, 100],
      [0.02e27, 0.05e27, 0.005e27, 3e18, 5e18],
      [0.02e27, 0.05e27, 0.005e27, 5e18, 3e18],
      [0.02e27, 0.05e27, 0.005e27, 500, 3e18],
      [0.02e27, 0.05e27, 0.005e27, 0, 500],
      [0.02e27, 0.05e27, 0.005e27, 0, 500, 100],
      [0.02e27, 0.05e27, 0.005e27, 500, 0],
      [0.02e27, 0.05e27, 0.005e27, 0, 0],
      [0.02e27, 0.05e27, 0.005e27, 3e18, 500],

      // 5.5% dsr, 18% duty, 0.5% base
      [0.055e27, 0.18e27, 0.005e27, 500, 100],
      [0.055e27, 0.18e27, 0.005e27, 1000, 900],
      [0.055e27, 0.18e27, 0.005e27, 1000, 950],
      [0.055e27, 0.18e27, 0.005e27, 500, 100],
      [0.055e27, 0.18e27, 0.005e27, 3e18, 5e18],
      [0.055e27, 0.18e27, 0.005e27, 5e18, 3e18],
      [0.055e27, 0.18e27, 0.005e27, 500, 3e18],
      [0.055e27, 0.18e27, 0.005e27, 0, 500],
      [0.055e27, 0.18e27, 0.005e27, 0, 500, 100],
      [0.055e27, 0.18e27, 0.005e27, 500, 0],
      [0.055e27, 0.18e27, 0.005e27, 0, 0],
      [0.055e27, 0.18e27, 0.005e27, 3e18, 500],

      // 0% dsr, 10% duty, 0.5% base
      [0e27, 0.1e27, 0.005e27, 500, 100],
      [0e27, 0.1e27, 0.005e27, 1000, 900],
      [0e27, 0.1e27, 0.005e27, 1000, 950],
      [0e27, 0.1e27, 0.005e27, 500, 100],
      [0e27, 0.1e27, 0.005e27, 3e18, 5e18],
      [0e27, 0.1e27, 0.005e27, 5e18, 3e18],
      [0e27, 0.1e27, 0.005e27, 500, 3e18],
      [0e27, 0.1e27, 0.005e27, 0, 500],
      [0e27, 0.1e27, 0.005e27, 0, 500, 100],
      [0e27, 0.1e27, 0.005e27, 500, 0],
      [0e27, 0.1e27, 0.005e27, 0, 0],
      [0e27, 0.1e27, 0.005e27, 3e18, 500],

    ].map(vs => vs.map(Number))
      .forEach(([dsr, duty, base, cash, borrows, reserves = 0, jump = 0.8e18, kink = 0.9e18]) => {
        it(`calculates correct borrow value for dsr=${(dsr / 1e25)}%, duty=${(duty / 1e25)}%, base=${(base / 1e25)}%, jump=${jump / 1e18}, cash=${cash}, borrows=${borrows}, reserves=${reserves}`, async () => {
          const onePlusPerSecondDsr = 1e27 + (dsr / secondsPerYear);
          const onePlusPerSecondDuty = 1e27 + (duty / secondsPerYear);
          const perSecondBase = base / secondsPerYear;

          const pot = await deploy('MockPot', [ etherUnsigned(onePlusPerSecondDsr) ]);
          const jug = await deploy('MockJug', [
            etherUnsigned(onePlusPerSecondDuty),
            etherUnsigned(perSecondBase)
          ]);

          const daiIRM = await deploy('DAIInterestRateModelV3', [
            etherUnsigned(jump),
            etherUnsigned(kink),
            pot._address,
            jug._address,
            "0x0000000000000000000000000000000000000000" // dummy Timelock
          ]);

          const expected = baseRoofRateFn(onePlusPerSecondDsr / 1e27, onePlusPerSecondDuty / 1e27, perSecondBase / 1e27, jump / 1e18, kink / 1e18, cash, borrows, reserves);
          const actual = await getBorrowRate(daiIRM, cash, borrows, reserves);

          expect(Number(actual) / 1e18).toBeWithinDelta(expected, 1e-3);
        });
      });
  });

  describe('getSupplyRate', () => {
    [
      // Description of tests arrays:
      // duty + base = stability fee
      // [dsr, duty, base, cash, borrows, reserves]

      // 2% dsr, 5% duty, 0.5% base
      [0.02e27, 0.05e27, 0.005e27, 500, 100],
      [0.02e27, 0.05e27, 0.005e27, 1000, 900],
      [0.02e27, 0.05e27, 0.005e27, 1000, 950],
      [0.02e27, 0.05e27, 0.005e27, 500, 100],
      [0.02e27, 0.05e27, 0.005e27, 3e18, 5e18],
      [0.02e27, 0.05e27, 0.005e27, 5e18, 3e18],
      [0.02e27, 0.05e27, 0.005e27, 500, 3e18],
      [0.02e27, 0.05e27, 0.005e27, 0, 500],
      [0.02e27, 0.05e27, 0.005e27, 0, 500, 100],
      [0.02e27, 0.05e27, 0.005e27, 500, 0],
      [0.02e27, 0.05e27, 0.005e27, 0, 0],
      [0.02e27, 0.05e27, 0.005e27, 3e18, 500],

      // 5.5% dsr, 18% duty, 0.5% base
      [0.055e27, 0.18e27, 0.005e27, 500, 100],
      [0.055e27, 0.18e27, 0.005e27, 1000, 900],
      [0.055e27, 0.18e27, 0.005e27, 1000, 950],
      [0.055e27, 0.18e27, 0.005e27, 500, 100],
      [0.055e27, 0.18e27, 0.005e27, 3e18, 5e18],
      [0.055e27, 0.18e27, 0.005e27, 5e18, 3e18],
      [0.055e27, 0.18e27, 0.005e27, 500, 3e18],
      [0.055e27, 0.18e27, 0.005e27, 0, 500],
      [0.055e27, 0.18e27, 0.005e27, 0, 500, 100],
      [0.055e27, 0.18e27, 0.005e27, 500, 0],
      [0.055e27, 0.18e27, 0.005e27, 0, 0],
      [0.055e27, 0.18e27, 0.005e27, 3e18, 500],

      // 0% dsr, 10% duty, 0.5% base
      [0e27, 0.1e27, 0.005e27, 500, 100],
      [0e27, 0.1e27, 0.005e27, 1000, 900],
      [0e27, 0.1e27, 0.005e27, 1000, 950],
      [0e27, 0.1e27, 0.005e27, 500, 100],
      [0e27, 0.1e27, 0.005e27, 3e18, 5e18],
      [0e27, 0.1e27, 0.005e27, 5e18, 3e18],
      [0e27, 0.1e27, 0.005e27, 500, 3e18],
      [0e27, 0.1e27, 0.005e27, 0, 500],
      [0e27, 0.1e27, 0.005e27, 0, 500, 100],
      [0e27, 0.1e27, 0.005e27, 500, 0],
      [0e27, 0.1e27, 0.005e27, 0, 0],
      [0e27, 0.1e27, 0.005e27, 3e18, 500],

    ].map(vs => vs.map(Number))
      .forEach(([dsr, duty, base, cash, borrows, reserves = 0, jump = 0.8e18, kink = 0.9e18, reserveFactor = 0.1e18]) => {
        it(`calculates correct supply value for dsr=${(dsr / 1e25)}%, duty=${(duty / 1e25)}%, base=${(base / 1e25)}%, cash=${cash}, borrows=${borrows}, reserves=${reserves}`, async () => {
          const onePlusPerSecondDsr = 1e27 + (dsr / secondsPerYear);
          const onePlusPerSecondDuty = 1e27 + (duty / secondsPerYear);
          const perSecondBase = base / secondsPerYear;

          const pot = await deploy('MockPot', [ etherUnsigned(onePlusPerSecondDsr) ]);
          const jug = await deploy('MockJug', [
            etherUnsigned(onePlusPerSecondDuty),
            etherUnsigned(perSecondBase)
          ]);

          const daiIRM = await deploy('DAIInterestRateModelV3', [
            etherUnsigned(jump),
            etherUnsigned(kink),
            pot._address,
            jug._address,
            "0x0000000000000000000000000000000000000000" // dummy Timelock
          ]);

          const expected = daiSupplyRate(onePlusPerSecondDsr / 1e27, onePlusPerSecondDuty / 1e27, perSecondBase / 1e27, jump / 1e18, kink / 1e18, cash, borrows, reserves, reserveFactor / 1e18);
          const actual = await getSupplyRate(daiIRM, cash, borrows, reserves, reserveFactor);

          expect(Number(actual) / 1e18).toBeWithinDelta(expected, 1e-3);
        });
      });
  });

});
