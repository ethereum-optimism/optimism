// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
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

package types

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// EventClaimsClaimed is emitted when an account claims some DOTs
type EventClaimsClaimed struct {
	Phase           Phase
	Who             AccountID
	EthereumAddress H160
	Amount          U128
	Topics          []Hash
}

// EventBalancesEndowed is emitted when an account is created with some free balance
type EventBalancesEndowed struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventDustLost is emitted when an account is removed with a balance that is
// non-zero but below ExistentialDeposit, resulting in a loss.
type EventBalancesDustLost struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventBalancesTransfer is emitted when a transfer succeeded (from, to, value)
type EventBalancesTransfer struct {
	Phase  Phase
	From   AccountID
	To     AccountID
	Value  U128
	Topics []Hash
}

// EventBalanceSet is emitted when a balance is set by root
type EventBalancesBalanceSet struct {
	Phase    Phase
	Who      AccountID
	Free     U128
	Reserved U128
	Topics   []Hash
}

// EventDeposit is emitted when an account receives some free balance
type EventBalancesDeposit struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventBalancesReserved is emitted when some balance was reserved (moved from free to reserved)
type EventBalancesReserved struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventBalancesUnreserved is emitted when some balance was unreserved (moved from reserved to free)
type EventBalancesUnreserved struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventBalancesReserveRepatriated is emitted when some balance was moved from the reserve of the first account to the
// second account.
type EventBalancesReserveRepatriated struct {
	Phase             Phase
	From              AccountID
	To                AccountID
	Balance           U128
	DestinationStatus BalanceStatus
	Topics            []Hash
}

// EventBalancesWithdraw is emitted when some amount was withdrawn from the account (e.g. for transaction fees)
type EventBalancesWithdraw struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventBalancesSlashed is emitted when some amount was removed from the account (e.g. for misbehavior)
type EventBalancesSlashed struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventGrandpaNewAuthorities is emitted when a new authority set has been applied
type EventGrandpaNewAuthorities struct {
	Phase          Phase
	NewAuthorities []struct {
		AuthorityID     AuthorityID
		AuthorityWeight U64
	}
	Topics []Hash
}

// EventGrandpaPaused is emitted when the current authority set has been paused
type EventGrandpaPaused struct {
	Phase  Phase
	Topics []Hash
}

// EventGrandpaResumed is emitted when the current authority set has been resumed
type EventGrandpaResumed struct {
	Phase  Phase
	Topics []Hash
}

// EventHRMPOpenChannelRequested is emitted when an open HRMP channel is requested.
type EventHRMPOpenChannelRequested struct {
	Phase                  Phase
	Sender                 ParachainID
	Recipient              ParachainID
	ProposedMaxCapacity    U32
	ProposedMaxMessageSize U32
	Topics                 []Hash
}

// EventHRMPOpenChannelCanceled is emitted when an HRMP channel request
// sent by the receiver was canceled by either party.
type EventHRMPOpenChannelCanceled struct {
	Phase       Phase
	ByParachain ParachainID
	ChannelID   HRMPChannelID
	Topics      []Hash
}

// EventHRMPOpenChannelAccepted is emitted when an open HRMP channel is accepted.
type EventHRMPOpenChannelAccepted struct {
	Phase     Phase
	Sender    ParachainID
	Recipient ParachainID
	Topics    []Hash
}

// EventHRMPChannelClosed is emitted when an HRMP channel is closed.
type EventHRMPChannelClosed struct {
	Phase       Phase
	ByParachain ParachainID
	ChannelID   HRMPChannelID
	Topics      []Hash
}

// EventImOnlineHeartbeatReceived is emitted when a new heartbeat was received from AuthorityId
type EventImOnlineHeartbeatReceived struct {
	Phase       Phase
	AuthorityID AuthorityID
	Topics      []Hash
}

// EventImOnlineAllGood is emitted when at the end of the session, no offence was committed
type EventImOnlineAllGood struct {
	Phase  Phase
	Topics []Hash
}

// Exposure lists the own and nominated stake of a validator
type Exposure struct {
	Total  UCompact
	Own    UCompact
	Others []IndividualExposure
}

// IndividualExposure contains the nominated stake by one specific third party
type IndividualExposure struct {
	Who   AccountID
	Value UCompact
}

// EventImOnlineSomeOffline is emitted when the end of the session, at least once validator was found to be offline
type EventImOnlineSomeOffline struct {
	Phase                Phase
	IdentificationTuples []struct {
		ValidatorID        AccountID
		FullIdentification Exposure
	}
	Topics []Hash
}

// EventIndicesIndexAssigned is emitted when an index is assigned to an AccountID.
type EventIndicesIndexAssigned struct {
	Phase        Phase
	AccountID    AccountID
	AccountIndex AccountIndex
	Topics       []Hash
}

// EventIndicesIndexFreed is emitted when an index is unassigned.
type EventIndicesIndexFreed struct {
	Phase        Phase
	AccountIndex AccountIndex
	Topics       []Hash
}

// EventIndicesIndexFrozen is emitted when an index is frozen to its current account ID.
type EventIndicesIndexFrozen struct {
	Phase        Phase
	AccountIndex AccountIndex
	AccountID    AccountID
	Topics       []Hash
}

// EventLotteryLotteryStarted is emitted when a lottery has been started.
type EventLotteryLotteryStarted struct {
	Phase  Phase
	Topics []Hash
}

// EventLotteryCallsUpdated is emitted when a new set of calls has been set.
type EventLotteryCallsUpdated struct {
	Phase  Phase
	Topics []Hash
}

// EventLotteryWinner is emitted when a winner has been chosen.
type EventLotteryWinner struct {
	Phase          Phase
	Winner         AccountID
	LotteryBalance U128
	Topics         []Hash
}

// EventLotteryTicketBought is emitted when a ticket has been bought.
type EventLotteryTicketBought struct {
	Phase     Phase
	Who       AccountID
	CallIndex LotteryCallIndex
	Topics    []Hash
}

// EventOffencesOffence is emitted when there is an offence reported of the given kind happened at the session_index
// and (kind-specific) time slot. This event is not deposited for duplicate slashes
type EventOffencesOffence struct {
	Phase          Phase
	Kind           Bytes16
	OpaqueTimeSlot Bytes
	Topics         []Hash
}

// EventParasCurrentCodeUpdated is emitted when the current code has been updated for a Para.
type EventParasCurrentCodeUpdated struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasCurrentHeadUpdated is emitted when the current head has been updated for a Para.
type EventParasCurrentHeadUpdated struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasCodeUpgradeScheduled is emitted when a code upgrade has been scheduled for a Para.
type EventParasCodeUpgradeScheduled struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasNewHeadNoted is emitted when a new head has been noted for a Para.
type EventParasNewHeadNoted struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasActionQueued is emitted when a para has been queued to execute pending actions.
type EventParasActionQueued struct {
	Phase        Phase
	ParachainID  ParachainID
	SessionIndex U32
	Topics       []Hash
}

