# MockAttestationDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/BondManager.t.sol)

**Inherits:**
[IDisputeGame](/contracts/dispute/IDisputeGame.sol/interface.IDisputeGame.md)

*A mock dispute game for testing bond seizures.*


## State Variables
### gameStatus

```solidity
GameStatus internal gameStatus;
```


### bm

```solidity
BondManager bm;
```


### rc

```solidity
Claim internal rc;
```


### ed

```solidity
bytes internal ed;
```


### bondId

```solidity
bytes32 internal bondId;
```


### challengers

```solidity
address[] internal challengers;
```


## Functions
### getChallengers


```solidity
function getChallengers() public view returns (address[] memory);
```

### setBondId


```solidity
function setBondId(bytes32 bid) external;
```

### setBondManager


```solidity
function setBondManager(BondManager _bm) external;
```

### setGameStatus


```solidity
function setGameStatus(GameStatus _gs) external;
```

### setRootClaim


```solidity
function setRootClaim(Claim _rc) external;
```

### setExtraData


```solidity
function setExtraData(bytes memory _ed) external;
```

### receive


```solidity
receive() external payable;
```

### fallback


```solidity
fallback() external payable;
```

### splitResolve


```solidity
function splitResolve() public;
```

### initialize

-------------------------------------------
Initializable Functions
-------------------------------------------


```solidity
function initialize() external;
```

### version

-------------------------------------------
IVersioned Functions
-------------------------------------------


```solidity
function version() external pure returns (string memory _version);
```

### createdAt

-------------------------------------------
IDisputeGame Functions
-------------------------------------------


```solidity
function createdAt() external pure override returns (Timestamp _createdAt);
```

### status


```solidity
function status() external view override returns (GameStatus _status);
```

### gameType


```solidity
function gameType() external pure returns (GameType _gameType);
```

### rootClaim


```solidity
function rootClaim() external view override returns (Claim _rootClaim);
```

### extraData


```solidity
function extraData() external view returns (bytes memory _extraData);
```

### bondManager


```solidity
function bondManager() external view override returns (IBondManager _bondManager);
```

### resolve


```solidity
function resolve() external returns (GameStatus _status);
```

