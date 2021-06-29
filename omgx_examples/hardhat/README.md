# OMGX First Steps

## 1. To deploy on a LOCAL system

First, spin up a local system as usual:

```bash
$ cd optimism
$ yarn clean
$ yarn
$ yarn build
$ cd ops
```
Make sure you have the docker app running!


```
$ docker-compose down -v
$ docker-compose build
$ docker-compose up
```

Second, run:

```bash
$ yarn compile
$ yarn compile:ovm
```

This will compile the contracts

```
yarn run v1.22.10
$ hardhat compile
Compiling 7 files with 0.7.6
contracts/L2DepositedERC20.sol:44:9: Warning: Unused function parameter. Remove or comment out the variable name to silence this warning.
        address _to,
        ^---------^

Compilation finished successfully
✨  Done in 3.58s.
```

```
yarn run v1.22.10
$ hardhat compile --network optimism
Compiling 7 files with 0.7.6
contracts/L2DepositedERC20.sol:44:9: Warning: Unused function parameter. Remove or comment out the variable name to silence this warning.
        address _to,
        ^---------^

Compilation finished successfully
✨  Done in 6.26s.

```

Now, see if it works

```bash
$ yarn test:integration:ovm
```

```bash
yarn run v1.22.10
$ hardhat test --network optimism


  L1 <> L2 Deposit and Withdrawal
    Initialization and initial balances
      ✓ should initialize L2 ERC20 (128ms)
      ✓ should have initial L1 balance of 1234 and initial L2 balance of 0 (44ms)
    L1 to L2 deposit
      ✓ should approve 1234 tokens for ERC20 gateway (150ms)
      ✓ should deposit 1234 tokens into L2 ERC20 (1747ms)
      ✓ should relay deposit message to L2 (4081ms)
      ✓ should have changed L1 balance to 0 and L2 balance to 1234 (38ms)
    L2 to L1 withdrawal
      ✓ should withdraw tokens back to L1 ERC20 and relay the message (273ms)
      ✓ should relay withdrawal message to L1 (8074ms)
      ✓ should have changed L1 balance back to 1234 and L2 balance back to 0


  9 passing (15s)

✨  Done in 17.82s.
```

Finally, deploy the contract, 

```
$ yarn deploy:ovm
```

```bash
yarn run v1.22.10
$ hardhat deploy --network optimism
Nothing to compile
deploying "ERC20" (tx: 0x87d1425bdc08cb41967f96d9b5f85bdb2f24dc430c4d73d64349e1df4ffa3a9d)...: deployed at 0x9A9f2CCfdE556A7E9Ff0848998Aa4a0CFD8863AE with 1982889 gas
✨  Done in 1.37s.
```

## 2. To deploy on RINKEBY

First, run:

```bash
$ yarn compile:omgx
```

This will compile the contract, Now, see if it works

```bash
$ yarn test:integration:omgx
```

**Note that two tests will fail, which reflects how we set the fees on rinkeby.omgx.network.**

```bash
$ hardhat test --network omgx_rinkeby


  ERC20
    ✓ should have a name (1420ms)
    ✓ should have a total supply equal to the initial supply (2379ms)
    ✓ should give the initial supply to the creator's address (1417ms)
    transfer(...)
      1) should revert when the sender does not have enough balance
      ✓ should succeed when the sender has enough balance (5637ms)
    transferFrom(...)
      2) should revert when the sender does not have enough of an allowance
      ✓ should succeed when the owner has enough balance and the sender has a large enough allowance (9318ms)


  5 passing (47s)
  2 failing

  1) ERC20
       transfer(...)
         should revert when the sender does not have enough balance:

      AssertionError: Expected transaction to be reverted
      + expected - actual

      -Transaction NOT reverted.
      +Transaction reverted.
      
  

  2) ERC20
       transferFrom(...)
         should revert when the sender does not have enough of an allowance:

      AssertionError: Expected transaction to be reverted
      + expected - actual

      -Transaction NOT reverted.
      +Transaction reverted.

error Command failed with exit code 2.
info Visit https://yarnpkg.com/en/docs/cli/run for documentation about this command.
```

Finally, deploy the contract, 

```
$ yarn deploy:omgx
```

```bash
yarn run v1.22.10
$ hardhat deploy --network omgx_rinkeby
Nothing to compile
deploying "ERC20" (tx: 0xe0eb90b6edca6a6d5a9c965dd59afd45ef26d36c292e031c49e4cbcca0fa3feb)...: deployed at 0x4A679253410272dd5232B3Ff7cF5dbB88f295319 with 1902037 gas
✨  Done in 5.48s.
```