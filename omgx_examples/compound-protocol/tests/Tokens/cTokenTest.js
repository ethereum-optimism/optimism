const {
  etherUnsigned,
  etherMantissa,
  UInt256Max
} = require('../Utils/Ethereum');

const {
  makeCToken,
  setBorrowRate,
  pretendBorrow
} = require('../Utils/Compound');

describe('CToken', function () {
  let root, admin, accounts;
  beforeEach(async () => {
    [root, admin, ...accounts] = saddle.accounts;
  });

  describe('constructor', () => {
    it("fails when non erc-20 underlying", async () => {
      await expect(makeCToken({ underlying: { _address: root } })).rejects.toRevert("revert");
    });

    it("fails when 0 initial exchange rate", async () => {
      await expect(makeCToken({ exchangeRate: 0 })).rejects.toRevert("revert initial exchange rate must be greater than zero.");
    });

    it("succeeds with erc-20 underlying and non-zero exchange rate", async () => {
      const cToken = await makeCToken();
      expect(await call(cToken, 'underlying')).toEqual(cToken.underlying._address);
      expect(await call(cToken, 'admin')).toEqual(root);
    });

    it("succeeds when setting admin to contructor argument", async () => {
      const cToken = await makeCToken({ admin: admin });
      expect(await call(cToken, 'admin')).toEqual(admin);
    });
  });

  describe('name, symbol, decimals', () => {
    let cToken;

    beforeEach(async () => {
      cToken = await makeCToken({ name: "CToken Foo", symbol: "cFOO", decimals: 10 });
    });

    it('should return correct name', async () => {
      expect(await call(cToken, 'name')).toEqual("CToken Foo");
    });

    it('should return correct symbol', async () => {
      expect(await call(cToken, 'symbol')).toEqual("cFOO");
    });

    it('should return correct decimals', async () => {
      expect(await call(cToken, 'decimals')).toEqualNumber(10);
    });
  });

  describe('balanceOfUnderlying', () => {
    it("has an underlying balance", async () => {
      const cToken = await makeCToken({ supportMarket: true, exchangeRate: 2 });
      await send(cToken, 'harnessSetBalance', [root, 100]);
      expect(await call(cToken, 'balanceOfUnderlying', [root])).toEqualNumber(200);
    });
  });

  describe('borrowRatePerBlock', () => {
    it("has a borrow rate", async () => {
      const cToken = await makeCToken({ supportMarket: true, interestRateModelOpts: { kind: 'jump-rate', baseRate: .05, multiplier: 0.45, kink: 0.95, jump: 5 } });
      const perBlock = await call(cToken, 'borrowRatePerBlock');
      expect(Math.abs(perBlock * 2102400 - 5e16)).toBeLessThanOrEqual(1e8);
    });
  });

  describe('supplyRatePerBlock', () => {
    it("returns 0 if there's no supply", async () => {
      const cToken = await makeCToken({ supportMarket: true, interestRateModelOpts: { kind: 'jump-rate', baseRate: .05, multiplier: 0.45, kink: 0.95, jump: 5 } });
      const perBlock = await call(cToken, 'supplyRatePerBlock');
      await expect(perBlock).toEqualNumber(0);
    });

    it("has a supply rate", async () => {
      const baseRate = 0.05;
      const multiplier = 0.45;
      const kink = 0.95;
      const jump = 5 * multiplier;
      const cToken = await makeCToken({ supportMarket: true, interestRateModelOpts: { kind: 'jump-rate', baseRate, multiplier, kink, jump } });
      await send(cToken, 'harnessSetReserveFactorFresh', [etherMantissa(.01)]);
      await send(cToken, 'harnessExchangeRateDetails', [1, 1, 0]);
      await send(cToken, 'harnessSetExchangeRate', [etherMantissa(1)]);
      // Full utilization (Over the kink so jump is included), 1% reserves
      const borrowRate = baseRate + multiplier * kink + jump * .05;
      const expectedSuplyRate = borrowRate * .99;

      const perBlock = await call(cToken, 'supplyRatePerBlock');
      expect(Math.abs(perBlock * 2102400 - expectedSuplyRate * 1e18)).toBeLessThanOrEqual(1e8);
    });
  });

  describe("borrowBalanceCurrent", () => {
    let borrower;
    let cToken;

    beforeEach(async () => {
      borrower = accounts[0];
      cToken = await makeCToken();
    });

    beforeEach(async () => {
      await setBorrowRate(cToken, .001)
      await send(cToken.interestRateModel, 'setFailBorrowRate', [false]);
    });

    it("reverts if interest accrual fails", async () => {
      await send(cToken.interestRateModel, 'setFailBorrowRate', [true]);
      // make sure we accrue interest
      await send(cToken, 'harnessFastForward', [1]);
      await expect(send(cToken, 'borrowBalanceCurrent', [borrower])).rejects.toRevert("revert INTEREST_RATE_MODEL_ERROR");
    });

    it("returns successful result from borrowBalanceStored with no interest", async () => {
      await setBorrowRate(cToken, 0);
      await pretendBorrow(cToken, borrower, 1, 1, 5e18);
      expect(await call(cToken, 'borrowBalanceCurrent', [borrower])).toEqualNumber(5e18)
    });

    it("returns successful result from borrowBalanceCurrent with no interest", async () => {
      await setBorrowRate(cToken, 0);
      await pretendBorrow(cToken, borrower, 1, 3, 5e18);
      expect(await send(cToken, 'harnessFastForward', [5])).toSucceed();
      expect(await call(cToken, 'borrowBalanceCurrent', [borrower])).toEqualNumber(5e18 * 3)
    });
  });

  describe("borrowBalanceStored", () => {
    let borrower;
    let cToken;

    beforeEach(async () => {
      borrower = accounts[0];
      cToken = await makeCToken({ comptrollerOpts: { kind: 'bool' } });
    });

    it("returns 0 for account with no borrows", async () => {
      expect(await call(cToken, 'borrowBalanceStored', [borrower])).toEqualNumber(0)
    });

    it("returns stored principal when account and market indexes are the same", async () => {
      await pretendBorrow(cToken, borrower, 1, 1, 5e18);
      expect(await call(cToken, 'borrowBalanceStored', [borrower])).toEqualNumber(5e18);
    });

    it("returns calculated balance when market index is higher than account index", async () => {
      await pretendBorrow(cToken, borrower, 1, 3, 5e18);
      expect(await call(cToken, 'borrowBalanceStored', [borrower])).toEqualNumber(5e18 * 3);
    });

    it("has undefined behavior when market index is lower than account index", async () => {
      // The market index < account index should NEVER happen, so we don't test this case
    });

    it("reverts on overflow of principal", async () => {
      await pretendBorrow(cToken, borrower, 1, 3, UInt256Max());
      await expect(call(cToken, 'borrowBalanceStored', [borrower])).rejects.toRevert("revert borrowBalanceStored: borrowBalanceStoredInternal failed");
    });

    it("reverts on non-zero stored principal with zero account index", async () => {
      await pretendBorrow(cToken, borrower, 0, 3, 5);
      await expect(call(cToken, 'borrowBalanceStored', [borrower])).rejects.toRevert("revert borrowBalanceStored: borrowBalanceStoredInternal failed");
    });
  });

  describe('exchangeRateStored', () => {
    let cToken, exchangeRate = 2;

    beforeEach(async () => {
      cToken = await makeCToken({ exchangeRate });
    });

    it("returns initial exchange rate with zero cTokenSupply", async () => {
      const result = await call(cToken, 'exchangeRateStored');
      expect(result).toEqualNumber(etherMantissa(exchangeRate));
    });

    it("calculates with single cTokenSupply and single total borrow", async () => {
      const cTokenSupply = 1, totalBorrows = 1, totalReserves = 0;
      await send(cToken, 'harnessExchangeRateDetails', [cTokenSupply, totalBorrows, totalReserves]);
      const result = await call(cToken, 'exchangeRateStored');
      expect(result).toEqualNumber(etherMantissa(1));
    });

    it("calculates with cTokenSupply and total borrows", async () => {
      const cTokenSupply = 100e18, totalBorrows = 10e18, totalReserves = 0;
      await send(cToken, 'harnessExchangeRateDetails', [cTokenSupply, totalBorrows, totalReserves].map(etherUnsigned));
      const result = await call(cToken, 'exchangeRateStored');
      expect(result).toEqualNumber(etherMantissa(.1));
    });

    it("calculates with cash and cTokenSupply", async () => {
      const cTokenSupply = 5e18, totalBorrows = 0, totalReserves = 0;
      expect(
        await send(cToken.underlying, 'transfer', [cToken._address, etherMantissa(500)])
      ).toSucceed();
      await send(cToken, 'harnessExchangeRateDetails', [cTokenSupply, totalBorrows, totalReserves].map(etherUnsigned));
      const result = await call(cToken, 'exchangeRateStored');
      expect(result).toEqualNumber(etherMantissa(100));
    });

    it("calculates with cash, borrows, reserves and cTokenSupply", async () => {
      const cTokenSupply = 500e18, totalBorrows = 500e18, totalReserves = 5e18;
      expect(
        await send(cToken.underlying, 'transfer', [cToken._address, etherMantissa(500)])
      ).toSucceed();
      await send(cToken, 'harnessExchangeRateDetails', [cTokenSupply, totalBorrows, totalReserves].map(etherUnsigned));
      const result = await call(cToken, 'exchangeRateStored');
      expect(result).toEqualNumber(etherMantissa(1.99));
    });
  });

  describe('getCash', () => {
    it("gets the cash", async () => {
      const cToken = await makeCToken();
      const result = await call(cToken, 'getCash');
      expect(result).toEqualNumber(0);
    });
  });
});