// EventParasPvfCheckStarted is emitted when the given para either initiated or subscribed to a PVF
// check for the given validation code.
type EventParasPvfCheckStarted struct {
	Phase       Phase
	CodeHash    Hash
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasPvfCheckAccepted is emitted when the given validation code was accepted by the PVF pre-checking vote.
type EventParasPvfCheckAccepted struct {
	Phase       Phase
	CodeHash    Hash
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasPvfCheckRejected is emitted when the given validation code was rejected by the PVF pre-checking vote.
type EventParasPvfCheckRejected struct {
	Phase       Phase
	CodeHash    Hash
	ParachainID ParachainID
	Topics      []Hash
}

// EventParasDisputesDisputeInitiated is emitted when a dispute has been initiated.
type EventParasDisputesDisputeInitiated struct {
	Phase           Phase
	CandidateHash   Hash
	DisputeLocation DisputeLocation
	Topics          []Hash
}

// EventParasDisputesDisputeConcluded is emitted when a dispute has concluded for or against a candidate.
type EventParasDisputesDisputeConcluded struct {
	Phase           Phase
	CandidateHash   Hash
	DisputeLocation DisputeResult
	Topics          []Hash
}

// EventParasDisputesDisputeTimedOut is emitted when a dispute has timed out due to insufficient participation.
type EventParasDisputesDisputeTimedOut struct {
	Phase         Phase
	CandidateHash Hash
	Topics        []Hash
}

// EventParasDisputesRevert is emitted when a dispute has concluded with supermajority against a candidate.
// Block authors should no longer build on top of this head and should
// instead revert the block at the given height. This should be the
// number of the child of the last known valid block in the chain.
type EventParasDisputesRevert struct {
	Phase       Phase
	BlockNumber U32
	Topics      []Hash
}

type HeadData []U8

type CoreIndex U32

type GroupIndex U32

// EventParaInclusionCandidateBacked is emitted when a candidate was backed.
type EventParaInclusionCandidateBacked struct {
	Phase            Phase
	CandidateReceipt CandidateReceipt
	HeadData         HeadData
	CoreIndex        CoreIndex
	GroupIndex       GroupIndex
	Topics           []Hash
}

// EventParaInclusionCandidateIncluded is emitted when a candidate was included.
type EventParaInclusionCandidateIncluded struct {
	Phase            Phase
	CandidateReceipt CandidateReceipt
	HeadData         HeadData
	CoreIndex        CoreIndex
	GroupIndex       GroupIndex
	Topics           []Hash
}

// EventParaInclusionCandidateTimedOut is emitted when a candidate timed out.
type EventParaInclusionCandidateTimedOut struct {
	Phase            Phase
	CandidateReceipt CandidateReceipt
	HeadData         HeadData
	CoreIndex        CoreIndex
	Topics           []Hash
}

// EventParachainSystemValidationFunctionStored is emitted when the validation function has been scheduled to apply.
type EventParachainSystemValidationFunctionStored struct {
	Phase  Phase
	Topics []Hash
}

// EventParachainSystemValidationFunctionApplied is emitted when the validation function was applied
// as of the contained relay chain block number.
type EventParachainSystemValidationFunctionApplied struct {
	Phase                 Phase
	RelayChainBlockNumber U32
	Topics                []Hash
}

// EventParachainSystemValidationFunctionDiscarded is emitted when the relay-chain aborted the upgrade process.
type EventParachainSystemValidationFunctionDiscarded struct {
	Phase  Phase
	Topics []Hash
}

// EventParachainSystemUpgradeAuthorized is emitted when an upgrade has been authorized.
type EventParachainSystemUpgradeAuthorized struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventParachainSystemDownwardMessagesReceived is emitted when some downward messages
// have been received and will be processed.
type EventParachainSystemDownwardMessagesReceived struct {
	Phase  Phase
	Count  U32
	Topics []Hash
}

// EventParachainSystemDownwardMessagesProcessed is emitted when downward messages
// were processed using the given weight.
type EventParachainSystemDownwardMessagesProcessed struct {
	Phase         Phase
	Weight        Weight
	ResultMqcHead Hash
	Topics        []Hash
}

// EventSessionNewSession is emitted when a new session has happened. Note that the argument is the session index,
// not the block number as the type might suggest
type EventSessionNewSession struct {
	Phase        Phase
	SessionIndex U32
	Topics       []Hash
}

// EventSlotsNewLeasePeriod is emitted when a new `[lease_period]` is beginning.
type EventSlotsNewLeasePeriod struct {
	Phase       Phase
	LeasePeriod U32
	Topics      []Hash
}

type ParachainID U32

// EventSlotsLeased is emitted when a para has won the right to a continuous set of lease periods as a parachain.
// First balance is any extra amount reserved on top of the para's existing deposit.
// Second balance is the total amount reserved.
type EventSlotsLeased struct {
	Phase         Phase
	ParachainID   ParachainID
	Leaser        AccountID
	PeriodBegin   U32
	PeriodCount   U32
	ExtraReserved U128
	TotalAmount   U128
	Topics        []Hash
}

// EventStakingEraPaid is emitted when the era payout has been set;
type EventStakingEraPaid struct {
	Phase           Phase
	EraIndex        U32
	ValidatorPayout U128
	Remainder       U128
	Topics          []Hash
}

// EventStakingRewarded is emitted when the staker has been rewarded by this amount.
type EventStakingRewarded struct {
	Phase  Phase
	Stash  AccountID
	Amount U128
	Topics []Hash
}

// EventStakingSlashed is emitted when one validator (and its nominators) has been slashed by the given amount
type EventStakingSlashed struct {
	Phase     Phase
	AccountID AccountID
	Balance   U128
	Topics    []Hash
}

// EventStakingOldSlashingReportDiscarded is emitted when an old slashing report from a prior era was discarded because
// it could not be processed
type EventStakingOldSlashingReportDiscarded struct {
	Phase        Phase
	SessionIndex U32
	Topics       []Hash
}

// EventStakingStakersElected is emitted when a new set of stakers was elected
type EventStakingStakersElected struct {
	Phase  Phase
	Topics []Hash
}

// EventStakingStakingElectionFailed is emitted when the election failed. No new era is planned.
type EventStakingStakingElectionFailed struct {
	Phase  Phase
	Topics []Hash
}

// EventStakingSolutionStored is emitted when a new solution for the upcoming election has been stored
type EventStakingSolutionStored struct {
	Phase   Phase
	Compute ElectionCompute
	Topics  []Hash
}

// EventStakingBonded is emitted when an account has bonded this amount
type EventStakingBonded struct {
	Phase  Phase
	Stash  AccountID
	Amount U128
	Topics []Hash
}

// EventStakingChilled is emitted when an account has stopped participating as either a validator or nominator
type EventStakingChilled struct {
	Phase  Phase
	Stash  AccountID
	Topics []Hash
}

// EventStakingKicked is emitted when a nominator has been kicked from a validator.
type EventStakingKicked struct {
	Phase     Phase
	Nominator AccountID
	Stash     AccountID
	Topics    []Hash
}

// EventStakingPayoutStarted is emitted when the stakers' rewards are getting paid
type EventStakingPayoutStarted struct {
	Phase    Phase
	EraIndex U32
	Stash    AccountID
	Topics   []Hash
}

// EventStakingUnbonded is emitted when an account has unbonded this amount
type EventStakingUnbonded struct {
	Phase  Phase
	Stash  AccountID
	Amount U128
	Topics []Hash
}

// EventStakingWithdrawn is emitted when an account has called `withdraw_unbonded` and removed unbonding chunks
// worth `Balance` from the unlocking queue.
type EventStakingWithdrawn struct {
	Phase  Phase
	Stash  AccountID
	Amount U128
	Topics []Hash
}

// EventStateTrieMigrationMigrated is emitted when the given number of `(top, child)` keys were migrated respectively,
// with the given `compute`.
type EventStateTrieMigrationMigrated struct {
	Phase   Phase
	Top     U32
	Child   U32
	Compute MigrationCompute
	Topics  []Hash
}

// EventStateTrieMigrationSlashed is emitted when some account got slashed by the given amount.
type EventStateTrieMigrationSlashed struct {
	Phase  Phase
	Who    AccountID
	Amount U128
	Topics []Hash
}

// EventStateTrieMigrationAutoMigrationFinished is emitted when the auto migration task has finished.
type EventStateTrieMigrationAutoMigrationFinished struct {
	Phase  Phase
	Topics []Hash
}

// EventStateTrieMigrationHalted is emitted when the migration got halted.
type EventStateTrieMigrationHalted struct {
	Phase  Phase
	Topics []Hash
}

// EventSystemExtrinsicSuccessV8 is emitted when an extrinsic completed successfully
//
// Deprecated: EventSystemExtrinsicSuccessV8 exists to allow users to simply implement their own EventRecords struct if
// they are on metadata version 8 or below. Use EventSystemExtrinsicSuccess otherwise
type EventSystemExtrinsicSuccessV8 struct {
	Phase  Phase
	Topics []Hash
}

// EventSystemExtrinsicSuccess is emitted when an extrinsic completed successfully
type EventSystemExtrinsicSuccess struct {
	Phase        Phase
	DispatchInfo DispatchInfo
	Topics       []Hash
}

type Pays struct {
	IsYes bool
	IsNo  bool
}

func (p *Pays) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		p.IsYes = true
	case 1:
		p.IsNo = true
	}

	return nil
}

func (p Pays) Encode(encoder scale.Encoder) error {
	var err error
	if p.IsYes {
		err = encoder.PushByte(0)
	} else if p.IsNo {
		err = encoder.PushByte(1)
	}
	return err
}

// DispatchInfo contains a bundle of static information collected from the `#[weight = $x]` attributes.
type DispatchInfo struct {
	// Weight of this transaction
	Weight Weight
	// Class of this transaction
	Class DispatchClass
	// PaysFee indicates whether this transaction pays fees
	PaysFee Pays
}

func (d *DispatchInfo) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&d.Weight); err != nil {
		return err
	}

	if err := decoder.Decode(&d.Class); err != nil {
		return err
	}

	return decoder.Decode(&d.PaysFee)
}

// DispatchClass is a generalized group of dispatch types. This is only distinguishing normal, user-triggered
// transactions (`Normal`) and anything beyond which serves a higher purpose to the system (`Operational`).
type DispatchClass struct {
	// A normal dispatch
	IsNormal bool
	// An operational dispatch
	IsOperational bool
	// A mandatory dispatch
	IsMandatory bool
}

func (d *DispatchClass) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.IsNormal = true
	case 1:
		d.IsOperational = true
	case 2:
		d.IsMandatory = true
	}

	return nil
}

func (d DispatchClass) Encode(encoder scale.Encoder) error {
	switch {
	case d.IsNormal:
		return encoder.PushByte(0)
	case d.IsOperational:
		return encoder.PushByte(1)
	case d.IsMandatory:
		return encoder.PushByte(2)
	}

	return nil
}

// EventSystemExtrinsicFailedV8 is emitted when an extrinsic failed
//
// Deprecated: EventSystemExtrinsicFailedV8 exists to allow users to simply implement their own EventRecords struct if
// they are on metadata version 8 or below. Use EventSystemExtrinsicFailed otherwise
type EventSystemExtrinsicFailedV8 struct {
	Phase         Phase
	DispatchError DispatchError
	Topics        []Hash
}

// EventSystemExtrinsicFailed is emitted when an extrinsic failed
type EventSystemExtrinsicFailed struct {
	Phase         Phase
	DispatchError DispatchError
	DispatchInfo  DispatchInfo
	Topics        []Hash
}

// EventSystemCodeUpdated is emitted when the runtime code (`:code`) is updated
type EventSystemCodeUpdated struct {
	Phase  Phase
	Topics []Hash
}

// EventSystemNewAccount is emitted when a new account was created
type EventSystemNewAccount struct {
	Phase  Phase
	Who    AccountID
	Topics []Hash
}

// EventSystemRemarked is emitted when an on-chain remark happened
type EventSystemRemarked struct {
	Phase  Phase
	Who    AccountID
	Hash   Hash
	Topics []Hash
}

// EventSystemKilledAccount is emitted when an account is reaped
type EventSystemKilledAccount struct {
	Phase  Phase
	Who    AccountID
	Topics []Hash
}

