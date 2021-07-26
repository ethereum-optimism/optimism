const {
  etherGasCost,
  etherUnsigned,
  UInt256Max
} = require('../Utils/Ethereum');

const {
  makeCToken,
  fastForward,
  setBalance,
  getBalances,
  adjustBalances,
  pretendBorrow,
  preApprove
} = require('../Utils/Compound');

const repayAmount = etherUnsigned(10e2);
const seizeAmount = repayAmount;
const seizeTokens = seizeAmount.multipliedBy(4); // forced

async function preLiquidate(cToken, liquidator, borrower, repayAmount, cTokenCollateral) {
  // setup for success in liquidating
  await send(cToken.comptroller, 'setLiquidateBorrowAllowed', [true]);
  await send(cToken.comptroller, 'setLiquidateBorrowVerify', [true]);
  await send(cToken.comptroller, 'setRepayBorrowAllowed', [true]);
  await send(cToken.comptroller, 'setRepayBorrowVerify', [true]);
  await send(cToken.comptroller, 'setSeizeAllowed', [true]);
  await send(cToken.comptroller, 'setSeizeVerify', [true]);
  await send(cToken.comptroller, 'setFailCalculateSeizeTokens', [false]);
  await send(cToken.underlying, 'harnessSetFailTransferFromAddress', [liquidator, false]);
  await send(cToken.interestRateModel, 'setFailBorrowRate', [false]);
  await send(cTokenCollateral.interestRateModel, 'setFailBorrowRate', [false]);
  await send(cTokenCollateral.comptroller, 'setCalculatedSeizeTokens', [seizeTokens]);
  await setBalance(cTokenCollateral, liquidator, 0);
  await setBalance(cTokenCollateral, borrower, seizeTokens);
  await pretendBorrow(cTokenCollateral, borrower, 0, 1, 0);
  await pretendBorrow(cToken, borrower, 1, 1, repayAmount);
  await preApprove(cToken, liquidator, repayAmount);
}

async function liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral) {
  return send(cToken, 'harnessLiquidateBorrowFresh', [liquidator, borrower, repayAmount, cTokenCollateral._address]);
}

async function liquidate(cToken, liquidator, borrower, repayAmount, cTokenCollateral) {
  // make sure to have a block delta so we accrue interest
  await fastForward(cToken, 1);
  await fastForward(cTokenCollateral, 1);
  return send(cToken, 'liquidateBorrow', [borrower, repayAmount, cTokenCollateral._address], {from: liquidator});
}

async function seize(cToken, liquidator, borrower, seizeAmount) {
  return send(cToken, 'seize', [liquidator, borrower, seizeAmount]);
}

