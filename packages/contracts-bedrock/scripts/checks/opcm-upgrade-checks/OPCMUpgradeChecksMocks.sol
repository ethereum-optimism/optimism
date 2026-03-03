// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

IUpgradeable constant UPGRADE_CONTRACT = IUpgradeable(address(111));
uint8 constant NOT_FOUND = 0;
uint8 constant UPGRADE_EXTERNAL_CALL = 1;
uint8 constant UPGRADE_INTERNAL_CALL = 2;

interface IUpgradeable {
    function upgrade() external;
    function upgradeAndCall(address _newImplementation, address _newImplementationCode, bytes memory _data) external;
}

///////// INDIRECT UPGRADE CALLS //////////

contract InternalUpgradeFunction {
    function upgradeToAndCall(IUpgradeable _a, address _b, address _c, bytes memory _d) internal {
        _a.upgradeAndCall(_b, _c, _d);
    }
}

contract WithNoExternalUpgradeFunctionInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = NOT_FOUND;

    function aaa() external {
        upgradeToAndCall(
            UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
        );
    }
}

contract CorrectInterfaceButWrongFunctionTypeInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = NOT_FOUND;

    function upgrade() external {
        upgradeToAndCall(
            UPGRADE_CONTRACT,
            address(UPGRADE_CONTRACT),
            address(0),
            abi.encodeCall(
                IUpgradeable.upgradeAndCall, (address(0), address(0), abi.encodeCall(IUpgradeable.upgrade, ()))
            )
        );
    }
}

contract WrongInterfaceButCorrectFunctionTypeInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = NOT_FOUND;

    function upgrade() external {
        upgradeToAndCall(
            UPGRADE_CONTRACT,
            address(UPGRADE_CONTRACT),
            address(0),
            abi.encodeCall(WrongInterfaceButCorrectFunctionTypeInternal.upgrade, ())
        );
    }
}

contract WithinTopLevelFunctionInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade() external {
        upgradeToAndCall(
            UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
        );
    }
}

contract WithinBlockStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade() external {
        {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        }
    }
}

contract WithinForLoopInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade() external {
        for (uint256 i = 0; i < 10; i++) {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        }
    }
}

contract WithinWhileLoopInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade() external {
        while (true) {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        }
    }
}

contract WithinDoWhileLoopInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade() external {
        do {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        } while (true);
    }
}

contract WithinTrueBlockOfIfStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        } else {
            revert();
        }
    }
}

contract WithinFalseBlockOfIfStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        }
    }
}

contract WithinElseIfBlockOfIfStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else if (_a < 20) {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        } else {
            revert();
        }
    }
}

contract WithinTrueBlockOfTernaryStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            )
            : this.mock();
    }
}

contract WithinFalseBlockOfTernaryStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? this.mock()
            : upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
    }
}

contract WithinTrueBlockOfTrueBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? _a < 5
                ? upgradeToAndCall(
                    UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
                )
                : this.mock()
            : this.mock();
    }
}

contract WithinFalseBlockOfTrueBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? _a < 5
                ? this.mock()
                : upgradeToAndCall(
                    UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
                )
            : this.mock();
    }
}

contract WithinFalseBlockOfFalseBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? this.mock()
            : _a > 5
                ? this.mock()
                : upgradeToAndCall(
                    UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
                );
    }
}

contract WithinTrueBlockOfFalseBlockOfNestedTernaryStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10
            ? this.mock()
            : _a > 5
                ? upgradeToAndCall(
                    UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
                )
                : this.mock();
    }
}

contract WithinTryBlockOfTryCatchStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade() external {
        try this.mock() {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        } catch { }
    }
}

contract WithinCatchBlockOfTryCatchStatementInternal is InternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_INTERNAL_CALL;

    function mock() external { }

    function upgrade() external {
        try this.mock() { }
        catch {
            upgradeToAndCall(
                UPGRADE_CONTRACT, address(UPGRADE_CONTRACT), address(0), abi.encodeCall(IUpgradeable.upgrade, ())
            );
        }
    }
}

///////// DIRECT UPGRADE CALLS //////////

contract WithNoExternalUpgradeFunction {
    uint8 constant EXPECTED_OUTPUT = NOT_FOUND;

    function aaa() external {
        UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinTopLevelFunction {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade() external {
        UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinBlockStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade() external {
        {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinForLoop {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade() external {
        for (uint256 i = 0; i < 10; i++) {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinWhileLoop {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade() external {
        while (true) {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinDoWhileLoop {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade() external {
        do {
            UPGRADE_CONTRACT.upgrade();
        } while (true);
    }
}

contract WithinTrueBlockOfIfStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            UPGRADE_CONTRACT.upgrade();
        } else {
            revert();
        }
    }
}

contract WithinFalseBlockOfIfStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade(uint256 _a) external {
        if (_a < 10) {
            revert();
        } else {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinElseIfBlockOfIfStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

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
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? UPGRADE_CONTRACT.upgrade() : this.mock();
    }
}

contract WithinFalseBlockOfTernaryStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinTrueBlockOfTrueBlockOfNestedTernaryStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? _a < 5 ? UPGRADE_CONTRACT.upgrade() : this.mock() : this.mock();
    }
}

contract WithinFalseBlockOfTrueBlockOfNestedTernaryStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? _a < 5 ? this.mock() : UPGRADE_CONTRACT.upgrade() : this.mock();
    }
}

contract WithinFalseBlockOfFalseBlockOfNestedTernaryStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : _a < 5 ? this.mock() : UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinTrueBlockOfFalseBlockOfNestedTernaryStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade(uint256 _a) external {
        _a < 10 ? this.mock() : _a < 5 ? UPGRADE_CONTRACT.upgrade() : this.mock();
    }
}

contract WithTryStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function upgrade() external {
        try UPGRADE_CONTRACT.upgrade() { } catch { }
    }
}

contract WithinTryBlockOfTryCatchStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade() external {
        try this.mock() {
            UPGRADE_CONTRACT.upgrade();
        } catch { }
    }
}

contract WithinCatchBlockOfTryCatchStatement {
    uint8 constant EXPECTED_OUTPUT = UPGRADE_EXTERNAL_CALL;

    function mock() external { }

    function upgrade() external {
        try this.mock() { }
        catch {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}