// EventAssetIssued is emitted when an asset is issued.
type EventAssetIssued struct {
	Phase   Phase
	AssetID U32
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventAssetCreated is emitted when an asset is created.
type EventAssetCreated struct {
	Phase   Phase
	AssetID U32
	Creator AccountID
	Owner   AccountID
	Topics  []Hash
}

// EventAssetTransferred is emitted when an asset is transferred.
type EventAssetTransferred struct {
	Phase   Phase
	AssetID U32
	To      AccountID
	From    AccountID
	Balance U128
	Topics  []Hash
}

// EventAssetBurned is emitted when an asset is destroyed.
type EventAssetBurned struct {
	Phase   Phase
	AssetID U32
	Owner   AccountID
	Balance U128
	Topics  []Hash
}

// EventAssetTeamChanged is emitted when the management team changed.
type EventAssetTeamChanged struct {
	Phase   Phase
	AssetID U32
	Issuer  AccountID
	Admin   AccountID
	Freezer AccountID
	Topics  []Hash
}

// EventAssetOwnerChanged is emitted when the owner changed.
type EventAssetOwnerChanged struct {
	Phase   Phase
	AssetID U32
	Owner   AccountID
	Topics  []Hash
}

// EventAssetFrozen is emitted when some account `who` was frozen.
type EventAssetFrozen struct {
	Phase   Phase
	AssetID U32
	Who     AccountID
	Topics  []Hash
}

// EventAssetThawed is emitted when some account `who` was thawed.
type EventAssetThawed struct {
	Phase   Phase
	AssetID U32
	Who     AccountID
	Topics  []Hash
}

// EventAssetAssetFrozen is emitted when some asset `asset_id` was frozen.
type EventAssetAssetFrozen struct {
	Phase   Phase
	AssetID U32
	Topics  []Hash
}

// EventAssetAssetThawed is emitted when some asset `asset_id` was thawed.
type EventAssetAssetThawed struct {
	Phase   Phase
	AssetID U32
	Topics  []Hash
}

// EventAssetDestroyed is emitted when an asset class is destroyed.
type EventAssetDestroyed struct {
	Phase   Phase
	AssetID U32
	Topics  []Hash
}

// EventAssetForceCreated is emitted when some asset class was force-created.
type EventAssetForceCreated struct {
	Phase   Phase
	AssetID U32
	Owner   AccountID
	Topics  []Hash
}

type MetadataSetName []byte
type MetadataSetSymbol []byte

// EventAssetMetadataSet is emitted when new metadata has been set for an asset.
type EventAssetMetadataSet struct {
	Phase    Phase
	AssetID  U32
	Name     MetadataSetName
	Symbol   MetadataSetSymbol
	Decimals U8
	IsFrozen bool
	Topics   []Hash
}

// EventAssetMetadataCleared is emitted when metadata has been cleared for an asset.
type EventAssetMetadataCleared struct {
	Phase   Phase
	AssetID U32
	Topics  []Hash
}

// EventAssetApprovedTransfer is emitted when (additional) funds have been approved
// for transfer to a destination account.
type EventAssetApprovedTransfer struct {
	Phase    Phase
	AssetID  U32
	Source   AccountID
	Delegate AccountID
	Amount   U128
	Topics   []Hash
}

// EventAssetApprovalCancelled is emitted when an approval for account `delegate` was cancelled by `owner`.
type EventAssetApprovalCancelled struct {
	Phase    Phase
	AssetID  U32
	Owner    AccountID
	Delegate AccountID
	Topics   []Hash
}

// EventAssetTransferredApproved is emitted when an `amount` was transferred in its
// entirety from `owner` to `destination` by the approved `delegate`.
type EventAssetTransferredApproved struct {
	Phase       Phase
	AssetID     U32
	Owner       AccountID
	Delegate    AccountID
	Destination AccountID
	Amount      U128
	Topics      []Hash
}

// EventAssetAssetStatusChanged is emitted when an asset has had its attributes changed by the `Force` origin.
type EventAssetAssetStatusChanged struct {
	Phase   Phase
	AssetID U32
	Topics  []Hash
}

// EventAuctionsAuctionStarted is emitted when an auction started. Provides its index and the block number
// where it will begin to close and the first lease period of the quadruplet that is auctioned.
type EventAuctionsAuctionStarted struct {
	Phase        Phase
	AuctionIndex U32
	LeasePeriod  U32
	Ending       U32
	Topics       []Hash
}

// EventAuctionsAuctionClosed is emitted when an auction ended. All funds become unreserved.
type EventAuctionsAuctionClosed struct {
	Phase        Phase
	AuctionIndex U32
	Topics       []Hash
}

// EventAuctionsReserved is emitted when funds were reserved for a winning bid.
// First balance is the extra amount reserved. Second is the total.
type EventAuctionsReserved struct {
	Phase         Phase
	Bidder        AccountID
	ExtraReserved U128
	TotalAmount   U128
	Topics        []Hash
}

// EventAuctionsUnreserved is emitted when funds were unreserved since bidder is no longer active.
type EventAuctionsUnreserved struct {
	Phase  Phase
	Bidder AccountID
	Amount U128
	Topics []Hash
}

// EventAuctionsReserveConfiscated is emitted when someone attempted to lease the same slot twice for a parachain.
// The amount is held in reserve but no parachain slot has been leased.
type EventAuctionsReserveConfiscated struct {
	Phase       Phase
	ParachainID ParachainID
	Leaser      AccountID
	Amount      U128
	Topics      []Hash
}

// EventAuctionsBidAccepted is emitted when a new bid has been accepted as the current winner.
type EventAuctionsBidAccepted struct {
	Phase       Phase
	Who         AccountID
	ParachainID ParachainID
	Amount      U128
	FirstSlot   U32
	LastSlot    U32
	Topics      []Hash
}

// EventAuctionsWinningOffset is emitted when the winning offset was chosen for an auction.
// This will map into the `Winning` storage map.
type EventAuctionsWinningOffset struct {
	Phase        Phase
	AuctionIndex U32
	BlockNumber  U32
	Topics       []Hash
}

// EventBagsListRebagged is emitted when an account was moved from one bag to another.
type EventBagsListRebagged struct {
	Phase  Phase
	Who    AccountID
	From   U64
	To     U64
	Topics []Hash
}

// EventDemocracyProposed is emitted when a motion has been proposed by a public account.
type EventDemocracyProposed struct {
	Phase         Phase
	ProposalIndex U32
	Balance       U128
	Topics        []Hash
}

// EventDemocracyTabled is emitted when a public proposal has been tabled for referendum vote.
type EventDemocracyTabled struct {
	Phase         Phase
	ProposalIndex U32
	Balance       U128
	Accounts      []AccountID
	Topics        []Hash
}

// EventDemocracyExternalTabled is emitted when an external proposal has been tabled.
type EventDemocracyExternalTabled struct {
	Phase  Phase
	Topics []Hash
}

// VoteThreshold is a means of determining if a vote is past pass threshold.
type VoteThreshold byte

const (
	// SuperMajorityApprove require super majority of approvals is needed to pass this vote.
	SuperMajorityApprove VoteThreshold = 0
	// SuperMajorityAgainst require super majority of rejects is needed to fail this vote.
	SuperMajorityAgainst VoteThreshold = 1
	// SimpleMajority require simple majority of approvals is needed to pass this vote.
	SimpleMajority VoteThreshold = 2
)

func (v *VoteThreshold) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := VoteThreshold(b)
	switch vb {
	case SuperMajorityApprove, SuperMajorityAgainst, SimpleMajority:
		*v = vb
	default:
		return fmt.Errorf("unknown VoteThreshold enum: %v", vb)
	}
	return err
}

func (v VoteThreshold) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(v))
}

type DemocracyConviction byte

const (
	// None 0.1x votes, unlocked
	None = 0
	// Locked1x votes, locked for an enactment period following a successful vote.
	Locked1x = 1
	// Locked2x votes, locked for 2x enactment periods following a successful vote.
	Locked2x = 2
	// Locked3x votes, locked for 4x...
	Locked3x = 3
	// Locked4x votes, locked for 8x...
	Locked4x = 4
	// Locked5x votes, locked for 16x...
	Locked5x = 5
	// Locked6x votes, locked for 32x...
	Locked6x = 6
)

func (dc *DemocracyConviction) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := DemocracyConviction(b)
	switch vb {
	case None, Locked1x, Locked2x, Locked3x, Locked4x, Locked5x, Locked6x:
		*dc = vb
	default:
		return fmt.Errorf("unknown DemocracyConviction enum: %v", vb)
	}
	return err
}

func (dc DemocracyConviction) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(dc))
}

type DemocracyVote struct {
	Aye        bool
	Conviction DemocracyConviction
}

const (
	aye uint8 = 1 << 7
)

//nolint:lll
func (d *DemocracyVote) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	// As per:
	// https://github.com/paritytech/substrate/blob/6a946fc36d68b89599d7ca1ab03803d10c78468c/frame/democracy/src/vote.rs#L44

	d.Aye = (b & aye) == aye
	d.Conviction = DemocracyConviction(b & (aye - 1))

	return nil
}

//nolint:lll
func (d DemocracyVote) Encode(encoder scale.Encoder) error {
	// As per:
	// https://github.com/paritytech/substrate/blob/6a946fc36d68b89599d7ca1ab03803d10c78468c/frame/democracy/src/vote.rs#L37

	var val uint8

	if d.Aye {
		val = aye
	}

	return encoder.PushByte(uint8(d.Conviction) | val)
}

type VoteAccountVoteAsStandard struct {
	Vote    DemocracyVote
	Balance U128
}

func (v *VoteAccountVoteAsStandard) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&v.Vote); err != nil {
		return err
	}

	return decoder.Decode(&v.Balance)
}

func (v VoteAccountVoteAsStandard) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(v.Vote); err != nil {
		return err
	}

	return encoder.Encode(v.Balance)
}

type VoteAccountVoteAsSplit struct {
	Aye U128
	Nay U128
}

type VoteAccountVote struct {
	IsStandard bool
	AsStandard VoteAccountVoteAsStandard
	IsSplit    bool
	AsSplit    VoteAccountVoteAsSplit
}

func (vv *VoteAccountVote) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		vv.IsStandard = true

		return decoder.Decode(&vv.AsStandard)
	case 1:
		vv.IsSplit = true

		return decoder.Decode(&vv.AsSplit)
	}

	return nil
}

func (vv VoteAccountVote) Encode(encoder scale.Encoder) error {
	switch {
	case vv.IsStandard:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(vv.AsStandard)
	case vv.IsSplit:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(vv.AsSplit)
	}

	return nil
}

// EventDemocracyStarted is emitted when a referendum has begun.
type EventDemocracyStarted struct {
	Phase           Phase
	ReferendumIndex U32
	VoteThreshold   VoteThreshold
	Topics          []Hash
}

// EventDemocracyPassed is emitted when a proposal has been approved by referendum.
type EventDemocracyPassed struct {
	Phase           Phase
	ReferendumIndex U32
	Topics          []Hash
}

// EventDemocracyNotPassed is emitted when a proposal has been rejected by referendum.
type EventDemocracyNotPassed struct {
	Phase           Phase
	ReferendumIndex U32
	Topics          []Hash
}

// EventDemocracyCancelled is emitted when a referendum has been cancelled.
type EventDemocracyCancelled struct {
	Phase           Phase
	ReferendumIndex U32
	Topics          []Hash
}

// EventDemocracyExecuted is emitted when a proposal has been enacted.
type EventDemocracyExecuted struct {
	Phase           Phase
	ReferendumIndex U32
	Result          DispatchResult
	Topics          []Hash
}

// EventDemocracyDelegated is emitted when an account has delegated their vote to another account.
type EventDemocracyDelegated struct {
	Phase  Phase
	Who    AccountID
	Target AccountID
	Topics []Hash
}

// EventDemocracyUndelegated is emitted when an account has cancelled a previous delegation operation.
type EventDemocracyUndelegated struct {
	Phase  Phase
	Target AccountID
	Topics []Hash
}

