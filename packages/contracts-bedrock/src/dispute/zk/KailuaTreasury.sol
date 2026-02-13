// Copyright 2024, 2025 RISC Zero, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import "./KailuaLib.sol";
import "./KailuaTournament.sol";
import "./KailuaVerifier.sol";
import "interfaces/dispute/IInitializable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

contract KailuaTreasury is KailuaTournament {
    // ------------------------------
    // Immutable configuration
    // ------------------------------

    /// @notice The initial root claim for the deployment
    Claim public immutable ROOT_CLAIM;

    /// @notice The L2 block number of the initial root claim for the deployment
    uint64 public immutable L2_BLOCK_NUMBER;

    constructor(
        KailuaVerifier _kailuaVerifier,
        uint64 _proposalOutputCount,
        uint64 _outputBlockSpan,
        GameType _gameType,
        IOptimismPortal2 _optimismPortal,
        Claim _rootClaim,
        uint64 _l2SequenceNumber
    )
        KailuaTournament(
            IKailuaTreasury(address(this)),
            _kailuaVerifier,
            _proposalOutputCount,
            _outputBlockSpan,
            _gameType,
            _optimismPortal
        )
    {
        ROOT_CLAIM = _rootClaim;
        L2_BLOCK_NUMBER = _l2SequenceNumber;
    }

    // ------------------------------
    // IInitializable implementation
    // ------------------------------

    /// @inheritdoc IInitializable
    function initialize() external payable override {
        super.initializeInternal();

        // Revert if the calldata size is not the expected length.
        //
        // This is to prevent adding extra or omitting bytes from to `extraData` that result in a different game UUID
        // in the factory, but are not used by the game, which would allow for multiple dispute games for the same
        // output proposal to be created.
        //
        // Expected length: 0x76
        // - 0x04 selector                      0x00 0x04
        // - 0x14 creator address               0x04 0x18
        // - 0x20 root claim                    0x18 0x38
        // - 0x20 l1 head                       0x38 0x58
        // - 0x1c extraData:                    0x58 0x74
        //      + 0x08 l2SequenceNumber            0x58 0x60
        //      + 0x14 kailuaTreasuryAddress    0x60 0x74
        // - 0x02 CWIA bytes                    0x74 0x76
        if (msg.data.length != 0x76) {
            revert BadExtraData();
        }

        // Accept only the initialized root claim
        if (rootClaim().raw() != ROOT_CLAIM.raw()) {
            revert UnexpectedRootClaim(rootClaim());
        }

        // Accept only the initialized l2 block number
        if (l2SequenceNumber() != L2_BLOCK_NUMBER) {
            revert BlockNumberMismatch(l2SequenceNumber(), L2_BLOCK_NUMBER);
        }

        // Accept only the address of the deployment treasury
        if (treasuryAddress() != address(KAILUA_TREASURY)) {
            revert BadExtraData();
        }
    }

    /// @notice Returns the treasury address used in initialization
    function treasuryAddress() public pure returns (address treasuryAddress_) {
        treasuryAddress_ = _getArgAddress(0x5c);
    }

    // ------------------------------
    // IDisputeGame implementation
    // ------------------------------

    /// @inheritdoc IDisputeGame
    function extraData() external pure returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata and
        // is 32 bytes long.
        extraData_ = _getArgBytes(0x54, 0x1c);
    }

    /// @inheritdoc IDisputeGame
    function resolve() external onlyFactoryOwner returns (GameStatus status_) {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // Update the status and emit the resolved event, note that we're performing a storage update here.
        emit Resolved(status = status_ = GameStatus.DEFENDER_WINS);

        // Mark resolution timestamp
        resolvedAt = Timestamp.wrap(uint64(block.timestamp));

        // Update lastResolved
        KAILUA_TREASURY.updateLastResolved();
    }

    // ------------------------------
    // Fault proving
    // ------------------------------

    /// @inheritdoc KailuaTournament
    function verifyIntermediateOutput(uint64, uint256, bytes calldata, bytes calldata)
        external
        pure
        override
        returns (bool success)
    {
        // No known blobs to reference
    }

    /// @inheritdoc KailuaTournament
    function getChallengerDuration(uint256) public pure override returns (Duration duration_) {
        // No challenge period
    }

    /// @inheritdoc KailuaTournament
    function minCreationTime() public view override returns (Timestamp minCreationTime_) {
        minCreationTime_ = createdAt;
    }

    /// @inheritdoc KailuaTournament
    function parentGame() public view override returns (KailuaTournament parentGame_) {
        parentGame_ = this;
    }

    // ------------------------------
    // IKailuaTreasury implementation
    // ------------------------------

    /// @notice Returns the game index at which proposer was proven faulty
    mapping(address => uint256) public eliminationRound;

    /// @notice Returns the proposer of a game
    mapping(address => address) public proposerOf;

    /// @notice Eliminates a child's proposer and allocates their bond to the prover
    function eliminate(address _child, address prover) external {
        KailuaTournament child = KailuaTournament(_child);

        // INVARIANT: Only the child's parent may call this
        KailuaTournament parent = child.parentGame();
        if (msg.sender != address(parent)) {
            revert Blacklisted(msg.sender, address(parent));
        }

        // INVARIANT: Only known proposals may be eliminated
        address eliminated = proposerOf[address(child)];
        if (eliminated == address(0x0)) {
            revert NotProposed();
        }

        // INVARIANT: Cannot double-eliminate players
        if (eliminationRound[eliminated] > 0) {
            revert AlreadyEliminated();
        }

        // Record elimination round
        eliminationRound[eliminated] = child.gameIndex();

        uint256 bond = paidBonds[eliminated];
        paidBonds[eliminated] = 0;

        // Split the slashed bond into prover / winner / burn.
        uint256 proverShare = (bond * ELIMINATION_SPLIT_PROVER_NUM) / ELIMINATION_SPLIT_DENOM;
        uint256 winnerShare = (bond * ELIMINATION_SPLIT_WINNER_NUM) / ELIMINATION_SPLIT_DENOM;
        uint256 burnShare = bond - proverShare - winnerShare;

        eliminationRewards[prover] += proverShare;
        winnerSharesByParent[parent] += winnerShare;
        // Burn by sending it to the zero address.
        // The zero address has no code, so this external call cannot reenter.
        KailuaPayLib.pay(burnShare, address(0));
    }

    /// @notice Returns true iff a proposal is currently being submitted
    bool public isProposing;

    /// @notice Returns the last resolved proposal contract address
    address public lastResolved;

    /// @notice Updates the last resolved contract address to that of the caller
    function updateLastResolved() external {
        address proposer = proposerOf[msg.sender];

        // INVARIANT: Only known proposal contracts may call this function
        if (proposer == address(0x0)) {
            revert NotProposed();
        }

        KailuaTournament parent = KailuaTournament(msg.sender).parentGame();
        eliminationRewards[proposer] += winnerSharesByParent[parent];
        winnerSharesByParent[parent] = 0;

        lastResolved = msg.sender;
    }

    // ------------------------------
    // Treasury
    // ------------------------------

    /// @notice Fixed split of a slashed participation bond between prover, winner, and burn.
    uint256 public constant ELIMINATION_SPLIT_DENOM = 3;
    uint256 public constant ELIMINATION_SPLIT_PROVER_NUM = 1;
    uint256 public constant ELIMINATION_SPLIT_WINNER_NUM = 1;

    /// @notice The locked collateral required for proposal submission
    uint256 public participationBond;

    /// @notice The locked collateral still paid by proposers for participation
    mapping(address => uint256) public paidBonds;

    /// @notice The total share of elimination bonds accumulated for the eventual tournament winner.
    /// @dev Keyed by the parent game (tournament) contract.
    mapping(KailuaTournament => uint256) private winnerSharesByParent;

    /// @notice The unpaid rewards from eliminated invalid proposals
    mapping(address => uint256) public eliminationRewards;

    /// @notice The last proposal made by each proposer
    mapping(address => KailuaTournament) public lastProposal;

    /// @notice The leading proposer that can extend the proposal tree
    address public vanguard;

    /// @notice The duration for which the vanguard may lead
    Duration public vanguardAdvantage;

    /// @notice Boolean flag to prevent re-entrant calls
    bool internal isLocked;

    modifier nonReentrant() {
        require(!isLocked);
        isLocked = true;
        _;
        isLocked = false;
    }

    modifier onlyFactoryOwner() {
        OwnableUpgradeable factoryContract = OwnableUpgradeable(address(DISPUTE_GAME_FACTORY));
        if (msg.sender != factoryContract.owner()) revert NotFactoryOwner();
        _;
    }

    /// @notice Pays the elimination rewards the sender has accrued
    function claimEliminationRewards() public nonReentrant {
        uint256 payout = eliminationRewards[msg.sender];
        eliminationRewards[msg.sender] = 0;

        if (payout > 0) {
            KailuaPayLib.pay(payout, msg.sender);
        }
    }

    /// @notice Pays the proposer back its bond
    function claimProposerBond() public nonReentrant {
        // INVARIANT: Can only claim back bond if not eliminated
        if (eliminationRound[msg.sender] != 0) {
            revert AlreadyEliminated();
        }

        // INVARIANT: Can only claim bond back if no pending proposals are left
        KailuaTournament previousGame = lastProposal[msg.sender];
        if (address(previousGame) != address(0x0)) {
            KailuaTournament lastTournament = previousGame.parentGame();
            if (lastTournament.children(lastTournament.contenderIndex()).status() != GameStatus.DEFENDER_WINS) {
                revert GameNotResolved();
            }
        }

        uint256 payout = paidBonds[msg.sender];
        // INVARIANT: Can only claim bond if it is paid
        if (payout == 0) {
            revert NoCreditToClaim();
        }

        // Pay out and clear bond
        paidBonds[msg.sender] = 0;
        KailuaPayLib.pay(payout, msg.sender);
    }

    /// @notice Updates the required bond for new proposals
    function setParticipationBond(uint256 amount) external onlyFactoryOwner {
        participationBond = amount;
        emit BondUpdated(amount);
    }

    /// @notice Updates the vanguard address and advantage duration
    function assignVanguard(address _vanguard, Duration _vanguardAdvantage) external onlyFactoryOwner {
        vanguard = _vanguard;
        vanguardAdvantage = _vanguardAdvantage;
    }

    /// @notice Checks the proposer's bonded amount and creates a new proposal through the factory
    function propose(Claim _rootClaim, bytes calldata _extraData)
        external
        payable
        returns (KailuaTournament tournament)
    {
        // Check proposer honesty
        if (eliminationRound[msg.sender] > 0) {
            revert BadAuth();
        }
        // Update proposer bond
        if (msg.value > 0) {
            paidBonds[msg.sender] += msg.value;
        }
        // Check proposer bond
        if (paidBonds[msg.sender] < participationBond) {
            revert IncorrectBondAmount();
        }
        // Create proposal
        isProposing = true;
        tournament = KailuaTournament(address(DISPUTE_GAME_FACTORY.create(GAME_TYPE, _rootClaim, _extraData)));
        isProposing = false;
        // Check proposal progression
        KailuaTournament previousGame = lastProposal[msg.sender];
        if (address(previousGame) != address(0x0)) {
            // INVARIANT: Proposers may only extend the proposal set incrementally
            if (previousGame.l2SequenceNumber() >= tournament.l2SequenceNumber()) {
                revert BlockNumberMismatch(previousGame.l2SequenceNumber(), tournament.l2SequenceNumber());
            }
        }
        // Check whether the proposer must follow a vanguard if one is set
        if (vanguard != address(0x0) && vanguard != msg.sender) {
            // The proposer may only counter the vanguard during the advantage time
            KailuaTournament proposalParent = tournament.parentGame();
            if (proposalParent.childCount() == 1) {
                // Count the advantage clock since proposal was possible
                uint64 elapsedAdvantage = uint64(block.timestamp - tournament.minCreationTime().raw());
                if (elapsedAdvantage < vanguardAdvantage.raw()) {
                    revert VanguardError(address(proposalParent));
                }
            }
        }
        // Record proposer
        proposerOf[address(tournament)] = msg.sender;
        // Record proposal
        lastProposal[msg.sender] = tournament;
    }
}
