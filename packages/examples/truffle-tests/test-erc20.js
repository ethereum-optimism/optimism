const TestERC20 = artifacts.require("TestERC20");

contract('TestERC20', accounts => {
    const account = accounts[0];
    const account2 = accounts[1];
    const initialValue = 100000000;
    const transferValue = 2000;
    let erc20;

    beforeEach(async () => {
        // Deploy a fresh contract every test
        erc20 = await TestERC20.new(initialValue, 'Test Token', 100, 'TEST',  {from: account});
    });

    it('gets total supply', async () => {
        const totalSupply = await erc20.totalSupply.call();
        assert.equal(totalSupply, initialValue, 'Total supply mismatch!');
    });

    it('gets balance of main account', async () => {
        const balance = await erc20.balanceOf.call(account);
        assert.equal(balance, initialValue, 'Sender balance mismatch!');
    });

    it('gets 0 balance of second account', async () => {
        const balance = await erc20.balanceOf.call(account2);
        assert.equal(balance, 0, 'Second account balance mismatch!');
    });

    it('transfers from account 1 to account 2', async () => {
        await erc20.transfer(account2, transferValue, {from: account});

        const accountBalance = await erc20.balanceOf.call(account);
        assert.equal(accountBalance, initialValue - transferValue, 'Account balance mismatch after transfer!');

        const account2Balance = await erc20.balanceOf.call(account2);
        assert.equal(account2Balance, transferValue, 'Account balance mismatch after transfer!');
    });

    it('transfers from account 1 to account 2 and then back', async () => {
        await erc20.transfer(account2, transferValue, {from: account});
        await erc20.transfer(account, transferValue, {from: account2});

        const accountBalance = await erc20.balanceOf.call(account);
        assert.equal(accountBalance, initialValue, 'Account balance mismatch after transfer!');

        const account2Balance = await erc20.balanceOf.call(account2);
        assert.equal(account2Balance, 0, 'Account balance mismatch after transfer!');
    });

    it('cannot transfer without sufficient balance', async () => {
        let threw = false;
        try {
            await erc20.transfer(account, transferValue, {from: account2});
        } catch (e) {
            threw = true;
        }
        assert.equal(threw, true, "This should have thrown because account 2 doesn't have sufficient funds!")
    });

    it('has correct nonce after failure -- send from 1 to 2', async () => {
        try {
            await erc20.transfer(account2, transferValue, {from: account});
        } catch (e) {
            assert.fail('This transaction should not fail.')
        }
    });
});