// EventDemocracyVetoed is emitted when an external proposal has been vetoed.
type EventDemocracyVetoed struct {
	Phase       Phase
	Who         AccountID
	Hash        Hash
	BlockNumber U32
	Topics      []Hash
}

// EventDemocracyVoted is emitted when an account has voted in a referendum.
type EventDemocracyVoted struct {
	Phase           Phase
	Who             AccountID
	ReferendumIndex U32
	Vote            VoteAccountVote
	Topics          []Hash
}

// EventElectionProviderMultiPhaseSolutionStored is emitted when a solution was stored with the given compute.
//
// If the solution is signed, this means that it hasn't yet been processed. If the
// solution is unsigned, this means that it has also been processed.
//
// The `bool` is `true` when a previous solution was ejected to make room for this one.
type EventElectionProviderMultiPhaseSolutionStored struct {
	Phase           Phase
	ElectionCompute ElectionCompute
	PrevEjected     bool
	Topics          []Hash
}

// EventElectionProviderMultiPhaseElectionFinalized is emitted when the election has been finalized,
// with `Some` of the given computation, or else if the election failed, `None`.
type EventElectionProviderMultiPhaseElectionFinalized struct {
	Phase           Phase
	ElectionCompute OptionElectionCompute
	Topics          []Hash
}

// EventElectionProviderMultiPhaseRewarded is emitted when an account has been rewarded for their
// signed submission being finalized.
type EventElectionProviderMultiPhaseRewarded struct {
	Phase   Phase
	Account AccountID
	Value   U128
	Topics  []Hash
}

// EventElectionProviderMultiPhaseSlashed is emitted when an account has been slashed for
// submitting an invalid signed submission.
type EventElectionProviderMultiPhaseSlashed struct {
	Phase   Phase
	Account AccountID
	Value   U128
	Topics  []Hash
}

// EventElectionProviderMultiPhaseSignedPhaseStarted is emitted when the signed phase of the given round has started.
type EventElectionProviderMultiPhaseSignedPhaseStarted struct {
	Phase  Phase
	Round  U32
	Topics []Hash
}

// EventElectionProviderMultiPhaseUnsignedPhaseStarted is emitted when the unsigned phase of
// the given round has started.
type EventElectionProviderMultiPhaseUnsignedPhaseStarted struct {
	Phase  Phase
	Round  U32
	Topics []Hash
}

// EventDemocracyPreimageNoted is emitted when a proposal's preimage was noted, and the deposit taken.
type EventDemocracyPreimageNoted struct {
	Phase     Phase
	Hash      Hash
	AccountID AccountID
	Balance   U128
	Topics    []Hash
}

// EventDemocracyPreimageUsed is emitted when a proposal preimage was removed and used (the deposit was returned).
type EventDemocracyPreimageUsed struct {
	Phase     Phase
	Hash      Hash
	AccountID AccountID
	Balance   U128
	Topics    []Hash
}

// EventDemocracyPreimageInvalid is emitted when a proposal could not be executed because its preimage was invalid.
type EventDemocracyPreimageInvalid struct {
	Phase           Phase
	Hash            Hash
	ReferendumIndex U32
	Topics          []Hash
}

// EventDemocracyPreimageMissing is emitted when a proposal could not be executed because its preimage was missing.
type EventDemocracyPreimageMissing struct {
	Phase           Phase
	Hash            Hash
	ReferendumIndex U32
	Topics          []Hash
}

// EventDemocracyPreimageReaped is emitted when a registered preimage was removed
// and the deposit collected by the reaper (last item).
type EventDemocracyPreimageReaped struct {
	Phase    Phase
	Hash     Hash
	Provider AccountID
	Balance  U128
	Who      AccountID
	Topics   []Hash
}

// EventDemocracySeconded is emitted when an account has seconded a proposal.
type EventDemocracySeconded struct {
	Phase     Phase
	AccountID AccountID
	Balance   U128
	Topics    []Hash
}

// EventDemocracyBlacklisted is emitted when A proposal has been blacklisted permanently
type EventDemocracyBlacklisted struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventCouncilProposed is emitted when a motion (given hash) has been proposed (by given account)
// with a threshold (given `MemberCount`).
type EventCouncilProposed struct {
	Phase         Phase
	Who           AccountID
	ProposalIndex U32
	Proposal      Hash
	MemberCount   U32
	Topics        []Hash
}

// EventCollectiveVote is emitted when a motion (given hash) has been voted on by given account, leaving
// a tally (yes votes and no votes given respectively as `MemberCount`).
type EventCouncilVoted struct {
	Phase    Phase
	Who      AccountID
	Proposal Hash
	Approve  bool
	YesCount U32
	NoCount  U32
	Topics   []Hash
}

// EventCrowdloanCreated is emitted when a new crowdloaning campaign is created.
type EventCrowdloanCreated struct {
	Phase     Phase
	FundIndex U32
	Topics    []Hash
}

// EventCrowdloanContributed is emitted when `who` contributed to a crowd sale.
type EventCrowdloanContributed struct {
	Phase     Phase
	Who       AccountID
	FundIndex U32
	Amount    U128
	Topics    []Hash
}

// EventCrowdloanWithdrew is emitted when the full balance of a contributor was withdrawn.
type EventCrowdloanWithdrew struct {
	Phase     Phase
	Who       AccountID
	FundIndex U32
	Amount    U128
	Topics    []Hash
}

// EventCrowdloanPartiallyRefunded is emitted when the loans in a fund have been partially dissolved, i.e.
// there are some left over child keys that still need to be killed.
type EventCrowdloanPartiallyRefunded struct {
	Phase     Phase
	FundIndex U32
	Topics    []Hash
}

// EventCrowdloanAllRefunded is emitted when all loans in a fund have been refunded.
type EventCrowdloanAllRefunded struct {
	Phase     Phase
	FundIndex U32
	Topics    []Hash
}

// EventCrowdloanDissolved is emitted when the fund is dissolved.
type EventCrowdloanDissolved struct {
	Phase     Phase
	FundIndex U32
	Topics    []Hash
}

// EventCrowdloanHandleBidResult is emitted when trying to submit a new bid to the Slots pallet.
type EventCrowdloanHandleBidResult struct {
	Phase          Phase
	FundIndex      U32
	DispatchResult DispatchResult
	Topics         []Hash
}

// EventCrowdloanEdited is emitted when the configuration to a crowdloan has been edited.
type EventCrowdloanEdited struct {
	Phase     Phase
	FundIndex U32
	Topics    []Hash
}

type CrowloadMemo []byte

// EventCrowdloanMemoUpdated is emitted when a memo has been updated.
type EventCrowdloanMemoUpdated struct {
	Phase     Phase
	Who       AccountID
	FundIndex U32
	Memo      CrowloadMemo
	Topics    []Hash
}

// EventCrowdloanAddedToNewRaise is emitted when a parachain has been moved to `NewRaise`.
type EventCrowdloanAddedToNewRaise struct {
	Phase     Phase
	FundIndex U32
	Topics    []Hash
}

// EventCouncilApproved is emitted when a motion was approved by the required threshold.
type EventCouncilApproved struct {
	Phase    Phase
	Proposal Hash
	Topics   []Hash
}

// EventCouncilDisapproved is emitted when a motion was not approved by the required threshold.
type EventCouncilDisapproved struct {
	Phase    Phase
	Proposal Hash
	Topics   []Hash
}

// EventCouncilExecuted is emitted when a motion was executed; `result` is true if returned without error.
type EventCouncilExecuted struct {
	Phase    Phase
	Proposal Hash
	Result   DispatchResult
	Topics   []Hash
}

// EventCouncilMemberExecuted is emitted when a single member did some action;
// `result` is true if returned without error.
type EventCouncilMemberExecuted struct {
	Phase    Phase
	Proposal Hash
	Result   DispatchResult
	Topics   []Hash
}

// EventCouncilClosed is emitted when a proposal was closed after its duration was up.
type EventCouncilClosed struct {
	Phase    Phase
	Proposal Hash
	YesCount U32
	NoCount  U32
	Topics   []Hash
}

// EventTechnicalCommitteeProposed is emitted when a motion (given hash) has been proposed (by given account)
// with a threshold (given, `MemberCount`)
type EventTechnicalCommitteeProposed struct {
	Phase         Phase
	Account       AccountID
	ProposalIndex U32
	Proposal      Hash
	Threshold     U32
	Topics        []Hash
}

// EventTechnicalCommitteeVoted is emitted when a motion (given hash) has been voted on by given account, leaving,
// a tally (yes votes and no votes given respectively as `MemberCount`).
type EventTechnicalCommitteeVoted struct {
	Phase    Phase
	Account  AccountID
	Proposal Hash
	Voted    bool
	YesCount U32
	NoCount  U32
	Topics   []Hash
}

// EventTechnicalCommitteeApproved is emitted when a motion was approved by the required threshold.
type EventTechnicalCommitteeApproved struct {
	Phase    Phase
	Proposal Hash
	Topics   []Hash
}

// EventTechnicalCommitteeDisapproved is emitted when a motion was not approved by the required threshold.
type EventTechnicalCommitteeDisapproved struct {
	Phase    Phase
	Proposal Hash
	Topics   []Hash
}

// EventTechnicalCommitteeExecuted is emitted when a motion was executed;
// result will be `Ok` if it returned without error.
type EventTechnicalCommitteeExecuted struct {
	Phase    Phase
	Proposal Hash
	Result   DispatchResult
	Topics   []Hash
}

// EventTechnicalCommitteeMemberExecuted is emitted when a single member did some action;
// result will be `Ok` if it returned without error
type EventTechnicalCommitteeMemberExecuted struct {
	Phase    Phase
	Proposal Hash
	Result   DispatchResult
	Topics   []Hash
}

// EventTechnicalCommitteeClosed is emitted when A proposal was closed because its threshold was reached
// or after its duration was up
type EventTechnicalCommitteeClosed struct {
	Phase    Phase
	Proposal Hash
	YesCount U32
	NoCount  U32
	Topics   []Hash
}

// EventTechnicalMembershipMemberAdded is emitted when the given member was added; see the transaction for who
type EventTechnicalMembershipMemberAdded struct {
	Phase  Phase
	Topics []Hash
}

// EventTechnicalMembershipMemberRemoved is emitted when the given member was removed; see the transaction for who
type EventTechnicalMembershipMemberRemoved struct {
	Phase  Phase
	Topics []Hash
}

// EventTechnicalMembershipMembersSwapped is emitted when two members were swapped;; see the transaction for who
type EventTechnicalMembershipMembersSwapped struct {
	Phase  Phase
	Topics []Hash
}

