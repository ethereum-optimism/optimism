pragma solidity ^0.5.0;

import {PartialStateManager} from "../PartialStateManager.sol";
import {StubSafetyChecker} from "./StubSafetyChecker.sol";

/**
 * @title DeploymentHelper
 * @notice The deployment helper deploys contracts for us to be used in testing!
 * Thank you deployment helper!
 */
contract DeploymentHelper {
    /**
     * @notice Deploys a test-ready PartialStateManager.
     */
    function partialStateManagerFactory(address _owner) public returns(address) {
        // First deploy a stub safety checker
        StubSafetyChecker stubSafetyChecker = new StubSafetyChecker();
        PartialStateManager partialStateManager = new PartialStateManager(address(stubSafetyChecker), _owner);
        return address(partialStateManager);
    }
}