describe('CToken', function () {
  let root, liquidator, borrower, accounts;
  let cToken, cTokenCollateral;

  beforeEach(async () => {
    [root, liquidator, borrower, ...accounts] = saddle.accounts;
    cToken = await makeCToken({comptrollerOpts: {kind: 'bool'}});
    cTokenCollateral = await makeCToken({comptroller: cToken.comptroller});
  });

  beforeEach(async () => {
    await preLiquidate(cToken, liquidator, borrower, repayAmount, cTokenCollateral);
  });

  describe('liquidateBorrowFresh', () => {
    it("fails if comptroller tells it to", async () => {
      await send(cToken.comptroller, 'setLiquidateBorrowAllowed', [false]);
      expect(
        await liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).toHaveTrollReject('LIQUIDATE_COMPTROLLER_REJECTION', 'MATH_ERROR');
    });

    it("proceeds if comptroller tells it to", async () => {
      expect(
        await liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).toSucceed();
    });

    it("fails if market not fresh", async () => {
      await fastForward(cToken);
      expect(
        await liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).toHaveTokenFailure('MARKET_NOT_FRESH', 'LIQUIDATE_FRESHNESS_CHECK');
    });

    it("fails if collateral market not fresh", async () => {
      await fastForward(cToken);
      await fastForward(cTokenCollateral);
      await send(cToken, 'accrueInterest');
      expect(
        await liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).toHaveTokenFailure('MARKET_NOT_FRESH', 'LIQUIDATE_COLLATERAL_FRESHNESS_CHECK');
    });

    it("fails if borrower is equal to liquidator", async () => {
      expect(
        await liquidateFresh(cToken, borrower, borrower, repayAmount, cTokenCollateral)
      ).toHaveTokenFailure('INVALID_ACCOUNT_PAIR', 'LIQUIDATE_LIQUIDATOR_IS_BORROWER');
    });

    it("fails if repayAmount = 0", async () => {
      expect(await liquidateFresh(cToken, liquidator, borrower, 0, cTokenCollateral)).toHaveTokenFailure('INVALID_CLOSE_AMOUNT_REQUESTED', 'LIQUIDATE_CLOSE_AMOUNT_IS_ZERO');
    });

    it("fails if calculating seize tokens fails and does not adjust balances", async () => {
      const beforeBalances = await getBalances([cToken, cTokenCollateral], [liquidator, borrower]);
      await send(cToken.comptroller, 'setFailCalculateSeizeTokens', [true]);
      await expect(
        liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).rejects.toRevert('revert LIQUIDATE_COMPTROLLER_CALCULATE_AMOUNT_SEIZE_FAILED');
      const afterBalances = await getBalances([cToken, cTokenCollateral], [liquidator, borrower]);
      expect(afterBalances).toEqual(beforeBalances);
    });

    it("fails if repay fails", async () => {
      await send(cToken.comptroller, 'setRepayBorrowAllowed', [false]);
      expect(
        await liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).toHaveTrollReject('LIQUIDATE_REPAY_BORROW_FRESH_FAILED');
    });

    it("reverts if seize fails", async () => {
      await send(cToken.comptroller, 'setSeizeAllowed', [false]);
      await expect(
        liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).rejects.toRevert("revert token seizure failed");
    });

    xit("reverts if liquidateBorrowVerify fails", async() => {
      await send(cToken.comptroller, 'setLiquidateBorrowVerify', [false]);
      await expect(
        liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral)
      ).rejects.toRevert("revert liquidateBorrowVerify rejected liquidateBorrow");
    });

    it("transfers the cash, borrows, tokens, and emits Transfer, LiquidateBorrow events", async () => {
      const beforeBalances = await getBalances([cToken, cTokenCollateral], [liquidator, borrower]);
      const result = await liquidateFresh(cToken, liquidator, borrower, repayAmount, cTokenCollateral);
      const afterBalances = await getBalances([cToken, cTokenCollateral], [liquidator, borrower]);
      expect(result).toSucceed();
      expect(result).toHaveLog('LiquidateBorrow', {
        liquidator: liquidator,
        borrower: borrower,
        repayAmount: repayAmount.toString(),
        cTokenCollateral: cTokenCollateral._address,
        seizeTokens: seizeTokens.toString()
      });
      expect(result).toHaveLog(['Transfer', 0], {
        from: liquidator,
        to: cToken._address,
        amount: repayAmount.toString()
      });
      expect(result).toHaveLog(['Transfer', 1], {
        from: borrower,
        to: liquidator,
        amount: seizeTokens.toString()
      });
      expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
        [cToken, 'cash', repayAmount],
        [cToken, 'borrows', -repayAmount],
        [cToken, liquidator, 'cash', -repayAmount],
        [cTokenCollateral, liquidator, 'tokens', seizeTokens],
        [cToken, borrower, 'borrows', -repayAmount],
        [cTokenCollateral, borrower, 'tokens', -seizeTokens]
      ]));
    });
  });

  describe('liquidateBorrow', () => {
    it("emits a liquidation failure if borrowed asset interest accrual fails", async () => {
      await send(cToken.interestRateModel, 'setFailBorrowRate', [true]);
      await expect(liquidate(cToken, liquidator, borrower, repayAmount, cTokenCollateral)).rejects.toRevert("revert INTEREST_RATE_MODEL_ERROR");
    });

    it("emits a liquidation failure if collateral asset interest accrual fails", async () => {
      await send(cTokenCollateral.interestRateModel, 'setFailBorrowRate', [true]);
      await expect(liquidate(cToken, liquidator, borrower, repayAmount, cTokenCollateral)).rejects.toRevert("revert INTEREST_RATE_MODEL_ERROR");
    });

    it("returns error from liquidateBorrowFresh without emitting any extra logs", async () => {
      expect(await liquidate(cToken, liquidator, borrower, 0, cTokenCollateral)).toHaveTokenFailure('INVALID_CLOSE_AMOUNT_REQUESTED', 'LIQUIDATE_CLOSE_AMOUNT_IS_ZERO');
    });

    it("returns success from liquidateBorrowFresh and transfers the correct amounts", async () => {
      const beforeBalances = await getBalances([cToken, cTokenCollateral], [liquidator, borrower]);
      const result = await liquidate(cToken, liquidator, borrower, repayAmount, cTokenCollateral);
      const gasCost = await etherGasCost(result);
      const afterBalances = await getBalances([cToken, cTokenCollateral], [liquidator, borrower]);
      expect(result).toSucceed();
      expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
        [cToken, 'cash', repayAmount],
        [cToken, 'borrows', -repayAmount],
        [cToken, liquidator, 'eth', -gasCost],
        [cToken, liquidator, 'cash', -repayAmount],
        [cTokenCollateral, liquidator, 'eth', -gasCost],
        [cTokenCollateral, liquidator, 'tokens', seizeTokens],
        [cToken, borrower, 'borrows', -repayAmount],
        [cTokenCollateral, borrower, 'tokens', -seizeTokens]
      ]));
    });
  });

  describe('seize', () => {
    // XXX verify callers are properly checked

    it("fails if seize is not allowed", async () => {
      await send(cToken.comptroller, 'setSeizeAllowed', [false]);
      expect(await seize(cTokenCollateral, liquidator, borrower, seizeTokens)).toHaveTrollReject('LIQUIDATE_SEIZE_COMPTROLLER_REJECTION', 'MATH_ERROR');
    });

    it("fails if cTokenBalances[borrower] < amount", async () => {
      await setBalance(cTokenCollateral, borrower, 1);
      expect(await seize(cTokenCollateral, liquidator, borrower, seizeTokens)).toHaveTokenMathFailure('LIQUIDATE_SEIZE_BALANCE_DECREMENT_FAILED', 'INTEGER_UNDERFLOW');
    });

    it("fails if cTokenBalances[liquidator] overflows", async () => {
      await setBalance(cTokenCollateral, liquidator, UInt256Max());
      expect(await seize(cTokenCollateral, liquidator, borrower, seizeTokens)).toHaveTokenMathFailure('LIQUIDATE_SEIZE_BALANCE_INCREMENT_FAILED', 'INTEGER_OVERFLOW');
    });

    it("succeeds, updates balances, and emits Transfer event", async () => {
      const beforeBalances = await getBalances([cTokenCollateral], [liquidator, borrower]);
      const result = await seize(cTokenCollateral, liquidator, borrower, seizeTokens);
      const afterBalances = await getBalances([cTokenCollateral], [liquidator, borrower]);
      expect(result).toSucceed();
      expect(result).toHaveLog('Transfer', {
        from: borrower,
        to: liquidator,
        amount: seizeTokens.toString()
      });
      expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
        [cTokenCollateral, liquidator, 'tokens', seizeTokens],
        [cTokenCollateral, borrower, 'tokens', -seizeTokens]
      ]));
    });
  });
});