// EventTechnicalMembershipMembersReset is emitted when the membership was reset;
// see the transaction for who the new set is.
type EventTechnicalMembershipMembersReset struct {
	Phase  Phase
	Topics []Hash
}

// EventTechnicalMembershipKeyChanged is emitted when one of the members' keys changed.
type EventTechnicalMembershipKeyChanged struct {
	Phase  Phase
	Topics []Hash
}

// EventTechnicalMembershipKeyChanged is emitted when - phantom member, never used.
type EventTechnicalMembershipDummy struct {
	Phase  Phase
	Topics []Hash
}

// EventElectionsNewTerm is emitted when a new term with new members.
// This indicates that enough candidates existed, not that enough have has been elected.
// The inner value must be examined for this purpose.
type EventElectionsNewTerm struct {
	Phase      Phase
	NewMembers []struct {
		Member  AccountID
		Balance U128
	}
	Topics []Hash
}

// EventElectionsCandidateSlashed is emitted when a candidate was slashed by amount due to failing to obtain a seat
// as member or runner-up. Note that old members and runners-up are also candidates.
type EventElectionsCandidateSlashed struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventElectionsEmptyTerm is emitted when No (or not enough) candidates existed for this round.
type EventElectionsEmptyTerm struct {
	Phase  Phase
	Topics []Hash
}

// EventElectionsElectionError is emitted when an internal error happened while trying to perform election
type EventElectionsElectionError struct {
	Phase  Phase
	Topics []Hash
}

// EventElectionsMemberKicked is emitted when a member has been removed.
// This should always be followed by either `NewTerm` or `EmptyTerm`.
type EventElectionsMemberKicked struct {
	Phase  Phase
	Member AccountID
	Topics []Hash
}

// EventElectionsRenounced is emitted when a member has renounced their candidacy.
type EventElectionsRenounced struct {
	Phase  Phase
	Member AccountID
	Topics []Hash
}

// EventElectionsSeatHolderSlashed is emitted when a seat holder was slashed by amount
// by being forcefully removed from the set
type EventElectionsSeatHolderSlashed struct {
	Phase   Phase
	Who     AccountID
	Balance U128
	Topics  []Hash
}

// EventGiltBidPlaced is emitted when a bid was successfully placed.
type EventGiltBidPlaced struct {
	Phase    Phase
	Who      AccountID
	Amount   U128
	Duration U32
	Topics   []Hash
}

// EventGiltBidRetracted is emitted when a bid was successfully removed (before being accepted as a gilt).
type EventGiltBidRetracted struct {
	Phase    Phase
	Who      AccountID
	Amount   U128
	Duration U32
	Topics   []Hash
}

// EventGiltGiltIssued is emitted when a bid was accepted as a gilt. The balance may not be released until expiry.
type EventGiltGiltIssued struct {
	Phase  Phase
	Index  U32
	Expiry U32
	Who    AccountID
	Amount U128
	Topics []Hash
}

// EventGiltGiltThawed is emitted when an expired gilt has been thawed.
type EventGiltGiltThawed struct {
	Phase            Phase
	Index            U32
	Who              AccountID
	OriginalAmount   U128
	AdditionalAmount U128
	Topics           []Hash
}

// A name was set or reset (which will remove all judgements).
type EventIdentitySet struct {
	Phase    Phase
	Identity AccountID
	Topics   []Hash
}

// A name was cleared, and the given balance returned.
type EventIdentityCleared struct {
	Phase    Phase
	Identity AccountID
	Balance  U128
	Topics   []Hash
}

// A name was removed and the given balance slashed.
type EventIdentityKilled struct {
	Phase    Phase
	Identity AccountID
	Balance  U128
	Topics   []Hash
}

// A judgement was asked from a registrar.
type EventIdentityJudgementRequested struct {
	Phase          Phase
	Sender         AccountID
	RegistrarIndex U32
	Topics         []Hash
}

// A judgement request was retracted.
type EventIdentityJudgementUnrequested struct {
	Phase          Phase
	Sender         AccountID
	RegistrarIndex U32
	Topics         []Hash
}

// A judgement was given by a registrar.
type EventIdentityJudgementGiven struct {
	Phase          Phase
	Target         AccountID
	RegistrarIndex U32
	Topics         []Hash
}

// A registrar was added.
type EventIdentityRegistrarAdded struct {
	Phase          Phase
	RegistrarIndex U32
	Topics         []Hash
}

// EventIdentitySubIdentityAdded is emitted when a sub-identity was added to an identity and the deposit paid
type EventIdentitySubIdentityAdded struct {
	Phase   Phase
	Sub     AccountID
	Main    AccountID
	Deposit U128
	Topics  []Hash
}

// EventIdentitySubIdentityRemoved is emitted when a sub-identity was removed from an identity and the deposit freed
type EventIdentitySubIdentityRemoved struct {
	Phase   Phase
	Sub     AccountID
	Main    AccountID
	Deposit U128
	Topics  []Hash
}

// EventIdentitySubIdentityRevoked is emitted when a sub-identity was cleared, and the given deposit repatriated from
// the main identity account to the sub-identity account.
type EventIdentitySubIdentityRevoked struct {
	Phase   Phase
	Sub     AccountID
	Main    AccountID
	Deposit U128
	Topics  []Hash
}

// EventSocietyFounded is emitted when the society is founded by the given identity
type EventSocietyFounded struct {
	Phase   Phase
	Founder AccountID
	Topics  []Hash
}

// EventSocietyBid is emitted when a membership bid just happened. The given account is the candidate's ID
// and their offer is the second
type EventSocietyBid struct {
	Phase     Phase
	Candidate AccountID
	Offer     U128
	Topics    []Hash
}

// EventSocietyVouch is emitted when a membership bid just happened by vouching.
// The given account is the candidate's ID and, their offer is the second. The vouching party is the third.
type EventSocietyVouch struct {
	Phase     Phase
	Candidate AccountID
	Offer     U128
	Vouching  AccountID
	Topics    []Hash
}

// EventSocietyAutoUnbid is emitted when a [candidate] was dropped (due to an excess of bids in the system)
type EventSocietyAutoUnbid struct {
	Phase     Phase
	Candidate AccountID
	Topics    []Hash
}

// EventSocietyUnbid is emitted when a [candidate] was dropped (by their request)
type EventSocietyUnbid struct {
	Phase     Phase
	Candidate AccountID
	Topics    []Hash
}

// EventSocietyUnvouch is emitted when a [candidate] was dropped (by request of who vouched for them)
type EventSocietyUnvouch struct {
	Phase     Phase
	Candidate AccountID
	Topics    []Hash
}

// EventSocietyInducted is emitted when a group of candidates have been inducted.
// The batch's primary is the first value, the batch in full is the second.
type EventSocietyInducted struct {
	Phase      Phase
	Primary    AccountID
	Candidates []AccountID
	Topics     []Hash
}

// EventSocietySuspendedMemberJudgement is emitted when a suspended member has been judged
type EventSocietySuspendedMemberJudgement struct {
	Phase  Phase
	Who    AccountID
	Judged bool
	Topics []Hash
}

// EventSocietyCandidateSuspended is emitted when a [candidate] has been suspended
type EventSocietyCandidateSuspended struct {
	Phase     Phase
	Candidate AccountID
	Topics    []Hash
}

// EventSocietyMemberSuspended is emitted when a [member] has been suspended
type EventSocietyMemberSuspended struct {
	Phase  Phase
	Member AccountID
	Topics []Hash
}

// EventSocietyChallenged is emitted when a [member] has been challenged
type EventSocietyChallenged struct {
	Phase  Phase
	Member AccountID
	Topics []Hash
}

// EventSocietyVote is emitted when a vote has been placed
type EventSocietyVote struct {
	Phase     Phase
	Candidate AccountID
	Voter     AccountID
	Vote      bool
	Topics    []Hash
}

// EventSocietyDefenderVote is emitted when a vote has been placed for a defending member
type EventSocietyDefenderVote struct {
	Phase  Phase
	Voter  AccountID
	Vote   bool
	Topics []Hash
}

// EventSocietyNewMaxMembers is emitted when a new [max] member count has been set
type EventSocietyNewMaxMembers struct {
	Phase  Phase
	Max    U32
	Topics []Hash
}

// EventSocietyUnfounded is emitted when society is unfounded
type EventSocietyUnfounded struct {
	Phase   Phase
	Founder AccountID
	Topics  []Hash
}

// EventSocietyDeposit is emitted when some funds were deposited into the society account
type EventSocietyDeposit struct {
	Phase  Phase
	Value  U128
	Topics []Hash
}

// EventRecoveryCreated is emitted when a recovery process has been set up for an account
type EventRecoveryCreated struct {
	Phase  Phase
	Who    AccountID
	Topics []Hash
}

// EventRecoveryInitiated is emitted when a recovery process has been initiated for account_1 by account_2
type EventRecoveryInitiated struct {
	Phase   Phase
	Account AccountID
	Who     AccountID
	Topics  []Hash
}

// EventRecoveryVouched is emitted when a recovery process for account_1 by account_2 has been vouched for by account_3
type EventRecoveryVouched struct {
	Phase   Phase
	Lost    AccountID
	Rescuer AccountID
	Who     AccountID
	Topics  []Hash
}

// EventRegistrarRegistered is emitted when a parachain is registered.
type EventRegistrarRegistered struct {
	Phase       Phase
	ParachainID ParachainID
	Account     AccountID
	Topics      []Hash
}

// EventRegistrarDeregistered is emitted when a parachain is deregistered.
type EventRegistrarDeregistered struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []Hash
}

// EventRegistrarReserved is emitted when a parachain slot is reserved.
type EventRegistrarReserved struct {
	Phase       Phase
	ParachainID ParachainID
	Account     AccountID
	Topics      []Hash
}

// EventReferendaSubmitted is emitted when a referendum has been submitted.
type EventReferendaSubmitted struct {
	Phase        Phase
	Index        U32
	Track        U8
	ProposalHash Hash
	Topics       []Hash
}

// EventReferendaDecisionDepositPlaced is emitted when the decision deposit has been placed.
type EventReferendaDecisionDepositPlaced struct {
	Phase  Phase
	Index  U32
	Who    AccountID
	Amount U128
	Topics []Hash
}

// EventReferendaDecisionDepositRefunded is emitted when the decision deposit has been refunded.
type EventReferendaDecisionDepositRefunded struct {
	Phase  Phase
	Index  U32
	Who    AccountID
	Amount U128
	Topics []Hash
}

