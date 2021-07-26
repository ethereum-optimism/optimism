const { address, both, etherMantissa } = require('../Utils/Ethereum');
const { makeComptroller, makeCToken } = require('../Utils/Compound');

describe('Comptroller', () => {
  let comptroller, cToken;
  let root, accounts;

  beforeEach(async () => {
    [root, ...accounts] = saddle.accounts;
  });

  describe("_setPauseGuardian", () => {
    beforeEach(async () => {
      comptroller = await makeComptroller();
    });

    describe("failing", () => {
      it("emits a failure log if not sent by admin", async () => {
        let result = await send(comptroller, '_setPauseGuardian', [root], {from: accounts[1]});
        expect(result).toHaveTrollFailure('UNAUTHORIZED', 'SET_PAUSE_GUARDIAN_OWNER_CHECK');
      });

      it("does not change the pause guardian", async () => {
        let pauseGuardian = await call(comptroller, 'pauseGuardian');
        expect(pauseGuardian).toEqual(address(0));
        await send(comptroller, '_setPauseGuardian', [root], {from: accounts[1]});

        pauseGuardian = await call(comptroller, 'pauseGuardian');
        expect(pauseGuardian).toEqual(address(0));
      });
    });


    describe('succesfully changing pause guardian', () => {
      let result;

      beforeEach(async () => {
        comptroller = await makeComptroller();

        result = await send(comptroller, '_setPauseGuardian', [accounts[1]]);
      });

      it('emits new pause guardian event', async () => {
        expect(result).toHaveLog(
          'NewPauseGuardian',
          {newPauseGuardian: accounts[1], oldPauseGuardian: address(0)}
        );
      });

      it('changes pending pause guardian', async () => {
        let pauseGuardian = await call(comptroller, 'pauseGuardian');
        expect(pauseGuardian).toEqual(accounts[1]);
      });
    });
  });

  describe('setting paused', () => {
    beforeEach(async () => {
      cToken = await makeCToken({supportMarket: true});
      comptroller = cToken.comptroller;
    });

    let globalMethods = ["Transfer", "Seize"];
    describe('succeeding', () => {
      let pauseGuardian;
      beforeEach(async () => {
        pauseGuardian = accounts[1];
        await send(comptroller, '_setPauseGuardian', [accounts[1]], {from: root});
      });

      globalMethods.forEach(async (method) => {
        it(`only pause guardian or admin can pause ${method}`, async () => {
          await expect(send(comptroller, `_set${method}Paused`, [true], {from: accounts[2]})).rejects.toRevert("revert only pause guardian and admin can pause");
          await expect(send(comptroller, `_set${method}Paused`, [false], {from: accounts[2]})).rejects.toRevert("revert only pause guardian and admin can pause");
        });

        it(`PauseGuardian can pause of ${method}GuardianPaused`, async () => {
          result = await send(comptroller, `_set${method}Paused`, [true], {from: pauseGuardian});
          expect(result).toHaveLog(`ActionPaused`, {action: method, pauseState: true});

          let camelCase = method.charAt(0).toLowerCase() + method.substring(1);

          state = await call(comptroller, `${camelCase}GuardianPaused`);
          expect(state).toEqual(true);

          await expect(send(comptroller, `_set${method}Paused`, [false], {from: pauseGuardian})).rejects.toRevert("revert only admin can unpause");
          result = await send(comptroller, `_set${method}Paused`, [false]);

          expect(result).toHaveLog(`ActionPaused`, {action: method, pauseState: false});

          state = await call(comptroller, `${camelCase}GuardianPaused`);
          expect(state).toEqual(false);
        });

        it(`pauses ${method}`, async() => {
          await send(comptroller, `_set${method}Paused`, [true], {from: pauseGuardian});
          switch (method) {
          case "Transfer":
            await expect(
              send(comptroller, 'transferAllowed', [address(1), address(2), address(3), 1])
            ).rejects.toRevert(`revert ${method.toLowerCase()} is paused`);
            break;

          case "Seize":
            await expect(
              send(comptroller, 'seizeAllowed', [address(1), address(2), address(3), address(4), 1])
            ).rejects.toRevert(`revert ${method.toLowerCase()} is paused`);
            break;

          default:
            break;
          }
        });
      });
    });

    let marketMethods = ["Borrow", "Mint"];
    describe('succeeding', () => {
      let pauseGuardian;
      beforeEach(async () => {
        pauseGuardian = accounts[1];
        await send(comptroller, '_setPauseGuardian', [accounts[1]], {from: root});
      });

      marketMethods.forEach(async (method) => {
        it(`only pause guardian or admin can pause ${method}`, async () => {
          await expect(send(comptroller, `_set${method}Paused`, [cToken._address, true], {from: accounts[2]})).rejects.toRevert("revert only pause guardian and admin can pause");
          await expect(send(comptroller, `_set${method}Paused`, [cToken._address, false], {from: accounts[2]})).rejects.toRevert("revert only pause guardian and admin can pause");
        });

        it(`PauseGuardian can pause of ${method}GuardianPaused`, async () => {
          result = await send(comptroller, `_set${method}Paused`, [cToken._address, true], {from: pauseGuardian});
          expect(result).toHaveLog(`ActionPaused`, {cToken: cToken._address, action: method, pauseState: true});

          let camelCase = method.charAt(0).toLowerCase() + method.substring(1);

          state = await call(comptroller, `${camelCase}GuardianPaused`, [cToken._address]);
          expect(state).toEqual(true);

          await expect(send(comptroller, `_set${method}Paused`, [cToken._address, false], {from: pauseGuardian})).rejects.toRevert("revert only admin can unpause");
          result = await send(comptroller, `_set${method}Paused`, [cToken._address, false]);

          expect(result).toHaveLog(`ActionPaused`, {cToken: cToken._address, action: method, pauseState: false});

          state = await call(comptroller, `${camelCase}GuardianPaused`, [cToken._address]);
          expect(state).toEqual(false);
        });

        it(`pauses ${method}`, async() => {
          await send(comptroller, `_set${method}Paused`, [cToken._address, true], {from: pauseGuardian});
          switch (method) {
          case "Mint":
            expect(await call(comptroller, 'mintAllowed', [address(1), address(2), 1])).toHaveTrollError('MARKET_NOT_LISTED');
            await expect(send(comptroller, 'mintAllowed', [cToken._address, address(2), 1])).rejects.toRevert(`revert ${method.toLowerCase()} is paused`);
            break;

          case "Borrow":
            expect(await call(comptroller, 'borrowAllowed', [address(1), address(2), 1])).toHaveTrollError('MARKET_NOT_LISTED');
            await expect(send(comptroller, 'borrowAllowed', [cToken._address, address(2), 1])).rejects.toRevert(`revert ${method.toLowerCase()} is paused`);
            break;

          default:
            break;
          }
        });
      });
    });
  });
});
