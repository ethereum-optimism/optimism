# L2StandardTokenFactory



> L2StandardTokenFactory



*Factory contract for creating standard L2 token representations of L1 ERC20s compatible with and working on the standard bridge.*

## Methods

### createStandardL2Token

```solidity
function createStandardL2Token(address _l1Token, string _name, string _symbol) external nonpayable
```



*Creates an instance of the standard ERC20 token on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of the corresponding L1 token.
| _name | string | ERC20 name.
| _symbol | string | ERC20 symbol.



## Events

### StandardL2TokenCreated

```solidity
event StandardL2TokenCreated(address indexed _l1Token, address indexed _l2Token)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |



