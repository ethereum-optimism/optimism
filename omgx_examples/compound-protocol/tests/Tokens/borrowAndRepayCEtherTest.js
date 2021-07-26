const {
  etherGasCost,
  etherUnsigned,
  etherMantissa,
  UInt256Max
} = require('../Utils/Ethereum');

const {
  makeCToken,
  balanceOf,
  borrowSnapshot,
  totalBorrows,
  fastForward,
  setBalance,
  preApprove,
  pretendBorrow,
  setEtherBalance,
  getBalances,
  adjustBalances
} = require('../Utils/Compound');

const BigNumber = require('bignumber.js');

const borrowAmount = etherUnsigned(10e3);
const repayAmount = etherUnsigned(10e2);

async function preBorrow(cToken, borrower, borrowAmount) {
  await send(cToken.comptroller, 'setBorrowAllowed', [true]);
  await send(cToken.comptroller, 'setBorrowVerify', [true]);
  await send(cToken.interestRateModel, 'setFailBorrowRate', [false]);
  await send(cToken, 'harnessSetFailTransferToAddress', [borrower, false]);
  await send(cToken, 'harnessSetAccountBorrows', [borrower, 0, 0]);
  await send(cToken, 'harnessSetTotalBorrows', [0]);
  await setEtherBalance(cToken, borrowAmount);
}

async function borrowFresh(cToken, borrower, borrowAmount) {
  return send(cToken, 'harnessBorrowFresh', [borrower, borrowAmount], {from: borrower});
}

async function borrow(cToken, borrower, borrowAmount, opts = {}) {
  await send(cToken, 'harnessFastForward', [1]);
  return send(cToken, 'borrow', [borrowAmount], {from: borrower});
}

async function preRepay(cToken, benefactor, borrower, repayAmount) {
  // setup either benefactor OR borrower for success in repaying
  await send(cToken.comptroller, 'setRepayBorrowAllowed', [true]);
  await send(cToken.comptroller, 'setRepayBorrowVerify', [true]);
  await send(cToken.interestRateModel, 'setFailBorrowRate', [false]);
  await pretendBorrow(cToken, borrower, 1, 1, repayAmount);
}

async function repayBorrowFresh(cToken, payer, borrower, repayAmount) {
  return send(cToken, 'harnessRepayBorrowFresh', [payer, borrower, repayAmount], {from: payer, value: repayAmount});
}

async function repayBorrow(cToken, borrower, repayAmount) {
  await send(cToken, 'harnessFastForward', [1]);
  return send(cToken, 'repayBorrow', [], {from: borrower, value: repayAmount});
}

async function repayBorrowBehalf(cToken, payer, borrower, repayAmount) {
  await send(cToken, 'harnessFastForward', [1]);
  return send(cToken, 'repayBorrowBehalf', [borrower], {from: payer, value: repayAmount});
}

