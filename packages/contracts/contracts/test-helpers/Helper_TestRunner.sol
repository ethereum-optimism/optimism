// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Logging */
import { console } from "hardhat/console.sol";

/**
 * @title Helper_TestRunner
 */
contract Helper_TestRunner {
    struct TestStep {
        string functionName;
        bytes functionData;
        bool expectedReturnStatus;
        bytes expectedReturnData;
        bool onlyValidateFlag;
    }

    function runSingleTestStep(
        TestStep memory _step
    )
        public
    {
        bytes32 namehash = keccak256(abi.encodePacked(_step.functionName));
        if (namehash == keccak256("evmRETURN")) {
            bytes memory functionData = _step.functionData;
            assembly {
                return(add(functionData, 0x20), mload(functionData))
            }
        }
        if (namehash == keccak256("evmREVERT")) {
            bytes memory functionData = _step.functionData;
            assembly {
                revert(add(functionData, 0x20), mload(functionData))
            }
        }
        if (namehash == keccak256("evmINVALID")) {
            assembly {
                invalid()
            }
        }

        (bool success, bytes memory returndata) = address(msg.sender).call(_step.functionData);

        if (success != _step.expectedReturnStatus) {
            if (success == true) {
                console.log("ERROR: Expected function to revert, but function returned successfully");
                console.log("Offending Step: %s", _step.functionName);
                console.log("Return Data:");
                console.logBytes(returndata);
                console.log("");
            } else {
                (
                    uint256 _flag,
                    uint256 _nuisanceGasLeft,
                    uint256 _ovmGasRefund,
                    bytes memory _data
                ) = _decodeRevertData(returndata);

                console.log("ERROR: Expected function to return successfully, but function reverted");
                console.log("Offending Step: %s", _step.functionName);
                console.log("Flag: %s", _flag);
                console.log("Nuisance Gas Left: %s", _nuisanceGasLeft);
                console.log("OVM Gas Refund: %s", _ovmGasRefund);
                console.log("Extra Data:");
                console.logBytes(_data);
                console.log("");
            }

            _failStep();
        }

        if (keccak256(returndata) != keccak256(_step.expectedReturnData)) {
            if (success == true) {
                console.log("ERROR: Actual return data does not match expected return data");
                console.log("Offending Step: %s", _step.functionName);
                console.log("Expected:");
                console.logBytes(_step.expectedReturnData);
                console.log("Actual:");
                console.logBytes(returndata);
                console.log("");

                _failStep();
            } else {
                (
                    uint256 _expectedFlag,
                    uint256 _expectedNuisanceGasLeft,
                    uint256 _expectedOvmGasRefund,
                    bytes memory _expectedData
                ) = _decodeRevertData(_step.expectedReturnData);

                (
                    uint256 _flag,
                    uint256 _nuisanceGasLeft,
                    uint256 _ovmGasRefund,
                    bytes memory _data
                ) = _decodeRevertData(returndata);

                if (
                    _step.onlyValidateFlag
                ) {
                    if (
                        _expectedFlag != _flag
                    ) {
                        console.log("ERROR: Actual revert flag does not match expected revert flag data");
                        console.log("Offending Step: %s", _step.functionName);
                        console.log("Expected Flag: %s", _expectedFlag);
                        console.log("Actual Flag: %s", _flag);
                        _failStep();
                    }
                } else {
                    console.log("ERROR: Actual revert flag data does not match expected revert flag data");
                    console.log("Offending Step: %s", _step.functionName);
                    console.log("Expected Flag: %s", _expectedFlag);
                    console.log("Actual Flag: %s", _flag);
                    console.log("Expected Nuisance Gas Left: %s", _expectedNuisanceGasLeft);
                    console.log("Actual Nuisance Gas Left: %s", _nuisanceGasLeft);
                    console.log("Expected OVM Gas Refund: %s", _expectedOvmGasRefund);
                    console.log("Actual OVM Gas Refund: %s", _ovmGasRefund);
                    console.log("Expected Extra Data:");
                    console.logBytes(_expectedData);
                    console.log("Actual Extra Data:");
                    console.logBytes(_data);
                    console.log("");
                    _failStep();
                }
            }
        }

        if (success == false) {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }

    function runMultipleTestSteps(
        TestStep[] memory _steps
    )
        public
    {
        for (uint256 i = 0; i < _steps.length; i++) {
            runSingleTestStep(_steps[i]);
        }
    }

    function _decodeRevertData(
        bytes memory _revertdata
    )
        internal
        pure
        returns (
            uint256 _flag,
            uint256 _nuisanceGasLeft,
            uint256 _ovmGasRefund,
            bytes memory _data
        )
    {
        if (_revertdata.length == 0) {
            return (
                0,
                0,
                0,
                bytes('')
            );
        }

        return abi.decode(_revertdata, (uint256, uint256, uint256, bytes));
    }

    function _failStep()
        internal
        pure
    {
        revert("Test step failed.");
    }
}

contract Helper_TestRunner_CREATE is Helper_TestRunner {
    constructor(
        bytes memory _bytecode,
        TestStep[] memory _steps
    )
    {
        if (_steps.length > 0) {
            runMultipleTestSteps(_steps);
        } else {
            assembly {
                return(add(_bytecode, 0x20), mload(_bytecode))
            }
        }
    }
}
