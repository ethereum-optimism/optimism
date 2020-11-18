// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_BondManager, Errors, ERC20 } from "../../iOVM/verification/iOVM_BondManager.sol";
import { iOVM_FraudVerifier } from "../../iOVM/verification/iOVM_FraudVerifier.sol";

/**
 * @title OVM_BondManager
 */
contract OVM_BondManager is iOVM_BondManager, Lib_AddressResolver {

    /****************************
     * Constants and Parameters *
     ****************************/

    /// The dispute period
    uint256 public constant disputePeriodSeconds = 7 days;

    /// The minimum collateral a sequencer must post
    uint256 public requiredCollateral = 1 ether;

    /// The maximum multiplier for updating the `requiredCollateral`
    uint256 public constant MAX = 5;

    /// Owner used to bump the security bond size
    address immutable public owner;


    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    /// The bond token
    ERC20 immutable public token;

    /// The fraud verifier contract, used to get data about transitioners for a pre-state root
    address public ovmFraudVerifier;


    /********************************************
     * Contract Variables: Internal Accounting  *
     *******************************************/

    /// The bonds posted by each proposer
    mapping(address => Bond) public bonds;

    /// For each pre-state root, there's an array of witnessProviders that must be rewarded
    /// for posting witnesses
    mapping(bytes32 => Rewards) public witnessProviders;


    /***************
     * Constructor *
     ***************/

    /// Initializes with a ERC20 token to be used for the fidelity bonds
    /// and with the Address Manager
    constructor(ERC20 _token, address _libAddressManager)
        Lib_AddressResolver(_libAddressManager)
    {
        owner = msg.sender;
        token = _token;
    }


    /********************
     * Public Functions *
     ********************/

    /// Adds `who` to the list of witnessProviders for the provided `preStateRoot`.
    function recordGasSpent(bytes32 _preStateRoot, address who, uint256 gasSpent) override public {
        // The sender must be the transitioner that corresponds to the claimed pre-state root
        address transitioner = address(iOVM_FraudVerifier(resolve("OVM_FraudVerifier")).getStateTransitioner(_preStateRoot));
        require(transitioner == msg.sender, Errors.ONLY_TRANSITIONER);

        witnessProviders[_preStateRoot].total += gasSpent;
        witnessProviders[_preStateRoot].gasSpent[who] += gasSpent;
    }

    /// Slashes + distributes rewards or frees up the sequencer's bond, only called by
    /// `FraudVerifier.finalizeFraudVerification`
    function finalize(bytes32 _preStateRoot, uint256 batchIndex, address publisher, uint256 timestamp) override public {
        require(msg.sender == resolve("OVM_FraudVerifier"), Errors.ONLY_FRAUD_VERIFIER);
        require(witnessProviders[_preStateRoot].canClaim == false, Errors.ALREADY_FINALIZED);

        // allow users to claim from that state root's
        // pool of collateral (effectively slashing the sequencer)
        witnessProviders[_preStateRoot].canClaim = true;

        Bond storage bond = bonds[publisher];

        // if the fraud proof's dispute period does not intersect with the 
        // withdrawal's timestamp, then the user should not be slashed
        // e.g if a user at day 10 submits a withdrawal, and a fraud proof
        // from day 1 gets published, the user won't be slashed since day 8 (1d + 7d)
        // is before the user started their withdrawal. on the contrary, if the user
        // had started their withdrawal at, say, day 6, they would be slashed
        if (
            bond.withdrawalTimestamp != 0 && 
            uint256(bond.withdrawalTimestamp) > timestamp + disputePeriodSeconds &&
            bond.state == State.WITHDRAWING
        ) {
            return;
        }

        // slash!
        bond.state = State.NOT_COLLATERALIZED;
    }

    /// Sequencers call this function to post collateral which will be used for
    /// the `appendBatch` call
    function deposit(uint256 amount) override public {
        require(
            token.transferFrom(msg.sender, address(this), amount),
            Errors.ERC20_ERR
        );

        // This cannot overflow
        bonds[msg.sender].state = State.COLLATERALIZED;
    }

    /// Starts the withdrawal for a publisher
    function startWithdrawal() override public {
        Bond storage bond = bonds[msg.sender];
        require(bond.withdrawalTimestamp == 0, Errors.WITHDRAWAL_PENDING);
        require(bond.state == State.COLLATERALIZED, Errors.WRONG_STATE);

        bond.state = State.WITHDRAWING;
        bond.withdrawalTimestamp = uint32(block.timestamp);
    }

    /// Finalizes a pending withdrawal from a publisher
    function finalizeWithdrawal() override public {
        Bond storage bond = bonds[msg.sender];

        require(
            block.timestamp >= uint256(bond.withdrawalTimestamp) + disputePeriodSeconds, 
            Errors.TOO_EARLY
        );
        require(bond.state == State.WITHDRAWING, Errors.SLASHED);
        
        // refunds!
        bond.state = State.NOT_COLLATERALIZED;
        bond.withdrawalTimestamp = 0;
        
        require(
            token.transfer(msg.sender, requiredCollateral),
            Errors.ERC20_ERR
        );
    }

    /// Claims the user's reward for the witnesses they provided
    function claim(bytes32 _preStateRoot) override public {
        Rewards storage rewards = witnessProviders[_preStateRoot];

        // only allow claiming if fraud was proven in `finalize`
        require(rewards.canClaim, Errors.CANNOT_CLAIM);

        // proportional allocation - only reward 50% (rest gets locked in the
        // contract forever
        uint256 amount = (requiredCollateral * rewards.gasSpent[msg.sender]) / (2 * rewards.total);

        // reset the user's spent gas so they cannot double claim
        rewards.gasSpent[msg.sender] = 0;

        // transfer
        require(token.transfer(msg.sender, amount), Errors.ERC20_ERR);
    }

    /// Sets the required collateral for posting a state root
    /// Callable only by the contract's deployer.
    function setRequiredCollateral(uint256 newValue) override public {
        require(newValue > requiredCollateral, Errors.LOW_VALUE);
        require(newValue < MAX * requiredCollateral, Errors.HIGH_VALUE);
        require(msg.sender == owner, Errors.ONLY_OWNER);
        requiredCollateral = newValue;
    }

    /// Checks if the user is collateralized for the batchIndex
    function isCollateralized(address who) override public view returns (bool) {
        return bonds[who].state == State.COLLATERALIZED;
    }

    /// Gets how many witnesses the user has provided for the state root
    function getGasSpent(bytes32 preStateRoot, address who) override public view returns (uint256) {
        return witnessProviders[preStateRoot].gasSpent[who];
    }
}
