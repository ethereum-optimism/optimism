// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

IUpgradeable constant UPGRADE_CONTRACT = IUpgradeable(address(111));

interface IUpgradeable {
    function upgrade() external;
}

contract WithinTopLevelFunction {
    function upgrade() external {
        UPGRADE_CONTRACT.upgrade();
    }
}

contract WithinBlockStatement {
    function upgrade() external {
        {
            UPGRADE_CONTRACT.upgrade();
        }
    }
    function aaa() external {

    }
}

contract WithinForLoop {
    function upgrade() external {
        for (uint256 i = 0; i < 10; i++) {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinWhileLoop {
    function upgrade() external {
        while (true) {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinDoWhileLoop {
    function upgrade() external {
        do {
            UPGRADE_CONTRACT.upgrade();
        } while (true);
    }
}

contract WithinTrueBlockOfIfStatement {
    function upgrade(uint256 a) external {
        if (a < 10) {
            UPGRADE_CONTRACT.upgrade();
        } else {
            revert("Not allowed");
        }
    }
}

contract WithinFalseBlockOfIfStatement {
    function upgrade(uint256 a) external {
        if (a < 10) {
            revert("Not allowed");
        } else {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}

contract WithinElseIfBlockOfIfStatement {
    function upgrade(uint256 a) external {
        if (a < 10) {
            revert("Not allowed");
        } else if (a < 20) {
            UPGRADE_CONTRACT.upgrade();
        } else {
            revert("Not allowed");
        }
    }
}

contract WithinTrueBlockOfTernaryStatement {
    function upgrade(uint256 a) external {
        a < 10 ? UPGRADE_CONTRACT.upgrade() : revert("Not allowed");
    }
}

contract WithinFalseBlockOfTernaryStatement {
    function upgrade(uint256 a) external {
        a < 10 ? revert("Not allowed") : UPGRADE_CONTRACT.upgrade();
    }
}

contract WithTryStatement {
    function upgrade() external {
        try UPGRADE_CONTRACT.upgrade() {} catch {}
    }
}

contract WithinTryBlockOfTryCatchStatement {
    function mock() external {}
    function upgrade() external {
        try this.mock() {
            UPGRADE_CONTRACT.upgrade();
        } catch {}
    }
}

contract WithinCatchBlockOfTryCatchStatement {
    function mock() external {}
    function upgrade() external {
        try this.mock() {} catch {
            UPGRADE_CONTRACT.upgrade();
        }
    }
}
