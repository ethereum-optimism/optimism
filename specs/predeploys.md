# Predeploys

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [L2ToL1MessagePasser](#l2tol1messagepasser)
- [DeployerWhitelist](#deployerwhitelist)
- [OVM\_ETH](#ovm%5C_eth)
- [WETH9](#weth9)
- [L2CrossDomainMessenger](#l2crossdomainmessenger)
- [L2StandardBridge](#l2standardbridge)
- [SequencerFeeVault](#sequencerfeevault)
- [OptimismMintableERC20Factory](#optimismmintableerc20factory)
- [L1BlockNumber](#l1blocknumber)
- [GasPriceOracle](#gaspriceoracle)
- [L1Block](#l1block)
- [ProxyAdmin](#proxyadmin)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

Predeployed smart contracts exist on Optimism at predetermined addresses in
the genesis state. They are similar to precompiles but instead run directly
in the EVM instead of running native code outside of the EVM.

Predeploy addresses exist in 2 byte namespaces where the prefixes
are one of:

- `0x420000000000000000000000000000000000xxxx`
- `0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeADxxxx`

The following table includes each of the predeploys. The system version
indicates when the predeploy was introduced. The possible values are `Legacy`
or `Bedrock`. Deprecated contracts should not be used.

| Name                          | Address                                    | Introduced | Deprecated |
| ----------------------------- | ------------------------------------------ | ---------- | ---------- |
| LegacyMessagePasser           | 0x4200000000000000000000000000000000000000 | Legacy     | Yes        |
| DeployerWhitelist             | 0x4200000000000000000000000000000000000002 | Legacy     | Yes        |
| LegacyERC20ETH                | 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000 | Legacy     | Yes        |
| WETH9                         | 0x4200000000000000000000000000000000000006 | Legacy     | No         |
| L2CrossDomainMessenger        | 0x4200000000000000000000000000000000000007 | Legacy     | No         |
| L2StandardBridge              | 0x4200000000000000000000000000000000000010 | Legacy     | No         |
| SequencerFeeVault             | 0x4200000000000000000000000000000000000011 | Legacy     | No         |
| OptimismMintableERC20Factory  | 0x4200000000000000000000000000000000000012 | Legacy     | No         |
| L1BlockNumber                 | 0x4200000000000000000000000000000000000013 | Legacy     | Yes        |
| GasPriceOracle                | 0x420000000000000000000000000000000000000F | Legacy     | No         |
| GovernanceToken               | 0x4200000000000000000000000000000000000042 | Legacy     | No         |
| L1Block                       | 0x4200000000000000000000000000000000000015 | Bedrock    | No         |
| L2ToL1MessagePasser           | 0x4200000000000000000000000000000000000016 | Bedrock    | No         |
| L2ERC721Bridge                | 0x4200000000000000000000000000000000000014 | Legacy     | No         |
| OptimismMintableERC721Factory | 0x4200000000000000000000000000000000000017 | Bedrock    | No         |
| ProxyAdmin                    | 0x4200000000000000000000000000000000000018 | Bedrock    | No         |
| BaseFeeVault                  | 0x4200000000000000000000000000000000000019 | Bedrock    | No         |
| L1FeeVault                    | 0x420000000000000000000000000000000000001a | Bedrock    | No         |

## L2ToL1MessagePasser

The `OVM_L2ToL1MessagePasser` stores commitments to withdrawal transactions.
When a user is submitting the withdrawing transaction on L1, they provide a
proof that the transaction that they withdrew on L2 is in the `sentMessages`
mapping of this contract.

Any withdrawn ETH accumulates into this contract on L2 and can be
permissionlessly removed from the L2 supply by calling the `burn()` function.

The legacy interface is not preserved but included below.

```solidity
interface iLegacyOVM_L2ToL1MessagePasser {
    event L2ToL1Message(uint256 _nonce, address _sender, bytes _data);
    function sentMessages(bytes32 _msgHash) public returns (bool);
    function passMessageToL1(bytes calldata _message) external;
}
```

## DeployerWhitelist

The `DeployerWhitelist` is a predeploy used to provide additional
safety during the initial phases of Optimism. It is owned by the
Optimism team, and defines accounts which are allowed to deploy contracts to the
network.

Arbitrary contract deployment has been enabled and it is not possible to turn
off. In the legacy system, this contract was hooked into `CREATE` and
`CREATE2` to ensure that the deployer was allowlisted.

In the Bedrock system, this contract will no longer be used as part of the
`CREATE` codepath.

This contract is deprecated and its usage should be avoided.

```solidity
interface iDeployerWhitelist {
    event OwnerChanged(address,address);
    event WhitelistStatusChanged(address,bool);
    event WhitelistDisabled(address);

    function owner() public return (address);
    function setOwner(address _owner) public;

    function whitelist(address) public returns (bool);

    /**
     * @dev Adds or removes an address from the deployment whitelist.
     * @param _deployer Address to update permissions for.
     * @param _isWhitelisted Whether or not the address is whitelisted.
     */
    function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) external;

    /**
     * @dev Permanently enables arbitrary contract deployment and deletes the owner.
     */
    function enableArbitraryContractDeployment() external;

    /**
     * @dev Checks whether an address is allowed to deploy contracts.
     * @param _deployer Address to check.
     * @return _allowed Whether or not the address can deploy contracts.
     */
    function isDeployerAllowed(address _deployer) external view returns (bool);
}
```

## OVM\_ETH

The `OVM_ETH` contains the ERC20 represented balances of ETH that has been
deposited to L2. As part of the Bedrock upgrade, the balances will be migrated
from this contract to the actual Ethereum level accounts to preserve EVM
equivalence.

This contract is deprecated and its usage should be avoided.

```solidity
interface iOVM_ETH {
    event Mint(address indexed _account, uint256 _amount);
    event Burn(address indexed _account, uint256 _amount);
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    function supportsInterface(bytes4 interfaceId) external view returns (bool);
    function l1Token() external returns (address);
    function mint(address _to, uint256 _amount) external;
    function burn(address _from, uint256 _amount) external;
    function totalSupply() external view returns (uint256);
    function balanceOf(address account) external view returns (uint256);
    function transfer(address to, uint256 amount) external returns (bool);
    function allowance(address owner, address spender) external view returns (uint256);
    function approve(address spender, uint256 amount) external returns (bool);
    function transferFrom(
        address from,
        address to,
        uint256 amount
    ) external returns (bool);
}
```

## WETH9

`WETH9` is the standard implementation of Wrapped Ether on Optimism.

```solidity
interface WETH9 {
    function name() public returns (string);
    function symbol() public returns (string);
    function decimals public returns (uint8);

    event  Approval(address indexed src, address indexed guy, uint wad);
    event  Transfer(address indexed src, address indexed dst, uint wad);
    event  Deposit(address indexed dst, uint wad);
    event  Withdrawal(address indexed src, uint wad);

    function balanceOf(address) public returns (uint);
    function allowance(address, address) public returns (uint);

    function deposit() public;
    function withdraw(uint wad) public;
    function totalSupply() public view returns (uint);
    function approve(address guy, uint wad) public returns (bool);
    function transfer(address dst, uint wad) public returns (bool);

    function transferFrom(
        address src,
        address dst,
        uint wad
    ) public returns (bool);
}
```

## L2CrossDomainMessenger

The `L2CrossDomainMessenger` is used to give a better user experience when
sending cross domain messages from L2 to L1. It extends the
`CrossDomainMessenger`, which allows for replayability of messages. Any calls
through the `l1CrossDomainMessenger` go through the `L2CrossDomainMessenger`.

The `relayMessage` function executes a transaction from the remote domain while
the `sendMessage` function sends a transaction to be executed on the remote
domain through the remote domain's `relayMessage` function.

## L2StandardBridge

The `L2StandardBridge` is part of the ERC20 and ETH bridging system.
Users can send ERC20s or ETH to the `L1StandardBridge` and receive the asset on
L1 through the `L2StandardBridge`. Users can also withdraw their assets through the
`L2StandardBridge` to L1.

## SequencerFeeVault

Transaction fees accumulate in this predeploy and can be withdrawn by anybody
but only to the set `l1FeeWallet`.

```solidity
interface SequencerFeeVault {
    /**
     * @dev The minimal withdrawal amount in wei for a single withdrawal
     */
    function MIN_WITHDRAWAL_AMOUNT() public returns (uint256);

    /**
     * @dev The address on L1 that fees are withdrawn to
     */
    function l1FeeWallet() public returns (address);

    /**
     * @dev Call this to withdraw the ether held in this
     * account to the L1 fee wallet on L1.
     */
    function withdraw() public;
}
```

## OptimismMintableERC20Factory

The `OptimismMintableERC20Factory` can be used to create an ERC20 token contract
on a remote domain that maps to an ERC20 token contract on the local domain
where tokens can be deposited to the remote domain. It deploys an
`OptimismMintableERC20` which has the interface that works with the
`StandardBridge`.

## L1BlockNumber

The `L1BlockNumber` returns the last known L1 block number. This contract was
introduced in the legacy system and should be backwards compatible by calling
out to the `L1Block` contract under the hood.

```solidity
interface iOVM_L1BlockNumber {
    function getL1BlockNumber() external view returns (uint256);
}
```

## GasPriceOracle

The `GasPriceOracle` is pushed the L1 basefee and the L2 gas price by
an offchain actor. The offchain actor observes the L1 blockheaders to get the
L1 basefee as well as the gas usage on L2 to compute what the L2 gas price
should be based on a congestion control algorithm.

Its usage to be pushed the L2 gas price by an offchain actor is deprecated in
Bedrock, but it is still used to hold the `overhead`, `scalar`, and `decimals`
values which are used to compute the L1 portion of the transaction fee.

```solidity
interface GasPriceOracle {
    /**
     * @dev Returns the current gas price on L2
     */
    function gasPrice() public returns (uint256);

    /**
     * @dev Returns the latest known L1 basefee
     */
    function l1BaseFee() public returns (uint256);

    /**
     * @dev Returns the amortized cost of
     * batch submission per transaction
     */
    function overhead() public returns (uint256);

    /**
     * @dev Returns the value to scale the fee up by
     */
    function scalar() public returns (uint256);

    /**
     * @dev The number of decimals of the scalar
     */
    function decimals() public returns (uint256);

    /**
     * Allows the owner to modify the l2 gas price.
     * @param _gasPrice New l2 gas price.
     */
    function setGasPrice(uint256 _gasPrice) public;

    /**
     * Allows the owner to modify the l1 base fee.
     * @param _baseFee New l1 base fee
     */
    function setL1BaseFee(uint256 _baseFee) public;

    /**
     * Allows the owner to modify the overhead.
     * @param _overhead New overhead
     */
    function setOverhead(uint256 _overhead) public;

    /**
     * Allows the owner to modify the scalar.
     * @param _scalar New scalar
     */
    function setScalar(uint256 _scalar) public;

    /**
     * Allows the owner to modify the decimals.
     * @param _decimals New decimals
     */
    function setDecimals(uint256 _decimals) public;

    function getL1Fee(bytes memory _data) public view returns (uint256);
    function getL1GasUsed(bytes memory _data) public view returns (uint256);
}
```

## L1Block

[l1-block-predeploy]: glossary.md#l1-attributes-predeployed-contract

The [L1Block][l1-block-predeploy] was introduced in Bedrock and is responsible for
maintaining L1 context in L2. This allows for L1 state to be accessed in L2.

```solidity
interface L1Block {
    /**
     * @dev Returns the special account that can only send
     * transactions to this contract
     */
    function DEPOSITOR_ACCOUNT() public returns (address);

    /**
     * @dev Returns the latest known L1 block number
     */
    function number() public returns (uint256);

    /**
     * @dev Returns the latest known L1 timestamp
     */
    function timestamp() public returns (uint256);

    /**
     * @dev Returns the latest known L1 basefee
     */
    function basefee() public returns (uint256);

    /**
     * @dev Returns the latest known L1 transaction hash
     */
    function hash() public returns (bytes32);

    /**
     * @dev sets the latest L1 block attributes
     */
    function setL1BlockValues(
        uint64 _number,
        uint64 _timestamp,
        uint256 _basefee,
        bytes32 _hash,
        uint64 _sequenceNumber
    ) external;
}
```

## ProxyAdmin

The `ProxyAdmin` is the owner of all of the proxy contracts set at the
predeploys. It is not behind a proxy itself. The owner of the `ProxyAdmin` will
have the ability to upgrade any of the other predeploy contracts.
