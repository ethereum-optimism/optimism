pragma solidity ^0.5.16;

import "../../../contracts/Governance/Comp.sol";

contract CompCertora is Comp {
    constructor(address grantor) Comp(grantor) public {}

    function certoraOrdered(address account) external view returns (bool) {
        uint32 nCheckpoints = numCheckpoints[account];
        for (uint32 i = 1; i < nCheckpoints; i++) {
            if (checkpoints[account][i - 1].fromBlock >= checkpoints[account][i].fromBlock) {
                return false;
            }
        }

        // make sure the checkpoints are also all before the current block
        if (nCheckpoints > 0 && checkpoints[account][nCheckpoints - 1].fromBlock > block.number) {
            return false;
        }

        return true;
    }

    function certoraScan(address account, uint blockNumber) external view returns (uint) {
        // find most recent checkpoint from before blockNumber
        for (uint32 i = numCheckpoints[account]; i != 0; i--) {
            Checkpoint memory cp = checkpoints[account][i-1];
            if (cp.fromBlock <= blockNumber) {
                return cp.votes;
            }
        }

        // blockNumber is from before first checkpoint (or list is empty)
        return 0;
    }
}