describe('CEther', function () {
  let cToken, root, borrower, benefactor, accounts;
  beforeEach(async () => {
    [root, borrower, benefactor, ...accounts] = saddle.accounts;
    cToken = await makeCToken({kind: 'cether', comptrollerOpts: {kind: 'bool'}});
  });

  describe('borrowFresh', () => {
    beforeEach(async () => await preBorrow(cToken, borrower, borrowAmount));

    it("fails if comptroller tells it to", async () => {
      await send(cToken.comptroller, 'setBorrowAllowed', [false]);
      expect(await borrowFresh(cToken, borrower, borrowAmount)).toHaveTrollReject('BORROW_COMPTROLLER_REJECTION');
    });

    it("proceeds if comptroller tells it to", async () => {
      await expect(await borrowFresh(cToken, borrower, borrowAmount)).toSucceed();
    });

    it("fails if market not fresh", async () => {
      await fastForward(cToken);
      expect(await borrowFresh(cToken, borrower, borrowAmount)).toHaveTokenFailure('MARKET_NOT_FRESH', 'BORROW_FRESHNESS_CHECK');
    });

    it("continues if fresh", async () => {
      await expect(await send(cToken, 'accrueInterest')).toSucceed();
      await expect(await borrowFresh(cToken, borrower, borrowAmount)).toSucceed();
    });

    it("fails if protocol has less than borrowAmount of underlying", async () => {
      expect(await borrowFresh(cToken, borrower, borrowAmount.plus(1))).toHaveTokenFailure('TOKEN_INSUFFICIENT_CASH', 'BORROW_CASH_NOT_AVAILABLE');
    });

    it("fails if borrowBalanceStored fails (due to non-zero stored principal with zero account index)", async () => {
      await pretendBorrow(cToken, borrower, 0, 3e18, 5e18);
      expect(await borrowFresh(cToken, borrower, borrowAmount)).toHaveTokenFailure('MATH_ERROR', 'BORROW_ACCUMULATED_BALANCE_CALCULATION_FAILED');
    });

    it("fails if calculating account new total borrow balance overflows", async () => {
      await pretendBorrow(cToken, borrower, 1e-18, 1e-18, UInt256Max());
      expect(await borrowFresh(cToken, borrower, borrowAmount)).toHaveTokenFailure('MATH_ERROR', 'BORROW_NEW_ACCOUNT_BORROW_BALANCE_CALCULATION_FAILED');
    });

    it("fails if calculation of new total borrow balance overflows", async () => {
      await send(cToken, 'harnessSetTotalBorrows', [UInt256Max()]);
      expect(await borrowFresh(cToken, borrower, borrowAmount)).toHaveTokenFailure('MATH_ERROR', 'BORROW_NEW_TOTAL_BALANCE_CALCULATION_FAILED');
    });

    it("reverts if transfer out fails", async () => {
      await send(cToken, 'harnessSetFailTransferToAddress', [borrower, true]);
      await expect(borrowFresh(cToken, borrower, borrowAmount)).rejects.toRevert("revert TOKEN_TRANSFER_OUT_FAILED");
    });

    xit("reverts if borrowVerify fails", async() => {
      await send(cToken.comptroller, 'setBorrowVerify', [false]);
      await expect(borrowFresh(cToken, borrower, borrowAmount)).rejects.toRevert("revert borrowVerify rejected borrow");
    });

    it("transfers the underlying cash, tokens, and emits Borrow event", async () => {
      const beforeBalances = await getBalances([cToken], [borrower]);
      const beforeProtocolBorrows = await totalBorrows(cToken);
      const result = await borrowFresh(cToken, borrower, borrowAmount);
      const afterBalances = await getBalances([cToken], [borrower]);
      expect(result).toSucceed();
      expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
        [cToken, 'eth', -borrowAmount],
        [cToken, 'borrows', borrowAmount],
        [cToken, borrower, 'eth', borrowAmount.minus(await etherGasCost(result))],
        [cToken, borrower, 'borrows', borrowAmount]
      ]));
      expect(result).toHaveLog('Borrow', {
        borrower: borrower,
        borrowAmount: borrowAmount.toString(),
        accountBorrows: borrowAmount.toString(),
        totalBorrows: beforeProtocolBorrows.plus(borrowAmount).toString()
      });
    });

    it("stores new borrow principal and interest index", async () => {
      const beforeProtocolBorrows = await totalBorrows(cToken);
      await pretendBorrow(cToken, borrower, 0, 3, 0);
      await borrowFresh(cToken, borrower, borrowAmount);
      const borrowSnap = await borrowSnapshot(cToken, borrower);
      expect(borrowSnap.principal).toEqualNumber(borrowAmount);
      expect(borrowSnap.interestIndex).toEqualNumber(etherMantissa(3));
      expect(await totalBorrows(cToken)).toEqualNumber(beforeProtocolBorrows.plus(borrowAmount));
    });
  });

  describe('borrow', () => {
    beforeEach(async () => await preBorrow(cToken, borrower, borrowAmount));

    it("emits a borrow failure if interest accrual fails", async () => {
      await send(cToken.interestRateModel, 'setFailBorrowRate', [true]);
      await send(cToken, 'harnessFastForward', [1]);
      await expect(borrow(cToken, borrower, borrowAmount)).rejects.toRevert("revert INTEREST_RATE_MODEL_ERROR");
    });

    it("returns error from borrowFresh without emitting any extra logs", async () => {
      expect(await borrow(cToken, borrower, borrowAmount.plus(1))).toHaveTokenFailure('TOKEN_INSUFFICIENT_CASH', 'BORROW_CASH_NOT_AVAILABLE');
    });

    it("returns success from borrowFresh and transfers the correct amount", async () => {
      const beforeBalances = await getBalances([cToken], [borrower]);
      await fastForward(cToken);
      const result = await borrow(cToken, borrower, borrowAmount);
      const afterBalances = await getBalances([cToken], [borrower]);
      expect(result).toSucceed();
      expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
        [cToken, 'eth', -borrowAmount],
        [cToken, 'borrows', borrowAmount],
        [cToken, borrower, 'eth', borrowAmount.minus(await etherGasCost(result))],
        [cToken, borrower, 'borrows', borrowAmount]
      ]));
    });
  });

  describe('repayBorrowFresh', () => {
    [true, false].forEach(async (benefactorPaying) => {
      let payer;
      const label = benefactorPaying ? "benefactor paying" : "borrower paying";
      describe(label, () => {
        beforeEach(async () => {
          payer = benefactorPaying ? benefactor : borrower;

          await preRepay(cToken, payer, borrower, repayAmount);
        });

        it("fails if repay is not allowed", async () => {
          await send(cToken.comptroller, 'setRepayBorrowAllowed', [false]);
          expect(await repayBorrowFresh(cToken, payer, borrower, repayAmount)).toHaveTrollReject('REPAY_BORROW_COMPTROLLER_REJECTION', 'MATH_ERROR');
        });

        it("fails if block number ≠ current block number", async () => {
          await fastForward(cToken);
          expect(await repayBorrowFresh(cToken, payer, borrower, repayAmount)).toHaveTokenFailure('MARKET_NOT_FRESH', 'REPAY_BORROW_FRESHNESS_CHECK');
        });

        it("returns an error if calculating account new account borrow balance fails", async () => {
          await pretendBorrow(cToken, borrower, 1, 1, 1);
          await expect(repayBorrowFresh(cToken, payer, borrower, repayAmount)).rejects.toRevert('revert REPAY_BORROW_NEW_ACCOUNT_BORROW_BALANCE_CALCULATION_FAILED');
        });

        it("returns an error if calculation of new total borrow balance fails", async () => {
          await send(cToken, 'harnessSetTotalBorrows', [1]);
          await expect(repayBorrowFresh(cToken, payer, borrower, repayAmount)).rejects.toRevert('revert REPAY_BORROW_NEW_TOTAL_BALANCE_CALCULATION_FAILED');
        });

        it("reverts if checkTransferIn fails", async () => {
          await expect(
            send(cToken, 'harnessRepayBorrowFresh', [payer, borrower, repayAmount], {from: root, value: repayAmount})
          ).rejects.toRevert("revert sender mismatch");
          await expect(
            send(cToken, 'harnessRepayBorrowFresh', [payer, borrower, repayAmount], {from: payer, value: 1})
          ).rejects.toRevert("revert value mismatch");
        });

        xit("reverts if repayBorrowVerify fails", async() => {
          await send(cToken.comptroller, 'setRepayBorrowVerify', [false]);
          await expect(repayBorrowFresh(cToken, payer, borrower, repayAmount)).rejects.toRevert("revert repayBorrowVerify rejected repayBorrow");
        });

        it("transfers the underlying cash, and emits RepayBorrow event", async () => {
          const beforeBalances = await getBalances([cToken], [borrower]);
          const result = await repayBorrowFresh(cToken, payer, borrower, repayAmount);
          const afterBalances = await getBalances([cToken], [borrower]);
          expect(result).toSucceed();
          if (borrower == payer) {
            expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
              [cToken, 'eth', repayAmount],
              [cToken, 'borrows', -repayAmount],
              [cToken, borrower, 'borrows', -repayAmount],
              [cToken, borrower, 'eth', -repayAmount.plus(await etherGasCost(result))]
            ]));
          } else {
            expect(afterBalances).toEqual(await adjustBalances(beforeBalances, [
              [cToken, 'eth', repayAmount],
              [cToken, 'borrows', -repayAmount],
              [cToken, borrower, 'borrows', -repayAmount],
            ]));
          }
          expect(result).toHaveLog('RepayBorrow', {
            payer: payer,
            borrower: borrower,
            repayAmount: repayAmount.toString(),
            accountBorrows: "0",
            totalBorrows: "0"
          });
        });

        it("stores new borrow principal and interest index", async () => {
          const beforeProtocolBorrows = await totalBorrows(cToken);
          const beforeAccountBorrowSnap = await borrowSnapshot(cToken, borrower);
          expect(await repayBorrowFresh(cToken, payer, borrower, repayAmount)).toSucceed();
          const afterAccountBorrows = await borrowSnapshot(cToken, borrower);
          expect(afterAccountBorrows.principal).toEqualNumber(beforeAccountBorrowSnap.principal.minus(repayAmount));
          expect(afterAccountBorrows.interestIndex).toEqualNumber(etherMantissa(1));
          expect(await totalBorrows(cToken)).toEqualNumber(beforeProtocolBorrows.minus(repayAmount));
        });
      });
    });
  });

  describe('repayBorrow', () => {
    beforeEach(async () => {
      await preRepay(cToken, borrower, borrower, repayAmount);
    });

    it("reverts if interest accrual fails", async () => {
      await send(cToken.interestRateModel, 'setFailBorrowRate', [true]);
      await expect(repayBorrow(cToken, borrower, repayAmount)).rejects.toRevert("revert INTEREST_RATE_MODEL_ERROR");
    });

    it("reverts when repay borrow fresh fails", async () => {
      await send(cToken.comptroller, 'setRepayBorrowAllowed', [false]);
      await expect(repayBorrow(cToken, borrower, repayAmount)).rejects.toRevertWithError('COMPTROLLER_REJECTION', "revert repayBorrow failed");
    });

    it("returns success from repayBorrowFresh and repays the right amount", async () => {
      await fastForward(cToken);
      const beforeAccountBorrowSnap = await borrowSnapshot(cToken, borrower);
      expect(await repayBorrow(cToken, borrower, repayAmount)).toSucceed();
      const afterAccountBorrowSnap = await borrowSnapshot(cToken, borrower);
      expect(afterAccountBorrowSnap.principal).toEqualNumber(beforeAccountBorrowSnap.principal.minus(repayAmount));
    });

    it("reverts if overpaying", async () => {
      const beforeAccountBorrowSnap = await borrowSnapshot(cToken, borrower);
      let tooMuch = new BigNumber(beforeAccountBorrowSnap.principal).plus(1);
      await expect(repayBorrow(cToken, borrower, tooMuch)).rejects.toRevert("revert REPAY_BORROW_NEW_ACCOUNT_BORROW_BALANCE_CALCULATION_FAILED");
      // await assert.toRevertWithError(repayBorrow(cToken, borrower, tooMuch), 'MATH_ERROR', "revert repayBorrow failed");
    });
  });

  describe('repayBorrowBehalf', () => {
    let payer;

    beforeEach(async () => {
      payer = benefactor;
      await preRepay(cToken, payer, borrower, repayAmount);
    });

    it("reverts if interest accrual fails", async () => {
      await send(cToken.interestRateModel, 'setFailBorrowRate', [true]);
      await expect(repayBorrowBehalf(cToken, payer, borrower, repayAmount)).rejects.toRevert("revert INTEREST_RATE_MODEL_ERROR");
    });

    it("reverts from within repay borrow fresh", async () => {
      await send(cToken.comptroller, 'setRepayBorrowAllowed', [false]);
      await expect(repayBorrowBehalf(cToken, payer, borrower, repayAmount)).rejects.toRevertWithError('COMPTROLLER_REJECTION', "revert repayBorrowBehalf failed");
    });

    it("returns success from repayBorrowFresh and repays the right amount", async () => {
      await fastForward(cToken);
      const beforeAccountBorrowSnap = await borrowSnapshot(cToken, borrower);
      expect(await repayBorrowBehalf(cToken, payer, borrower, repayAmount)).toSucceed();
      const afterAccountBorrowSnap = await borrowSnapshot(cToken, borrower);
      expect(afterAccountBorrowSnap.principal).toEqualNumber(beforeAccountBorrowSnap.principal.minus(repayAmount));
    });
  });
});