// EventReferendaDecisionSlashed is emitted when a deposit has been slashed.
type EventReferendaDecisionSlashed struct {
	Phase  Phase
	Who    AccountID
	Amount U128
	Topics []Hash
}

// EventReferendaDecisionStarted is emitted when a referendum has moved into the deciding phase.
type EventReferendaDecisionStarted struct {
	Phase        Phase
	Index        U32
	Track        U8
	ProposalHash Hash
	Tally        Tally
	Topics       []Hash
}

// EventReferendaConfirmStarted is emitted when a referendum has been started.
type EventReferendaConfirmStarted struct {
	Phase  Phase
	Index  U32
	Topics []Hash
}

// EventReferendaConfirmAborted is emitted when a referendum has been aborted.
type EventReferendaConfirmAborted struct {
	Phase  Phase
	Index  U32
	Topics []Hash
}

// EventReferendaConfirmed is emitted when a referendum has ended its confirmation phase and is ready for approval.
type EventReferendaConfirmed struct {
	Phase  Phase
	Index  U32
	Tally  Tally
	Topics []Hash
}

// EventReferendaApproved is emitted when a referendum has been approved and its proposal has been scheduled.
type EventReferendaApproved struct {
	Phase  Phase
	Index  U32
	Topics []Hash
}

// EventReferendaRejected is emitted when a proposal has been rejected by referendum.
type EventReferendaRejected struct {
	Phase  Phase
	Index  U32
	Tally  Tally
	Topics []Hash
}

// EventReferendaTimedOut is emitted when a referendum has been timed out without being decided.
type EventReferendaTimedOut struct {
	Phase  Phase
	Index  U32
	Tally  Tally
	Topics []Hash
}

// EventReferendaCancelled is emitted when a referendum has been cancelled.
type EventReferendaCancelled struct {
	Phase  Phase
	Index  U32
	Tally  Tally
	Topics []Hash
}

// EventReferendaKilled is emitted when a referendum has been killed.
type EventReferendaKilled struct {
	Phase  Phase
	Index  U32
	Tally  Tally
	Topics []Hash
}

// EventRecoveryClosed is emitted when a recovery process for account_1 by account_2 has been closed
type EventRecoveryClosed struct {
	Phase   Phase
	Who     AccountID
	Rescuer AccountID
	Topics  []Hash
}

// EventRecoveryAccountRecovered is emitted when account_1 has been successfully recovered by account_2
type EventRecoveryAccountRecovered struct {
	Phase   Phase
	Who     AccountID
	Rescuer AccountID
	Topics  []Hash
}

// EventRecoveryRemoved is emitted when a recovery process has been removed for an account
type EventRecoveryRemoved struct {
	Phase  Phase
	Who    AccountID
	Topics []Hash
}

// EventVestingVestingUpdated is emitted when the amount vested has been updated.
// This could indicate more funds are available.
// The balance given is the amount which is left unvested (and thus locked)
type EventVestingVestingUpdated struct {
	Phase    Phase
	Account  AccountID
	Unvested U128
	Topics   []Hash
}

// EventVoterListRebagged is emitted when an account is moved from one bag to another.
type EventVoterListRebagged struct {
	Phase  Phase
	Who    AccountID
	From   U64
	To     U64
	Topics []Hash
}

// EventVoterListScoreUpdated is emitted when the score of an account is updated to the given amount.
type EventVoterListScoreUpdated struct {
	Phase    Phase
	Who      AccountID
	NewScore U64
	Topics   []Hash
}

// EventWhitelistCallWhitelisted is emitted when a call has been whitelisted.
type EventWhitelistCallWhitelisted struct {
	Phase    Phase
	CallHash Hash
	Topics   []Hash
}

// EventWhitelistWhitelistedCallRemoved is emitted when a whitelisted call has been removed.
type EventWhitelistWhitelistedCallRemoved struct {
	Phase    Phase
	CallHash Hash
	Topics   []Hash
}

// EventWhitelistWhitelistedCallDispatched is emitted when a whitelisted call has been dispatched.
type EventWhitelistWhitelistedCallDispatched struct {
	Phase    Phase
	CallHash Hash
	Result   DispatchResult
	Topics   []Hash
}

// EventXcmPalletAttempted is emitted when the execution of an XCM message was attempted.
type EventXcmPalletAttempted struct {
	Phase   Phase
	Outcome Outcome
	Topics  []Hash
}

// EventXcmPalletSent is emitted when an XCM message was sent.
type EventXcmPalletSent struct {
	Phase       Phase
	Origin      MultiLocationV1
	Destination MultiLocationV1
	Message     []Instruction
	Topics      []Hash
}

// EventXcmPalletUnexpectedResponse is emitted when a query response which does not match a registered query
// is received.
// This may be because a matching query was never registered, it may be because it is a duplicate response, or
// because the query timed out.
type EventXcmPalletUnexpectedResponse struct {
	Phase          Phase
	OriginLocation MultiLocationV1
	QueryID        U64
	Topics         []Hash
}

// EventXcmPalletResponseReady is emitted when a query response has been received and is ready for
// taking with `take_response`. There is no registered notification call.
type EventXcmPalletResponseReady struct {
	Phase    Phase
	QueryID  U64
	Response Response
	Topics   []Hash
}

// EventXcmPalletNotified is emitted when a query response has been received and query is removed.
// The registered notification has been dispatched and executed successfully.
type EventXcmPalletNotified struct {
	Phase       Phase
	QueryID     U64
	PalletIndex U8
	CallIndex   U8
	Topics      []Hash
}

// EventXcmPalletNotifyOverweight is emitted when a query response has been received and query is removed.
// The registered notification could not be dispatched because the dispatch weight is greater than
// the maximum weight originally budgeted by this runtime for the query result.
type EventXcmPalletNotifyOverweight struct {
	Phase             Phase
	QueryID           U64
	PalletIndex       U8
	CallIndex         U8
	ActualWeight      Weight
	MaxBudgetedWeight Weight
	Topics            []Hash
}

// EventXcmPalletNotifyDispatchError is emitted when a query response has been received and query is removed.
// There was a general error with dispatching the notification call.
type EventXcmPalletNotifyDispatchError struct {
	Phase       Phase
	QueryID     U64
	PalletIndex U8
	CallIndex   U8
	Topics      []Hash
}

// EventXcmPalletNotifyDecodeFailed is emitted when a query response has been received and query is removed.
// The dispatch was unable to be decoded into a `Call`; this might be due to dispatch function having a signature
// which is not `(origin, QueryId, Response)`.
type EventXcmPalletNotifyDecodeFailed struct {
	Phase       Phase
	QueryID     U64
	PalletIndex U8
	CallIndex   U8
	Topics      []Hash
}

// EventXcmPalletInvalidResponder is emitted when the expected query response
// has been received but the origin location of the response does not match that expected.
// The query remains registered for a later, valid, response to be received and acted upon.
type EventXcmPalletInvalidResponder struct {
	Phase            Phase
	OriginLocation   MultiLocationV1
	QueryID          U64
	ExpectedLocation OptionMultiLocationV1
	Topics           []Hash
}

// EventXcmPalletInvalidResponderVersion is emitted when the expected query response
// has been received but the expected origin location placed in storage by this runtime
// previously cannot be decoded. The query remains registered.
// This is unexpected (since a location placed in storage in a previously executing
// runtime should be readable prior to query timeout) and dangerous since the possibly
// valid response will be dropped. Manual governance intervention is probably going to be
// needed.
type EventXcmPalletInvalidResponderVersion struct {
	Phase          Phase
	OriginLocation MultiLocationV1
	QueryID        U64
	Topics         []Hash
}

// EventXcmPalletResponseTaken is emitted when the received query response has been read and removed.
type EventXcmPalletResponseTaken struct {
	Phase   Phase
	QueryID U64
	Topics  []Hash
}

// EventXcmPalletAssetsTrapped is emitted when some assets have been placed in an asset trap.
type EventXcmPalletAssetsTrapped struct {
	Phase  Phase
	Hash   H256
	Origin MultiLocationV1
	Assets VersionedMultiAssets
	Topics []Hash
}

type XcmVersion U32

// EventXcmPalletVersionChangeNotified is emitted when an XCM version change notification
// message has been attempted to be sent.
type EventXcmPalletVersionChangeNotified struct {
	Phase       Phase
	Destination MultiLocationV1
	Result      XcmVersion
	Topics      []Hash
}

// EventXcmPalletSupportedVersionChanged is emitted when the supported version of a location has been changed.
// This might be through an automatic notification or a manual intervention.
type EventXcmPalletSupportedVersionChanged struct {
	Phase      Phase
	Location   MultiLocationV1
	XcmVersion XcmVersion
	Topics     []Hash
}

// EventXcmPalletNotifyTargetSendFail is emitted when a given location which had a version change
// subscription was dropped owing to an error sending the notification to it.
type EventXcmPalletNotifyTargetSendFail struct {
	Phase    Phase
	Location MultiLocationV1
	QueryID  U64
	XcmError XCMError
	Topics   []Hash
}

// EventXcmPalletNotifyTargetMigrationFail is emitted when a given location which had a
// version change subscription was dropped owing to an error migrating the location to our new XCM format.
type EventXcmPalletNotifyTargetMigrationFail struct {
	Phase    Phase
	Location VersionedMultiLocation
	QueryID  U64
	Topics   []Hash
}

// EventVestingVestingCompleted is emitted when an [account] has become fully vested. No further vesting can happen
type EventVestingVestingCompleted struct {
	Phase   Phase
	Account AccountID
	Topics  []Hash
}

// EventSchedulerScheduled is emitted when scheduled some task
type EventSchedulerScheduled struct {
	Phase  Phase
	When   U32
	Index  U32
	Topics []Hash
}

// EventSchedulerCanceled is emitted when canceled some task
type EventSchedulerCanceled struct {
	Phase  Phase
	When   U32
	Index  U32
	Topics []Hash
}

// EventSchedulerDispatched is emitted when dispatched some task
type EventSchedulerDispatched struct {
	Phase  Phase
	Task   TaskAddress
	ID     OptionBytes
	Result DispatchResult
	Topics []Hash
}

type SchedulerLookupError byte

const (
	// Unknown A call of this hash was not known.
	Unknown = 0
	// BadFormat The preimage for this hash was known but could not be decoded into a Call.
	BadFormat = 1
)

