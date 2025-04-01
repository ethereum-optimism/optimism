// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

IUpgradeable constant UPGRADE_CONTRACT = IUpgradeable(address(111));

interface IUpgradeable {
    function upgrade() external;
}

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
