# Ejemplos de Contratos OP Stack
# Contribución de vaiosx.base.eth

## 🎯 Propósito
Proporcionar ejemplos claros y prácticos para el uso de contratos del OP Stack.

## 📋 Contratos Principales

### 1. OptimismPortal
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import {OptimismPortal} from "src/L1/OptimismPortal.sol";

contract OptimismPortalExample {
    OptimismPortal public portal;
    
    constructor(address _portal) {
        portal = OptimismPortal(_portal);
    }
    
    // Depositar ETH a L2
    function depositETH() external payable {
        portal.depositTransaction{value: msg.value}(
            msg.sender,    // _to
            0,             // _value
            100000,        // _gasLimit
            false,         // _isCreation
            ""             // _data
        );
    }
    
    // Depositar ERC20 a L2
    function depositERC20(
        address _l1Token,
        address _l2Token,
        uint256 _amount,
        uint32 _gasLimit
    ) external {
        // Aprobar tokens
        IERC20(_l1Token).approve(address(portal), _amount);
        
        // Depositar
        portal.depositERC20(_l1Token, _l2Token, _amount, _gasLimit, "");
    }
}
```

### 2. L2OutputOracle
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import {L2OutputOracle} from "src/L1/L2OutputOracle.sol";

contract L2OutputOracleExample {
    L2OutputOracle public oracle;
    
    constructor(address _oracle) {
        oracle = L2OutputOracle(_oracle);
    }
    
    // Obtener el último output
    function getLatestOutput() external view returns (bytes32) {
        return oracle.getL2Output(oracle.latestOutputIndex());
    }
    
    // Verificar si un output existe
    function outputExists(uint256 _l2OutputIndex) external view returns (bool) {
        try oracle.getL2Output(_l2OutputIndex) returns (bytes32) {
            return true;
        } catch {
            return false;
        }
    }
}
```

### 3. SystemConfig
```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import {SystemConfig} from "src/L1/SystemConfig.sol";

contract SystemConfigExample {
    SystemConfig public config;
    
    constructor(address _config) {
        config = SystemConfig(_config);
    }
    
    // Obtener configuración del sistema
    function getSystemConfig() external view returns (
        uint256 overhead,
        uint256 scalar,
        bytes32 batcherHash,
        uint64 gasLimit,
        address unsafeBlockSigner
    ) {
        return (
            config.overhead(),
            config.scalar(),
            config.batcherHash(),
            config.gasLimit(),
            config.unsafeBlockSigner()
        );
    }
}
```

## 🔧 Ejemplos de Uso

### 1. Depositar y Retirar
```solidity
contract DepositWithdrawExample {
    OptimismPortal public portal;
    L2ToL1MessagePasser public messagePasser;
    
    // Depositar ETH
    function depositETH(uint256 _amount) external payable {
        require(msg.value >= _amount, "Insufficient ETH");
        
        portal.depositTransaction{value: _amount}(
            msg.sender,
            0,
            100000,
            false,
            ""
        );
    }
    
    // Iniciar retiro de L2
    function initiateWithdrawal(uint256 _amount) external {
        messagePasser.initiateWithdrawal{value: _amount}();
    }
}
```

### 2. Monitoreo de Outputs
```solidity
contract OutputMonitor {
    L2OutputOracle public oracle;
    
    event OutputSubmitted(uint256 indexed index, bytes32 output);
    
    function monitorOutputs() external {
        uint256 latestIndex = oracle.latestOutputIndex();
        
        for (uint256 i = 0; i <= latestIndex; i++) {
            bytes32 output = oracle.getL2Output(i);
            emit OutputSubmitted(i, output);
        }
    }
}
```

## 🚀 Scripts de Deployment

### 1. Deploy Script
```javascript
// scripts/deploy-example.js
// Contribución de vaiosx.base.eth

const { ethers } = require("hardhat");

async function main() {
    const [deployer] = await ethers.getSigners();
    
    console.log("Deploying contracts with account:", deployer.address);
    console.log("Account balance:", (await deployer.getBalance()).toString());
    
    // Deploy OptimismPortal
    const OptimismPortal = await ethers.getContractFactory("OptimismPortal");
    const portal = await OptimismPortal.deploy();
    await portal.deployed();
    
    console.log("OptimismPortal deployed to:", portal.address);
    
    // Deploy L2OutputOracle
    const L2OutputOracle = await ethers.getContractFactory("L2OutputOracle");
    const oracle = await L2OutputOracle.deploy();
    await oracle.deployed();
    
    console.log("L2OutputOracle deployed to:", oracle.address);
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error(error);
        process.exit(1);
    });
```

### 2. Test Script
```javascript
// test/contracts-example.test.js
// Contribución de vaiosx.base.eth

const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("OP Stack Contracts", function () {
    let portal, oracle, owner, user;
    
    beforeEach(async function () {
        [owner, user] = await ethers.getSigners();
        
        // Deploy contracts
        const OptimismPortal = await ethers.getContractFactory("OptimismPortal");
        portal = await OptimismPortal.deploy();
        
        const L2OutputOracle = await ethers.getContractFactory("L2OutputOracle");
        oracle = await L2OutputOracle.deploy();
    });
    
    it("Should deploy contracts successfully", async function () {
        expect(await portal.address).to.not.equal(ethers.constants.AddressZero);
        expect(await oracle.address).to.not.equal(ethers.constants.AddressZero);
    });
    
    it("Should deposit ETH successfully", async function () {
        const amount = ethers.utils.parseEther("1.0");
        
        await expect(
            portal.depositTransaction(
                user.address,
                0,
                100000,
                false,
                "0x"
            )
        ).to.emit(portal, "TransactionDeposited");
    });
});
```

## 📚 Recursos Adicionales

- [Documentación de Contratos](https://docs.optimism.io/builders/contracts)
- [Guía de Deployment](https://docs.optimism.io/builders/contracts/deployment)
- [API Reference](https://docs.optimism.io/builders/contracts/api)

---

**Contribución de**: vaiosx.base.eth  
**Fecha**: 2025-01-13  
**Builder Rewards**: $WCT Tokens  
**Impacto**: High - Mejora significativa en ejemplos de contratos