func (sle *SchedulerLookupError) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := SchedulerLookupError(b)
	switch vb {
	case Unknown, BadFormat:
		*sle = vb
	default:
		return fmt.Errorf("unknown SchedulerLookupError enum: %v", vb)
	}
	return err
}

func (sle SchedulerLookupError) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(sle))
}

// EventSchedulerCallLookupFailed is emitted when the call for the provided hash was not found
// so the task has been aborted.
type EventSchedulerCallLookupFailed struct {
	Phase  Phase
	Task   TaskAddress
	ID     OptionBytes
	Error  SchedulerLookupError
	Topics []Hash
}

// EventPreimageCleared is emitted when a preimage has been cleared
type EventPreimageCleared struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventPreimageNoted is emitted when a preimage has been noted
type EventPreimageNoted struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventPreimageRequested is emitted when a preimage has been requested
type EventPreimageRequested struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventProxyProxyExecuted is emitted when a proxy was executed correctly, with the given [result]
type EventProxyProxyExecuted struct {
	Phase  Phase
	Result DispatchResult
	Topics []Hash
}

// EventProxyPureCreated is emitted when an anonymous account has been created by new proxy with given,
// disambiguation index and proxy type.
type EventProxyPureCreated struct {
	Phase               Phase
	Pure                AccountID
	Who                 AccountID
	ProxyType           U8
	DisambiguationIndex U16
	Topics              []Hash
}

// EventProxyProxyAdded is emitted when a proxy was added.
type EventProxyProxyAdded struct {
	Phase     Phase
	Delegator AccountID
	Delegatee AccountID
	ProxyType U8
	Delay     U32
	Topics    []Hash
}

// EventProxyProxyRemoved is emitted when a proxy was removed.
type EventProxyProxyRemoved struct {
	Phase       Phase
	Delegator   AccountID
	Delegatee   AccountID
	ProxyType   U8
	BlockNumber U32
	Topics      []Hash
}

// EventProxyAnnounced is emitted when an announcement was placed to make a call in the future
type EventProxyAnnounced struct {
	Phase    Phase
	Real     AccountID
	Proxy    AccountID
	CallHash Hash
	Topics   []Hash
}

// EventSudoSudid is emitted when a sudo just took place.
type EventSudoSudid struct {
	Phase  Phase
	Result DispatchResult
	Topics []Hash
}

// EventSudoKeyChanged is emitted when the sudoer just switched identity; the old key is supplied.
type EventSudoKeyChanged struct {
	Phase     Phase
	AccountID AccountID
	Topics    []Hash
}

// A sudo just took place.
type EventSudoAsDone struct {
	Phase  Phase
	Done   bool
	Topics []Hash
}

// EventTreasuryProposed is emitted when New proposal.
type EventTreasuryProposed struct {
	Phase         Phase
	ProposalIndex U32
	Topics        []Hash
}

// EventTreasurySpending is emitted when we have ended a spend period and will now allocate funds.
type EventTreasurySpending struct {
	Phase           Phase
	BudgetRemaining U128
	Topics          []Hash
}

// EventTreasuryAwarded is emitted when some funds have been allocated.
type EventTreasuryAwarded struct {
	Phase         Phase
	ProposalIndex U32
	Amount        U128
	Beneficiary   AccountID
	Topics        []Hash
}

// EventTreasuryRejected is emitted when s proposal was rejected; funds were slashed.
type EventTreasuryRejected struct {
	Phase         Phase
	ProposalIndex U32
	Amount        U128
	Topics        []Hash
}

// EventTreasuryBurnt is emitted when some of our funds have been burnt.
type EventTreasuryBurnt struct {
	Phase  Phase
	Burn   U128
	Topics []Hash
}

// EventTreasuryRollover is emitted when spending has finished; this is the amount that rolls over until next spend.
type EventTreasuryRollover struct {
	Phase           Phase
	BudgetRemaining U128
	Topics          []Hash
}

// EventTreasuryDeposit is emitted when some funds have been deposited.
type EventTreasuryDeposit struct {
	Phase     Phase
	Deposited U128
	Topics    []Hash
}

// EventTipsNewTip is emitted when a new tip suggestion has been opened.
type EventTipsNewTip struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventTipsTipClosing is emitted when a tip suggestion has reached threshold and is closing.
type EventTipsTipClosing struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

// EventTipsTipClosed is emitted when a tip suggestion has been closed.
type EventTipsTipClosed struct {
	Phase     Phase
	Hash      Hash
	AccountID AccountID
	Balance   U128
	Topics    []Hash
}

// EventTipsTipSlashed is emitted when a tip suggestion has been slashed.
type EventTipsTipSlashed struct {
	Phase     Phase
	Hash      Hash
	AccountID AccountID
	Balance   U128
	Topics    []Hash
}

// EventTransactionStorageStored is emitted when data is stored under a specific index.
type EventTransactionStorageStored struct {
	Phase  Phase
	Index  U32
	Topics []Hash
}

// EventTransactionStorageRenewed is emitted when data is renewed under a specific index.
type EventTransactionStorageRenewed struct {
	Phase  Phase
	Index  U32
	Topics []Hash
}

// EventTransactionStorageProofChecked is emitted when storage proof was successfully checked.
type EventTransactionStorageProofChecked struct {
	Phase  Phase
	Topics []Hash
}

type EventTransactionPaymentTransactionFeePaid struct {
	Phase     Phase
	Who       AccountID
	ActualFee U128
	Tip       U128
	Topics    []Hash
}

// EventTipsTipRetracted is emitted when a tip suggestion has been retracted.
type EventTipsTipRetracted struct {
	Phase  Phase
	Hash   Hash
	Topics []Hash
}

type BountyIndex U32

// EventBountiesBountyProposed is emitted for a new bounty proposal.
type EventBountiesBountyProposed struct {
	Phase         Phase
	ProposalIndex BountyIndex
	Topics        []Hash
}

// EventBountiesBountyRejected is emitted when a bounty proposal was rejected; funds were slashed.
type EventBountiesBountyRejected struct {
	Phase         Phase
	ProposalIndex BountyIndex
	Bond          U128
	Topics        []Hash
}

// EventBountiesBountyBecameActive is emitted when a bounty proposal is funded and became active
type EventBountiesBountyBecameActive struct {
	Phase  Phase
	Index  BountyIndex
	Topics []Hash
}

// EventBountiesBountyAwarded is emitted when a bounty is awarded to a beneficiary
type EventBountiesBountyAwarded struct {
	Phase       Phase
	Index       BountyIndex
	Beneficiary AccountID
	Topics      []Hash
}

// EventBountiesBountyClaimed is emitted when a bounty is claimed by beneficiary
type EventBountiesBountyClaimed struct {
	Phase       Phase
	Index       BountyIndex
	Payout      U128
	Beneficiary AccountID
	Topics      []Hash
}

// EventBountiesBountyCanceled is emitted when a bounty is cancelled.
type EventBountiesBountyCanceled struct {
	Phase  Phase
	Index  BountyIndex
	Topics []Hash
}

// EventBountiesBountyExtended is emitted when a bounty is extended.
type EventBountiesBountyExtended struct {
	Phase  Phase
	Index  BountyIndex
	Topics []Hash
}

// EventChildBountiesAdded is emitted when a child-bounty is added.
type EventChildBountiesAdded struct {
	Phase      Phase
	Index      BountyIndex
	ChildIndex BountyIndex
	Topics     []Hash
}

// EventChildBountiesAwarded is emitted when a child-bounty is awarded to a beneficiary.
type EventChildBountiesAwarded struct {
	Phase       Phase
	Index       BountyIndex
	ChildIndex  BountyIndex
	Beneficiary AccountID
	Topics      []Hash
}

// EventChildBountiesClaimed is emitted when a child-bounty is claimed by a beneficiary.
type EventChildBountiesClaimed struct {
	Phase       Phase
	Index       BountyIndex
	ChildIndex  BountyIndex
	Payout      U128
	Beneficiary AccountID
	Topics      []Hash
}

// EventChildBountiesCanceled is emitted when a child-bounty is canceled.
type EventChildBountiesCanceled struct {
	Phase      Phase
	Index      BountyIndex
	ChildIndex BountyIndex
	Topics     []Hash
}

// EventUniquesApprovalCancelled is emitted when an approval for a delegate account to transfer the instance of
// an asset class was cancelled by its owner
type EventUniquesApprovalCancelled struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Owner        AccountID
	Delegate     AccountID
	Topics       []Hash
}

// EventUniquesApprovedTransfer is emitted when an `instance` of an asset `class` has been approved by the `owner`
// for transfer by a `delegate`.
type EventUniquesApprovedTransfer struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Owner        AccountID
	Delegate     AccountID
	Topics       []Hash
}

// EventUniquesAssetStatusChanged is emitted when an asset `class` has had its attributes changed by the `Force` origin
type EventUniquesAssetStatusChanged struct {
	Phase        Phase
	CollectionID U64
	Topics       []Hash
}

// EventUniquesAttributeCleared is emitted when an attribute metadata has been cleared for an asset class or instance
type EventUniquesAttributeCleared struct {
	Phase        Phase
	CollectionID U64
	MaybeItem    Option[U128]
	Key          Bytes
	Topics       []Hash
}

// EventUniquesAttributeSet is emitted when a new attribute metadata has been set for an asset class or instance
type EventUniquesAttributeSet struct {
	Phase        Phase
	CollectionID U64
	MaybeItem    Option[U128]
	Key          Bytes
	Value        Bytes
	Topics       []Hash
}

// EventUniquesBurned is emitted when an asset `instance` was destroyed
type EventUniquesBurned struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Owner        AccountID
	Topics       []Hash
}

// EventUniquesClassFrozen is emitted when some asset `class` was frozen
type EventUniquesClassFrozen struct {
	Phase        Phase
	CollectionID U64
	Topics       []Hash
}

// EventUniquesClassMetadataCleared is emitted when metadata has been cleared for an asset class
type EventUniquesClassMetadataCleared struct {
	Phase        Phase
	CollectionID U64
	Topics       []Hash
}

// EventUniquesClassMetadataSet is emitted when new metadata has been set for an asset class
type EventUniquesClassMetadataSet struct {
	Phase        Phase
	CollectionID U64
	Data         Bytes
	IsFrozen     Bool
	Topics       []Hash
}

// EventUniquesClassThawed is emitted when some asset `class` was thawed
type EventUniquesClassThawed struct {
	Phase        Phase
	CollectionID U64
	Topics       []Hash
}

