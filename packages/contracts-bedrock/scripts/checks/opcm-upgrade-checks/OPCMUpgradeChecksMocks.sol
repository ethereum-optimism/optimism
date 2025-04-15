// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

IUpgradeable constant UPGRADE_CONTRACT = IUpgradeable(address(111));

interface IUpgradeable {
    function upgrade() external;
}

///////// INDIRECT UPGRADE CALLS //////////

contract InternalUpgradeFunction {
    function upgradeToAndCall(IUpgradeable _a, address _b, address _c, bytes memory _d) internal { }
}

contract WithNoExternalUpgradeFunctionInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = false;

    function aaa() external {
        upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
    }
}

contract WithinTopLevelFunctionInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
    }
}

contract WithinBlockStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        }
    }
}

contract WithinForLoopInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        for (uint256 i = 0; i < 10; i++) {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        }
    }
}

contract WithinWhileLoopInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        while (true) {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        }
    }
}

contract WithinDoWhileLoopInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        do {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        } while (true);
    }
}

contract WithinTrueBlockOfIfStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        } else {
            revert();
        }
    }
}

contract WithinFalseBlockOfIfStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        }
    }
}

contract WithinElseIfBlockOfIfStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else if (_a < 20) {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        } else {
            revert();
        }
    }
}

contract WithinTrueBlockOfTernaryStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes("")) : this.mock();
    }
}

contract WithinFalseBlockOfTernaryStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
    }
}

contract WithinTrueBlockOfTrueBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? _a < 5 ? upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes("")) : this.mock()
            : this.mock();
    }
}

contract WithinFalseBlockOfTrueBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? _a < 5 ? this.mock() : upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""))
            : this.mock();
    }
}

contract WithinFalseBlockOfFalseBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? this.mock()
            : _a > 5 ? this.mock() : upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
    }
}

contract WithinTrueBlockOfFalseBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? this.mock()
            : _a > 5 ? upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes("")) : this.mock();
    }
}

contract WithinTryBlockOfTryCatchStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade() external {
        try this.mock() {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        } catch { }
    }
}

contract WithinCatchBlockOfTryCatchStatementInternal is InternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade() external {
        try this.mock() { }
        catch {
            upgradeToAndCall(UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), bytes(""));
        }
    }
}

///////// DIRECT UPGRADE CALLS //////////

contract WithNoExternalUpgradeFunction {
    bool constant EXPECTED_OUTPUT = false;

    function aaa() external {
        UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinTopLevelFunction {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinBlockStatement {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinForLoop {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        for (uint256 i = 0; i < 10; i++) {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinWhileLoop {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        while (true) {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinDoWhileLoop {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        do {
            UPGRADE_CONTRACT.upgrade();
        } while (true);
    }
}

contract WithinTrueBlockOfIfStatement {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            UPGRADE_CONTRACT.upgrade();
        } else {
            revert();
        }
    }
}

contract WithinFalseBlockOfIfStatement {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinElseIfBlockOfIfStatement {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else if (_a < 20) {
            UPGRADE_CONTRACT.upgrade();
        } else {
            revert();
        }
    }
}

contract WithinTrueBlockOfTernaryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? UPGRADE_CONTRACT.upgrade() : this.mock();
    }
}

contract WithinFalseBlockOfTernaryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinTrueBlockOfTrueBlockOfNestedTernaryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? _a < 5 ? UPGRADE_CONTRACT.upgrade() : this.mock() : this.mock();
    }
}

contract WithinFalseBlockOfTrueBlockOfNestedTernaryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? _a < 5 ? this.mock() : UPGRADE_CONTRACT.upgrade() : this.mock();
    }
}

contract WithinFalseBlockOfFalseBlockOfNestedTernaryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : _a < 5 ? this.mock() : UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinTrueBlockOfFalseBlockOfNestedTernaryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : _a < 5 ? UPGRADE_CONTRACT.upgrade() : this.mock();
    }
}

contract WithTryStatement {
    bool constant EXPECTED_OUTPUT = true;

    function upgrade() external {
        try UPGRADE_CONTRACT.upgrade() { } catch { }
    }
}

contract WithinTryBlockOfTryCatchStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade() external {
        try this.mock() {
            UPGRADE_CONTRACT.upgrade();
        } catch { }
    }
}

contract WithinCatchBlockOfTryCatchStatement {
    bool constant EXPECTED_OUTPUT = true;

    function mock() external { }

    function upgrade() external {
        try this.mock() { }
        catch {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}
