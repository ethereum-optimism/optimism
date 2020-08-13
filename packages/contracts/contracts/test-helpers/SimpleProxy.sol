pragma solidity ^0.5.0;

import { ExecutionManager } from "../optimistic-ethereum/ovm/ExecutionManager.sol";

/**
 * @title SimpleProxy
 * @notice A simple contract which sends a call to an arbitrary address with arbitrary calldata, forwarding return/error data.
 */
contract SimpleProxy {
    function callContractWithData(
        address _target,
        bytes memory _data
    ) public {
        (bool success,) = _target.call(_data);
        assembly {
            let retsize := returndatasize()
            returndatacopy(0, 0, retsize)
            if success {
                return(0, retsize)
            }
            revert(0, retsize)
        }
    }
}