// EventUniquesCreated is emitted when an asset class was created
type EventUniquesCreated struct {
	Phase        Phase
	CollectionID U64
	Creator      AccountID
	Owner        AccountID
	Topics       []Hash
}

// EventUniquesDestroyed is emitted when an asset `class` was destroyed
type EventUniquesDestroyed struct {
	Phase        Phase
	CollectionID U64
	Topics       []Hash
}

// EventUniquesForceCreated is emitted when an asset class was force-created
type EventUniquesForceCreated struct {
	Phase        Phase
	CollectionID U64
	Owner        AccountID
	Topics       []Hash
}

// EventUniquesFrozen is emitted when some asset `instance` was frozen
type EventUniquesFrozen struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Topics       []Hash
}

// EventUniquesIssued is emitted when an asset instance was issued
type EventUniquesIssued struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Owner        AccountID
	Topics       []Hash
}

// EventUniquesMetadataCleared is emitted when metadata has been cleared for an asset instance
type EventUniquesMetadataCleared struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Topics       []Hash
}

// EventUniquesMetadataSet is emitted when metadata has been set for an asset instance
type EventUniquesMetadataSet struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Data         Bytes
	IsFrozen     Bool
	Topics       []Hash
}

// EventUniquesOwnerChanged is emitted when the owner changed
type EventUniquesOwnerChanged struct {
	Phase        Phase
	CollectionID U64
	NewOwner     AccountID
	Topics       []Hash
}

// EventUniquesRedeposited is emitted when metadata has been cleared for an asset instance
type EventUniquesRedeposited struct {
	Phase           Phase
	CollectionID    U64
	SuccessfulItems []U128
	Topics          []Hash
}

// EventUniquesTeamChanged is emitted when the management team changed
type EventUniquesTeamChanged struct {
	Phase        Phase
	CollectionID U64
	Issuer       AccountID
	Admin        AccountID
	Freezer      AccountID
	Topics       []Hash
}

// EventUniquesThawed is emitted when some asset instance was thawed
type EventUniquesThawed struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	Topics       []Hash
}

// EventUniquesTransferred is emitted when some asset instance was transferred
type EventUniquesTransferred struct {
	Phase        Phase
	CollectionID U64
	ItemID       U128
	From         AccountID
	To           AccountID
	Topics       []Hash
}

// EventUMPInvalidFormat is emitted when the upward message is invalid XCM.
type EventUMPInvalidFormat struct {
	Phase     Phase
	MessageID [32]U8
	Topics    []Hash
}

// EventUMPUnsupportedVersion is emitted when the upward message is unsupported version of XCM.
type EventUMPUnsupportedVersion struct {
	Phase     Phase
	MessageID [32]U8
	Topics    []Hash
}

// EventUMPExecutedUpward is emitted when the upward message executed with the given outcome.
type EventUMPExecutedUpward struct {
	Phase     Phase
	MessageID [32]U8
	Outcome   Outcome
	Topics    []Hash
}

// EventUMPWeightExhausted is emitted when the weight limit for handling upward messages was reached.
type EventUMPWeightExhausted struct {
	Phase     Phase
	MessageID [32]U8
	Remaining Weight
	Required  Weight
	Topics    []Hash
}

// EventUMPUpwardMessagesReceived is emitted when some upward messages have been received and will be processed.
type EventUMPUpwardMessagesReceived struct {
	Phase       Phase
	ParachainID ParachainID
	Count       U32
	Size        U32
	Topics      []Hash
}

// EventUMPOverweightEnqueued is emitted when the weight budget was exceeded for an individual upward message.
// This message can be later dispatched manually using `service_overweight` dispatchable using
// the assigned `overweight_index`.
type EventUMPOverweightEnqueued struct {
	Phase           Phase
	ParachainID     ParachainID
	MessageID       [32]U8
	OverweightIndex U64
	RequiredWeight  Weight
	Topics          []Hash
}

// EventUMPOverweightServiced is emitted when the upward message from the
// overweight queue was executed with the given actual weight used.
type EventUMPOverweightServiced struct {
	Phase           Phase
	OverweightIndex U64
	Used            Weight
	Topics          []Hash
}

// EventContractsInstantiated is emitted when a contract is deployed by address at the specified address
type EventContractsInstantiated struct {
	Phase    Phase
	Deployer AccountID
	Contract AccountID
	Topics   []Hash
}

// EventContractsTerminated The only way for a contract to be removed and emitting this event is by calling
// `seal_terminate`
type EventContractsTerminated struct {
	Phase       Phase
	Contract    AccountID
	Beneficiary AccountID
	Topics      []Hash
}

// EventConvictionVotingDelegated is emitted when an account has delegated their vote to another account.
type EventConvictionVotingDelegated struct {
	Phase  Phase
	Who    AccountID
	Target AccountID
	Topics []Hash
}

// EventConvictionVotingUndelegated is emitted when an account has delegated their vote to another account.
type EventConvictionVotingUndelegated struct {
	Phase  Phase
	Who    AccountID
	Target AccountID
	Topics []Hash
}

// EventContractsContractEmitted is emitted when a custom event emitted by the contract
type EventContractsContractEmitted struct {
	Phase    Phase
	Contract AccountID
	Data     Bytes
	Topics   []Hash
}

// EventContractsContractCodeUpdated is emitted when a contract's code was updated
type EventContractsContractCodeUpdated struct {
	Phase       Phase
	Contract    AccountID
	NewCodeHash Hash
	OldCodeHash Hash
	Topics      []Hash
}

type EventCollatorSelectionNewInvulnerables struct {
	Phase            Phase
	NewInvulnerables []AccountID
	Topics           []Hash
}

type EventCollatorSelectionNewDesiredCandidates struct {
	Phase                Phase
	NewDesiredCandidates U32
	Topics               []Hash
}

type EventCollatorSelectionNewCandidacyBond struct {
	Phase            Phase
	NewCandidacyBond U128
	Topics           []Hash
}

type EventCollatorSelectionCandidateAdded struct {
	Phase          Phase
	CandidateAdded AccountID
	Bond           U128
	Topics         []Hash
}

type EventCollatorSelectionCandidateRemoved struct {
	Phase            Phase
	CandidateRemoved AccountID
	Topics           []Hash
}

// EventContractsCodeRemoved is emitted when code with the specified hash was removed
type EventContractsCodeRemoved struct {
	Phase    Phase
	CodeHash Hash
	Topics   []Hash
}

// EventContractsCodeStored is emitted when code with the specified hash has been stored
type EventContractsCodeStored struct {
	Phase    Phase
	CodeHash Hash
	Topics   []Hash
}

// EventContractsScheduleUpdated is triggered when the current [schedule] is updated
type EventContractsScheduleUpdated struct {
	Phase    Phase
	Schedule U32
	Topics   []Hash
}

// EventContractsContractExecution is triggered when an event deposited upon execution of a contract from the account
type EventContractsContractExecution struct {
	Phase   Phase
	Account AccountID
	Data    Bytes
	Topics  []Hash
}

// EventUtilityBatchInterrupted is emitted when a batch of dispatches did not complete fully.
// Index of first failing dispatch given, as well as the error.
type EventUtilityBatchInterrupted struct {
	Phase         Phase
	Index         U32
	DispatchError DispatchError
	Topics        []Hash
}

// EventUtilityBatchCompleted is emitted when a batch of dispatches completed fully with no error.
type EventUtilityBatchCompleted struct {
	Phase  Phase
	Topics []Hash
}

// EventUtilityDispatchedAs is emitted when a call was dispatched
type EventUtilityDispatchedAs struct {
	Phase  Phase
	Index  U32
	Result DispatchResult
	Topics []Hash
}

// EventUtilityItemCompleted is emitted when a single item within a Batch of dispatches has completed with no error
type EventUtilityItemCompleted struct {
	Phase  Phase
	Topics []Hash
}

// EventUtilityNewMultisig is emitted when a new multisig operation has begun.
// First param is the account that is approving, second is the multisig account, third is hash of the call.
type EventMultisigNewMultisig struct {
	Phase    Phase
	Who, ID  AccountID
	CallHash Hash
	Topics   []Hash
}

// EventNftSalesForSale is emitted when an NFT is out for sale.
type EventNftSalesForSale struct {
	Phase      Phase
	ClassID    U64
	InstanceID U128
	Sale       Sale
	Topics     []Hash
}

// EventNftSalesRemoved is emitted when an NFT is removed.
type EventNftSalesRemoved struct {
	Phase      Phase
	ClassID    U64
	InstanceID U128
	Topics     []Hash
}

// EventNftSalesSold is emitted when an NFT is sold.
type EventNftSalesSold struct {
	Phase      Phase
	ClassID    U64
	InstanceID U128
	Sale       Sale
	Buyer      AccountID
	Topics     []Hash
}

// TimePoint is a global extrinsic index, formed as the extrinsic index within a block,
// together with that block's height.
type TimePoint struct {
	Height U32
	Index  U32
}

// TaskAddress holds the location of a scheduled task that can be used to remove it
type TaskAddress struct {
	When  U32
	Index U32
}

// EventUtility is emitted when a multisig operation has been approved by someone. First param is the account that is
// approving, third is the multisig account, fourth is hash of the call.
type EventMultisigApproval struct {
	Phase     Phase
	Who       AccountID
	TimePoint TimePoint
	ID        AccountID
	CallHash  Hash
	Topics    []Hash
}

// DispatchResult can be returned from dispatchable functions
type DispatchResult struct {
	Ok    bool
	Error DispatchError
}

func (d *DispatchResult) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.Ok = true
		return nil
	default:
		return decoder.Decode(&d.Error)
	}
}

func (d DispatchResult) Encode(encoder scale.Encoder) error {
	if d.Ok {
		return encoder.PushByte(0)
	}

	if err := encoder.PushByte(1); err != nil {
		return err
	}

	return encoder.Encode(d.Error)
}

// EventUtility is emitted when a multisig operation has been executed. First param is the account that is
// approving, third is the multisig account, fourth is hash of the call to be executed.
type EventMultisigExecuted struct {
	Phase     Phase
	Who       AccountID
	TimePoint TimePoint
	ID        AccountID
	CallHash  Hash
	Result    DispatchResult
	Topics    []Hash
}

// EventUtility is emitted when a multisig operation has been cancelled. First param is the account that is
// cancelling, third is the multisig account, fourth is hash of the call.
type EventMultisigCancelled struct {
	Phase     Phase
	Who       AccountID
	TimePoint TimePoint
	ID        AccountID
	CallHash  Hash
	Topics    []Hash
}
