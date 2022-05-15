# Standard Bridges

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Token Depositing](#token-depositing)
- [Upgradability](#upgradability)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The standard bridges are responsible for allowing cross domain
ETH and ERC20 token transfers. They are built on top of the cross domain
messenger contracts and give a standard interface for depositing tokens.

The bridge works for both L1 native tokens and L2 native tokens. The legacy API
is preserved to ensure that existing applications will not experience any
problems with the Bedrock `StandardBridge` contracts.

The `L2StandardBridge` is a predeploy contract located at
`0x4200000000000000000000000000000000000010`.

```solidity
interface StandardBridge {
    event ERC20BridgeFinalized(address indexed _localToken, address indexed _remoteToken, address indexed _from, address _to, uint256 _amount, bytes _data);
    event ERC20BridgeInitiated(address indexed _localToken, address indexed _remoteToken, address indexed _from, address _to, uint256 _amount, bytes _data);
    event ETHBridgeFinalized(address indexed _from, address indexed _to, uint256 _amount, bytes _data);
    event ETHBridgeInitiated(address indexed _from, address indexed _to, uint256 _amount, bytes _data);

    function bridgeERC20(address _localToken, address _remoteToken, uint256 _amount, uint32 _minGasLimit, bytes memory _data) external;
    function bridgeERC20To(address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes memory _data) external;
    function bridgeETH(uint32 _minGasLimit, bytes memory _data) payable external;
    function bridgeETHTo(address _to, uint32 _minGasLimit, bytes memory _data) payable external;
    function deposits(address, address) view external returns (uint256);
    function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes memory _data) external;
    function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes memory _data) payable external;
    function messenger() view external returns (address);
    function otherBridge() view external returns (address);
}
```

## Token Depositing

The `bridgeERC20` function is used to send a token from one domain to another
domain. An `OptimismMintableERC20` token contract must exist on the remote
domain to be able to deposit tokens to that domain. One of these tokens can be
deployed using the `OptimismMintableTokenFactory` contract.

## Upgradability

Both the L1 and L2 standard bridges should be behind upgradable proxies.
