// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

import { iOVM_StateManager} from "../optimistic-ethereum/iOVM/execution/iOVM_StateManager.sol";
import { OVM_ExecutionManager} from "../optimistic-ethereum/OVM/execution/OVM_ExecutionManager.sol";

contract Helper_ExecutionManager is OVM_ExecutionManager {
    constructor(address _libAddressManager, GasMeterConfig memory _gasMeterConfig, GlobalContext memory _globalContext)
    OVM_ExecutionManager(_libAddressManager, _gasMeterConfig, _globalContext)
    {
    }

    function ovmCALLHelper(uint256 _gasLimit, address _address, bytes memory _calldata, iOVM_StateManager _ovmStateManager)
    external returns (bool _success, bytes memory _returndata)
    {
        ovmStateManager = _ovmStateManager;
        return ovmCALL(_gasLimit, _address, _calldata);
    }

    function returnData(bytes memory _returnData) public pure returns (bytes memory) {
        return _returnData;
    }
}
