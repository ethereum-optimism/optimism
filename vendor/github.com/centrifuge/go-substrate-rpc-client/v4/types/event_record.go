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
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/ethereum/go-ethereum/log"
)

// EventRecordsRaw is a raw record for a set of events, represented as the raw bytes. It exists since
// decoding of events can only be done with metadata, so events can't follow the static way of decoding
// other types do. It exposes functions to decode events using metadata and targets.
// Be careful using this in your own structs – it only works as the last value in a struct since it will consume the
// remainder of the encoded data. The reason for this is that it does not contain any length encoding, so it would
// not know where to stop.
type EventRecordsRaw []byte

// Encode implements encoding for Data, which just unwraps the bytes of Data
func (e EventRecordsRaw) Encode(encoder scale.Encoder) error {
	return encoder.Write(e)
}

// Decode implements decoding for Data, which just reads all the remaining bytes into Data
func (e *EventRecordsRaw) Decode(decoder scale.Decoder) error {
	for i := 0; true; i++ {
		b, err := decoder.ReadOneByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		*e = append((*e)[:i], b)
	}
	return nil
}

// EventRecords is a default set of possible event records that can be used as a target for
// `func (e EventRecordsRaw) Decode(...`
// Sources:
// https://github.com/polkadot-js/api/blob/master/packages/api-augment/src/substrate/events.ts
// https://github.com/polkadot-js/api/blob/master/packages/api-augment/src/polkadot/events.ts
//
//nolint:stylecheck,lll,revive
type EventRecords struct {
	Auctions_AuctionStarted     []EventAuctionsAuctionStarted     `test-gen-blockchain:"polkadot"`
	Auctions_AuctionClosed      []EventAuctionsAuctionClosed      `test-gen-blockchain:"polkadot"`
	Auctions_Reserved           []EventAuctionsReserved           `test-gen-blockchain:"polkadot"`
	Auctions_Unreserved         []EventAuctionsUnreserved         `test-gen-blockchain:"polkadot"`
	Auctions_ReserveConfiscated []EventAuctionsReserveConfiscated `test-gen-blockchain:"polkadot"`
	Auctions_BidAccepted        []EventAuctionsBidAccepted        `test-gen-blockchain:"polkadot"`
	Auctions_WinningOffset      []EventAuctionsWinningOffset      `test-gen-blockchain:"polkadot"`

	Assets_Created             []EventAssetCreated             `test-gen-skip:"true"`
	Assets_Issued              []EventAssetIssued              `test-gen-skip:"true"`
	Assets_Transferred         []EventAssetTransferred         `test-gen-skip:"true"`
	Assets_Burned              []EventAssetBurned              `test-gen-skip:"true"`
	Assets_TeamChanged         []EventAssetTeamChanged         `test-gen-skip:"true"`
	Assets_OwnerChanged        []EventAssetOwnerChanged        `test-gen-skip:"true"`
	Assets_Frozen              []EventAssetFrozen              `test-gen-skip:"true"`
	Assets_Thawed              []EventAssetThawed              `test-gen-skip:"true"`
	Assets_AssetFrozen         []EventAssetAssetFrozen         `test-gen-skip:"true"`
	Assets_AssetThawed         []EventAssetAssetThawed         `test-gen-skip:"true"`
	Assets_Destroyed           []EventAssetDestroyed           `test-gen-skip:"true"`
	Assets_ForceCreated        []EventAssetForceCreated        `test-gen-skip:"true"`
	Assets_MetadataSet         []EventAssetMetadataSet         `test-gen-skip:"true"`
	Assets_MetadataCleared     []EventAssetMetadataCleared     `test-gen-skip:"true"`
	Assets_ApprovedTransfer    []EventAssetApprovedTransfer    `test-gen-skip:"true"`
	Assets_ApprovalCancelled   []EventAssetApprovalCancelled   `test-gen-skip:"true"`
	Assets_TransferredApproved []EventAssetTransferredApproved `test-gen-skip:"true"`
	Assets_AssetStatusChanged  []EventAssetAssetStatusChanged  `test-gen-skip:"true"`

	BagsList_Rebagged []EventBagsListRebagged `test-gen-blockchain:"polkadot"`

	Balances_BalanceSet         []EventBalancesBalanceSet         `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Deposit            []EventBalancesDeposit            `test-gen-blockchain:"centrifuge-parachain"`
	Balances_DustLost           []EventBalancesDustLost           `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Endowed            []EventBalancesEndowed            `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Reserved           []EventBalancesReserved           `test-gen-blockchain:"centrifuge-parachain"`
	Balances_ReserveRepatriated []EventBalancesReserveRepatriated `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Slashed            []EventBalancesSlashed            `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Transfer           []EventBalancesTransfer           `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Unreserved         []EventBalancesUnreserved         `test-gen-blockchain:"centrifuge-parachain"`
	Balances_Withdraw           []EventBalancesWithdraw           `test-gen-blockchain:"centrifuge-parachain"`

	Bounties_BountyProposed     []EventBountiesBountyProposed     `test-gen-blockchain:"polkadot"`
	Bounties_BountyRejected     []EventBountiesBountyRejected     `test-gen-blockchain:"polkadot"`
	Bounties_BountyBecameActive []EventBountiesBountyBecameActive `test-gen-blockchain:"polkadot"`
	Bounties_BountyAwarded      []EventBountiesBountyAwarded      `test-gen-blockchain:"polkadot"`
	Bounties_BountyClaimed      []EventBountiesBountyClaimed      `test-gen-blockchain:"polkadot"`
	Bounties_BountyCanceled     []EventBountiesBountyCanceled     `test-gen-blockchain:"polkadot"`
	Bounties_BountyExtended     []EventBountiesBountyExtended     `test-gen-blockchain:"polkadot"`

	ChildBounties_Added    []EventChildBountiesAdded    `test-gen-skip:"true"`
	ChildBounties_Awarded  []EventChildBountiesAwarded  `test-gen-skip:"true"`
	ChildBounties_Claimed  []EventChildBountiesClaimed  `test-gen-skip:"true"`
	ChildBounties_Canceled []EventChildBountiesCanceled `test-gen-skip:"true"`

	Claims_Claimed []EventClaimsClaimed `test-gen-blockchain:"polkadot"`

	CollatorSelection_NewInvulnerables     []EventCollatorSelectionNewInvulnerables     `test-gen-blockchain:"altair"`
	CollatorSelection_NewDesiredCandidates []EventCollatorSelectionNewDesiredCandidates `test-gen-blockchain:"altair"`
	CollatorSelection_NewCandidacyBond     []EventCollatorSelectionNewCandidacyBond     `test-gen-blockchain:"altair"`
	CollatorSelection_CandidateAdded       []EventCollatorSelectionCandidateAdded       `test-gen-blockchain:"altair"`
	CollatorSelection_CandidateRemoved     []EventCollatorSelectionCandidateRemoved     `test-gen-blockchain:"altair"`

	Contracts_CodeRemoved         []EventContractsCodeRemoved         `test-gen-skip:"true"`
	Contracts_CodeStored          []EventContractsCodeStored          `test-gen-skip:"true"`
	Contracts_ContractCodeUpdated []EventContractsContractCodeUpdated `test-gen-skip:"true"`
	Contracts_ContractEmitted     []EventContractsContractEmitted     `test-gen-skip:"true"`
	Contracts_Instantiated        []EventContractsInstantiated        `test-gen-skip:"true"`
	Contracts_Terminated          []EventContractsTerminated          `test-gen-skip:"true"`

	ConvictionVoting_Delegated   []EventConvictionVotingDelegated   `test-gen-skip:"true"`
	ConvictionVoting_Undelegated []EventConvictionVotingUndelegated `test-gen-skip:"true"`

	Council_Approved       []EventCouncilApproved       `test-gen-blockchain:"centrifuge-parachain"`
	Council_Closed         []EventCouncilClosed         `test-gen-blockchain:"centrifuge-parachain"`
	Council_Disapproved    []EventCouncilDisapproved    `test-gen-blockchain:"centrifuge-parachain"`
	Council_Executed       []EventCouncilExecuted       `test-gen-blockchain:"centrifuge-parachain"`
	Council_MemberExecuted []EventCouncilMemberExecuted `test-gen-blockchain:"centrifuge-parachain"`
	Council_Proposed       []EventCouncilProposed       `test-gen-blockchain:"centrifuge-parachain"`
	Council_Voted          []EventCouncilVoted          `test-gen-blockchain:"centrifuge-parachain"`

	Crowdloan_Created           []EventCrowdloanCreated           `test-gen-blockchain:"polkadot"`
	Crowdloan_Contributed       []EventCrowdloanContributed       `test-gen-blockchain:"polkadot"`
	Crowdloan_Withdrew          []EventCrowdloanWithdrew          `test-gen-blockchain:"polkadot"`
	Crowdloan_PartiallyRefunded []EventCrowdloanPartiallyRefunded `test-gen-blockchain:"polkadot"`
	Crowdloan_AllRefunded       []EventCrowdloanAllRefunded       `test-gen-blockchain:"polkadot"`
	Crowdloan_Dissolved         []EventCrowdloanDissolved         `test-gen-blockchain:"polkadot"`
	Crowdloan_HandleBidResult   []EventCrowdloanHandleBidResult   `test-gen-blockchain:"polkadot"`
	Crowdloan_Edited            []EventCrowdloanEdited            `test-gen-blockchain:"polkadot"`
	Crowdloan_MemoUpdated       []EventCrowdloanMemoUpdated       `test-gen-blockchain:"polkadot"`
	Crowdloan_AddedToNewRaise   []EventCrowdloanAddedToNewRaise   `test-gen-blockchain:"polkadot"`

	Democracy_Blacklisted     []EventDemocracyBlacklisted     `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Cancelled       []EventDemocracyCancelled       `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Delegated       []EventDemocracyDelegated       `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Executed        []EventDemocracyExecuted        `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_ExternalTabled  []EventDemocracyExternalTabled  `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_NotPassed       []EventDemocracyNotPassed       `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Passed          []EventDemocracyPassed          `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_PreimageInvalid []EventDemocracyPreimageInvalid `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_PreimageMissing []EventDemocracyPreimageMissing `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_PreimageNoted   []EventDemocracyPreimageNoted   `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_PreimageReaped  []EventDemocracyPreimageReaped  `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_PreimageUsed    []EventDemocracyPreimageUsed    `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Proposed        []EventDemocracyProposed        `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Seconded        []EventDemocracySeconded        `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Started         []EventDemocracyStarted         `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Tabled          []EventDemocracyTabled          `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Undelegated     []EventDemocracyUndelegated     `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Vetoed          []EventDemocracyVetoed          `test-gen-blockchain:"centrifuge-parachain"`
	Democracy_Voted           []EventDemocracyVoted           `test-gen-blockchain:"centrifuge-parachain"`

	ElectionProviderMultiPhase_SolutionStored       []EventElectionProviderMultiPhaseSolutionStored       `test-gen-blockchain:"polkadot"`
	ElectionProviderMultiPhase_ElectionFinalized    []EventElectionProviderMultiPhaseElectionFinalized    `test-gen-blockchain:"polkadot"`
	ElectionProviderMultiPhase_Rewarded             []EventElectionProviderMultiPhaseRewarded             `test-gen-blockchain:"polkadot"`
	ElectionProviderMultiPhase_Slashed              []EventElectionProviderMultiPhaseSlashed              `test-gen-blockchain:"polkadot"`
	ElectionProviderMultiPhase_SignedPhaseStarted   []EventElectionProviderMultiPhaseSignedPhaseStarted   `test-gen-blockchain:"polkadot"`
	ElectionProviderMultiPhase_UnsignedPhaseStarted []EventElectionProviderMultiPhaseUnsignedPhaseStarted `test-gen-blockchain:"polkadot"`

	Elections_CandidateSlashed  []EventElectionsCandidateSlashed  `test-gen-blockchain:"altair"`
	Elections_ElectionError     []EventElectionsElectionError     `test-gen-blockchain:"altair"`
	Elections_EmptyTerm         []EventElectionsEmptyTerm         `test-gen-blockchain:"altair"`
	Elections_MemberKicked      []EventElectionsMemberKicked      `test-gen-blockchain:"altair"`
	Elections_NewTerm           []EventElectionsNewTerm           `test-gen-blockchain:"altair"`
	Elections_Renounced         []EventElectionsRenounced         `test-gen-blockchain:"altair"`
	Elections_SeatHolderSlashed []EventElectionsSeatHolderSlashed `test-gen-blockchain:"altair"`

	Gilt_BidPlaced    []EventGiltBidPlaced    `test-gen-skip:"true"`
	Gilt_BidRetracted []EventGiltBidRetracted `test-gen-skip:"true"`
	Gilt_GiltIssued   []EventGiltGiltIssued   `test-gen-skip:"true"`
	Gilt_GiltThawed   []EventGiltGiltThawed   `test-gen-skip:"true"`

	Grandpa_NewAuthorities []EventGrandpaNewAuthorities `test-gen-blockchain:"polkadot"`
	Grandpa_Paused         []EventGrandpaPaused         `test-gen-blockchain:"polkadot"`
	Grandpa_Resumed        []EventGrandpaResumed        `test-gen-blockchain:"polkadot"`

	Hrmp_OpenChannelRequested []EventHRMPOpenChannelRequested `test-gen-blockchain:"polkadot"`
	Hrmp_OpenChannelCanceled  []EventHRMPOpenChannelCanceled  `test-gen-blockchain:"polkadot"`
	Hrmp_OpenChannelAccepted  []EventHRMPOpenChannelAccepted  `test-gen-blockchain:"polkadot"`
	Hrmp_ChannelClosed        []EventHRMPChannelClosed        `test-gen-blockchain:"polkadot"`

	Identity_IdentityCleared      []EventIdentityCleared              `test-gen-blockchain:"centrifuge-parachain"`
	Identity_IdentityKilled       []EventIdentityKilled               `test-gen-blockchain:"centrifuge-parachain"`
	Identity_IdentitySet          []EventIdentitySet                  `test-gen-blockchain:"centrifuge-parachain"`
	Identity_JudgementGiven       []EventIdentityJudgementGiven       `test-gen-blockchain:"centrifuge-parachain"`
	Identity_JudgementRequested   []EventIdentityJudgementRequested   `test-gen-blockchain:"centrifuge-parachain"`
	Identity_JudgementUnrequested []EventIdentityJudgementUnrequested `test-gen-blockchain:"centrifuge-parachain"`
	Identity_RegistrarAdded       []EventIdentityRegistrarAdded       `test-gen-blockchain:"centrifuge-parachain"`
	Identity_SubIdentityAdded     []EventIdentitySubIdentityAdded     `test-gen-blockchain:"centrifuge-parachain"`
	Identity_SubIdentityRemoved   []EventIdentitySubIdentityRemoved   `test-gen-blockchain:"centrifuge-parachain"`
	Identity_SubIdentityRevoked   []EventIdentitySubIdentityRevoked   `test-gen-blockchain:"centrifuge-parachain"`

	ImOnline_AllGood           []EventImOnlineAllGood           `test-gen-blockchain:"polkadot"`
	ImOnline_HeartbeatReceived []EventImOnlineHeartbeatReceived `test-gen-blockchain:"polkadot"`
	ImOnline_SomeOffline       []EventImOnlineSomeOffline       `test-gen-blockchain:"polkadot"`

	Indices_IndexAssigned []EventIndicesIndexAssigned `test-gen-blockchain:"polkadot"`
	Indices_IndexFreed    []EventIndicesIndexFreed    `test-gen-blockchain:"polkadot"`
	Indices_IndexFrozen   []EventIndicesIndexFrozen   `test-gen-blockchain:"polkadot"`

	Lottery_LotteryStarted []EventLotteryLotteryStarted `test-gen-skip:"true"`
	Lottery_CallsUpdated   []EventLotteryCallsUpdated   `test-gen-skip:"true"`
	Lottery_Winner         []EventLotteryWinner         `test-gen-skip:"true"`
	Lottery_TicketBought   []EventLotteryTicketBought   `test-gen-skip:"true"`

	Multisig_MultisigApproval  []EventMultisigApproval    `test-gen-blockchain:"altair"`
	Multisig_MultisigCancelled []EventMultisigCancelled   `test-gen-blockchain:"altair"`
	Multisig_MultisigExecuted  []EventMultisigExecuted    `test-gen-blockchain:"altair"`
	Multisig_NewMultisig       []EventMultisigNewMultisig `test-gen-blockchain:"altair"`

	NftSales_ForSale []EventNftSalesForSale `test-gen-blockchain:"altair"`
	NftSales_Removed []EventNftSalesRemoved `test-gen-blockchain:"altair"`
	NftSales_Sold    []EventNftSalesSold    `test-gen-blockchain:"altair"`

	Offences_Offence []EventOffencesOffence `test-gen-blockchain:"polkadot"`

	Paras_CurrentCodeUpdated   []EventParasCurrentCodeUpdated   `test-gen-blockchain:"polkadot"`
	Paras_CurrentHeadUpdated   []EventParasCurrentHeadUpdated   `test-gen-blockchain:"polkadot"`
	Paras_CodeUpgradeScheduled []EventParasCodeUpgradeScheduled `test-gen-blockchain:"polkadot"`
	Paras_NewHeadNoted         []EventParasNewHeadNoted         `test-gen-blockchain:"polkadot"`
	Paras_ActionQueued         []EventParasActionQueued         `test-gen-blockchain:"polkadot"`
	Paras_PvfCheckStarted      []EventParasPvfCheckStarted      `test-gen-blockchain:"polkadot"`
	Paras_PvfCheckAccepted     []EventParasPvfCheckAccepted     `test-gen-blockchain:"polkadot"`
	Paras_PvfCheckRejected     []EventParasPvfCheckRejected     `test-gen-blockchain:"polkadot"`

	ParasDisputes_DisputeInitiated []EventParasDisputesDisputeInitiated `test-gen-skip:"true"`
	ParasDisputes_DisputeConcluded []EventParasDisputesDisputeConcluded `test-gen-skip:"true"`
	ParasDisputes_DisputeTimedOut  []EventParasDisputesDisputeTimedOut  `test-gen-skip:"true"`
	ParasDisputes_Revert           []EventParasDisputesRevert           `test-gen-skip:"true"`

	ParaInclusion_CandidateBacked   []EventParaInclusionCandidateBacked   `test-gen-blockchain:"polkadot"`
	ParaInclusion_CandidateIncluded []EventParaInclusionCandidateIncluded `test-gen-blockchain:"polkadot"`
	ParaInclusion_CandidateTimedOut []EventParaInclusionCandidateTimedOut `test-gen-blockchain:"polkadot"`

	ParachainSystem_ValidationFunctionStored    []EventParachainSystemValidationFunctionStored    `test-gen-blockchain:"centrifuge-parachain"`
	ParachainSystem_ValidationFunctionApplied   []EventParachainSystemValidationFunctionApplied   `test-gen-blockchain:"centrifuge-parachain"`
	ParachainSystem_ValidationFunctionDiscarded []EventParachainSystemValidationFunctionDiscarded `test-gen-blockchain:"centrifuge-parachain"`
	ParachainSystem_UpgradeAuthorized           []EventParachainSystemUpgradeAuthorized           `test-gen-blockchain:"centrifuge-parachain"`
	ParachainSystem_DownwardMessagesReceived    []EventParachainSystemDownwardMessagesReceived    `test-gen-blockchain:"centrifuge-parachain"`
	ParachainSystem_DownwardMessagesProcessed   []EventParachainSystemDownwardMessagesProcessed   `test-gen-blockchain:"centrifuge-parachain"`

	Preimage_Cleared   []EventPreimageCleared   `test-gen-skip:"true"`
	Preimage_Noted     []EventPreimageNoted     `test-gen-skip:"true"`
	Preimage_Requested []EventPreimageRequested `test-gen-skip:"true"`

	Proxy_Announced     []EventProxyAnnounced     `test-gen-blockchain:"centrifuge-parachain"`
	Proxy_PureCreated   []EventProxyPureCreated   `test-gen-blockchain:"centrifuge-parachain"`
	Proxy_ProxyAdded    []EventProxyProxyAdded    `test-gen-blockchain:"centrifuge-parachain"`
	Proxy_ProxyExecuted []EventProxyProxyExecuted `test-gen-blockchain:"centrifuge-parachain"`
	Proxy_ProxyRemoved  []EventProxyProxyRemoved  `test-gen-blockchain:"centrifuge-parachain"`

	Recovery_AccountRecovered  []EventRecoveryAccountRecovered `test-gen-skip:"true"`
	Recovery_RecoveryClosed    []EventRecoveryClosed           `test-gen-skip:"true"`
	Recovery_RecoveryCreated   []EventRecoveryCreated          `test-gen-skip:"true"`
	Recovery_RecoveryInitiated []EventRecoveryInitiated        `test-gen-skip:"true"`
	Recovery_RecoveryRemoved   []EventRecoveryRemoved          `test-gen-skip:"true"`
	Recovery_RecoveryVouched   []EventRecoveryVouched          `test-gen-skip:"true"`

	Registrar_Registered   []EventRegistrarRegistered   `test-gen-skip:"true"`
	Registrar_Deregistered []EventRegistrarDeregistered `test-gen-skip:"true"`
	Registrar_Reserved     []EventRegistrarReserved     `test-gen-skip:"true"`

	Referenda_Submitted               []EventReferendaSubmitted               `test-gen-skip:"true"`
	Referenda_DecisionDepositPlaced   []EventReferendaDecisionDepositPlaced   `test-gen-skip:"true"`
	Referenda_DecisionDepositRefunded []EventReferendaDecisionDepositRefunded `test-gen-skip:"true"`
	Referenda_DepositSlashed          []EventReferendaDecisionSlashed         `test-gen-skip:"true"`
	Referenda_DecisionStarted         []EventReferendaDecisionStarted         `test-gen-skip:"true"`
	Referenda_ConfirmStarted          []EventReferendaConfirmStarted          `test-gen-skip:"true"`
	Referenda_ConfirmAborted          []EventReferendaConfirmAborted          `test-gen-skip:"true"`
	Referenda_Confirmed               []EventReferendaConfirmed               `test-gen-skip:"true"`
	Referenda_Approved                []EventReferendaApproved                `test-gen-skip:"true"`
	Referenda_Rejected                []EventReferendaRejected                `test-gen-skip:"true"`
	Referenda_TimedOut                []EventReferendaTimedOut                `test-gen-skip:"true"`
	Referenda_Cancelled               []EventReferendaCancelled               `test-gen-skip:"true"`
	Referenda_Killed                  []EventReferendaKilled                  `test-gen-skip:"true"`

	Scheduler_CallLookupFailed []EventSchedulerCallLookupFailed `test-gen-blockchain:"polkadot"`
	Scheduler_Canceled         []EventSchedulerCanceled         `test-gen-blockchain:"polkadot"`
	Scheduler_Dispatched       []EventSchedulerDispatched       `test-gen-blockchain:"polkadot"`
	Scheduler_Scheduled        []EventSchedulerScheduled        `test-gen-blockchain:"polkadot"`

	Session_NewSession []EventSessionNewSession `test-gen-blockchain:"centrifuge-parachain"`

	Slots_NewLeasePeriod []EventSlotsNewLeasePeriod `test-gen-blockchain:"polkadot"`
	Slots_Leased         []EventSlotsLeased         `test-gen-blockchain:"polkadot"`

	Society_AutoUnbid                []EventSocietyAutoUnbid                `test-gen-skip:"true"`
	Society_Bid                      []EventSocietyBid                      `test-gen-skip:"true"`
	Society_CandidateSuspended       []EventSocietyCandidateSuspended       `test-gen-skip:"true"`
	Society_Challenged               []EventSocietyChallenged               `test-gen-skip:"true"`
	Society_DefenderVote             []EventSocietyDefenderVote             `test-gen-skip:"true"`
	Society_Deposit                  []EventSocietyDeposit                  `test-gen-skip:"true"`
	Society_Founded                  []EventSocietyFounded                  `test-gen-skip:"true"`
	Society_Inducted                 []EventSocietyInducted                 `test-gen-skip:"true"`
	Society_MemberSuspended          []EventSocietyMemberSuspended          `test-gen-skip:"true"`
	Society_NewMaxMembers            []EventSocietyNewMaxMembers            `test-gen-skip:"true"`
	Society_SuspendedMemberJudgement []EventSocietySuspendedMemberJudgement `test-gen-skip:"true"`
	Society_Unbid                    []EventSocietyUnbid                    `test-gen-skip:"true"`
	Society_Unfounded                []EventSocietyUnfounded                `test-gen-skip:"true"`
	Society_Unvouch                  []EventSocietyUnvouch                  `test-gen-skip:"true"`
	Society_Vote                     []EventSocietyVote                     `test-gen-skip:"true"`
	Society_Vouch                    []EventSocietyVouch                    `test-gen-skip:"true"`

	Staking_Bonded                     []EventStakingBonded                     `test-gen-blockchain:"polkadot"`
	Staking_Chilled                    []EventStakingChilled                    `test-gen-blockchain:"polkadot"`
	Staking_EraPaid                    []EventStakingEraPaid                    `test-gen-blockchain:"polkadot"`
	Staking_Kicked                     []EventStakingKicked                     `test-gen-blockchain:"polkadot"`
	Staking_OldSlashingReportDiscarded []EventStakingOldSlashingReportDiscarded `test-gen-blockchain:"polkadot"`
	Staking_PayoutStarted              []EventStakingPayoutStarted              `test-gen-blockchain:"polkadot"`
	Staking_Rewarded                   []EventStakingRewarded                   `test-gen-blockchain:"polkadot"`
	Staking_Slashed                    []EventStakingSlashed                    `test-gen-blockchain:"polkadot"`
	Staking_StakersElected             []EventStakingStakersElected             `test-gen-blockchain:"polkadot"`
	Staking_StakingElectionFailed      []EventStakingStakingElectionFailed      `test-gen-blockchain:"polkadot"`
	Staking_Unbonded                   []EventStakingUnbonded                   `test-gen-blockchain:"polkadot"`
	Staking_Withdrawn                  []EventStakingWithdrawn                  `test-gen-blockchain:"polkadot"`

	StateTrieMigration_Migrated              []EventStateTrieMigrationMigrated              `test-gen-skip:"true"`
	StateTrieMigration_Slashed               []EventStateTrieMigrationSlashed               `test-gen-skip:"true"`
	StateTrieMigration_AutoMigrationFinished []EventStateTrieMigrationAutoMigrationFinished `test-gen-skip:"true"`
	StateTrieMigration_Halted                []EventStateTrieMigrationHalted                `test-gen-skip:"true"`

	Sudo_KeyChanged []EventSudoKeyChanged `test-gen-skip:"true"`
	Sudo_Sudid      []EventSudoSudid      `test-gen-skip:"true"`
	Sudo_SudoAsDone []EventSudoAsDone     `test-gen-skip:"true"`

	System_CodeUpdated      []EventSystemCodeUpdated      `test-gen-blockchain:"centrifuge-parachain"`
	System_ExtrinsicFailed  []EventSystemExtrinsicFailed  `test-gen-blockchain:"centrifuge-parachain"`
	System_ExtrinsicSuccess []EventSystemExtrinsicSuccess `test-gen-blockchain:"centrifuge-parachain"`
	System_KilledAccount    []EventSystemKilledAccount    `test-gen-blockchain:"centrifuge-parachain"`
	System_NewAccount       []EventSystemNewAccount       `test-gen-blockchain:"centrifuge-parachain"`
	System_Remarked         []EventSystemRemarked         `test-gen-blockchain:"centrifuge-parachain"`

	TechnicalCommittee_Approved       []EventTechnicalCommitteeApproved       `test-gen-blockchain:"polkadot"`
	TechnicalCommittee_Closed         []EventTechnicalCommitteeClosed         `test-gen-blockchain:"polkadot"`
	TechnicalCommittee_Disapproved    []EventTechnicalCommitteeDisapproved    `test-gen-blockchain:"polkadot"`
	TechnicalCommittee_Executed       []EventTechnicalCommitteeExecuted       `test-gen-blockchain:"polkadot"`
	TechnicalCommittee_MemberExecuted []EventTechnicalCommitteeMemberExecuted `test-gen-blockchain:"polkadot"`
	TechnicalCommittee_Proposed       []EventTechnicalCommitteeProposed       `test-gen-blockchain:"polkadot"`
	TechnicalCommittee_Voted          []EventTechnicalCommitteeVoted          `test-gen-blockchain:"polkadot"`

	TechnicalMembership_Dummy          []EventTechnicalMembershipDummy          `test-gen-blockchain:"polkadot"`
	TechnicalMembership_KeyChanged     []EventTechnicalMembershipKeyChanged     `test-gen-blockchain:"polkadot"`
	TechnicalMembership_MemberAdded    []EventTechnicalMembershipMemberAdded    `test-gen-blockchain:"polkadot"`
	TechnicalMembership_MemberRemoved  []EventTechnicalMembershipMemberRemoved  `test-gen-blockchain:"polkadot"`
	TechnicalMembership_MembersReset   []EventTechnicalMembershipMembersReset   `test-gen-blockchain:"polkadot"`
	TechnicalMembership_MembersSwapped []EventTechnicalMembershipMembersSwapped `test-gen-blockchain:"polkadot"`

	Tips_NewTip       []EventTipsNewTip       `test-gen-blockchain:"polkadot"`
	Tips_TipClosed    []EventTipsTipClosed    `test-gen-blockchain:"polkadot"`
	Tips_TipClosing   []EventTipsTipClosing   `test-gen-blockchain:"polkadot"`
	Tips_TipRetracted []EventTipsTipRetracted `test-gen-blockchain:"polkadot"`
	Tips_TipSlashed   []EventTipsTipSlashed   `test-gen-blockchain:"polkadot"`

	TransactionStorage_Stored       []EventTransactionStorageStored       `test-gen-skip:"true"`
	TransactionStorage_Renewed      []EventTransactionStorageRenewed      `test-gen-skip:"true"`
	TransactionStorage_ProofChecked []EventTransactionStorageProofChecked `test-gen-skip:"true"`

	TransactionPayment_TransactionFeePaid []EventTransactionPaymentTransactionFeePaid `test-gen-blockchain:"westend"`

	Treasury_Awarded  []EventTreasuryAwarded  `test-gen-blockchain:"altair"`
	Treasury_Burnt    []EventTreasuryBurnt    `test-gen-blockchain:"altair"`
	Treasury_Deposit  []EventTreasuryDeposit  `test-gen-blockchain:"altair"`
	Treasury_Proposed []EventTreasuryProposed `test-gen-blockchain:"altair"`
	Treasury_Rejected []EventTreasuryRejected `test-gen-blockchain:"altair"`
	Treasury_Rollover []EventTreasuryRollover `test-gen-blockchain:"altair"`
	Treasury_Spending []EventTreasurySpending `test-gen-blockchain:"altair"`

	Uniques_ApprovalCancelled    []EventUniquesApprovalCancelled    `test-gen-blockchain:"altair"`
	Uniques_ApprovedTransfer     []EventUniquesApprovedTransfer     `test-gen-blockchain:"altair"`
	Uniques_AssetStatusChanged   []EventUniquesAssetStatusChanged   `test-gen-blockchain:"altair"`
	Uniques_AttributeCleared     []EventUniquesAttributeCleared     `test-gen-blockchain:"altair"`
	Uniques_AttributeSet         []EventUniquesAttributeSet         `test-gen-blockchain:"altair"`
	Uniques_Burned               []EventUniquesBurned               `test-gen-blockchain:"altair"`
	Uniques_ClassFrozen          []EventUniquesClassFrozen          `test-gen-blockchain:"altair"`
	Uniques_ClassMetadataCleared []EventUniquesClassMetadataCleared `test-gen-blockchain:"altair"`
	Uniques_ClassMetadataSet     []EventUniquesClassMetadataSet     `test-gen-blockchain:"altair"`
	Uniques_ClassThawed          []EventUniquesClassThawed          `test-gen-blockchain:"altair"`
	Uniques_Created              []EventUniquesCreated              `test-gen-blockchain:"altair"`
	Uniques_Destroyed            []EventUniquesDestroyed            `test-gen-blockchain:"altair"`
	Uniques_ForceCreated         []EventUniquesForceCreated         `test-gen-blockchain:"altair"`
	Uniques_Frozen               []EventUniquesFrozen               `test-gen-blockchain:"altair"`
	Uniques_Issued               []EventUniquesIssued               `test-gen-blockchain:"altair"`
	Uniques_MetadataCleared      []EventUniquesMetadataCleared      `test-gen-blockchain:"altair"`
	Uniques_MetadataSet          []EventUniquesMetadataSet          `test-gen-blockchain:"altair"`
	Uniques_OwnerChanged         []EventUniquesOwnerChanged         `test-gen-blockchain:"altair"`
	Uniques_Redeposited          []EventUniquesRedeposited          `test-gen-blockchain:"altair"`
	Uniques_TeamChanged          []EventUniquesTeamChanged          `test-gen-blockchain:"altair"`
	Uniques_Thawed               []EventUniquesThawed               `test-gen-blockchain:"altair"`
	Uniques_Transferred          []EventUniquesTransferred          `test-gen-blockchain:"altair"`

	Ump_InvalidFormat          []EventUMPInvalidFormat          `test-gen-blockchain:"polkadot"`
	Ump_UnsupportedVersion     []EventUMPUnsupportedVersion     `test-gen-blockchain:"polkadot"`
	Ump_ExecutedUpward         []EventUMPExecutedUpward         `test-gen-blockchain:"polkadot"`
	Ump_WeightExhausted        []EventUMPWeightExhausted        `test-gen-blockchain:"polkadot"`
	Ump_UpwardMessagesReceived []EventUMPUpwardMessagesReceived `test-gen-blockchain:"polkadot"`
	Ump_OverweightEnqueued     []EventUMPOverweightEnqueued     `test-gen-blockchain:"polkadot"`
	Ump_OverweightServiced     []EventUMPOverweightServiced     `test-gen-blockchain:"polkadot"`

	Utility_BatchCompleted   []EventUtilityBatchCompleted   `test-gen-blockchain:"centrifuge-parachain"`
	Utility_BatchInterrupted []EventUtilityBatchInterrupted `test-gen-blockchain:"centrifuge-parachain"`
	Utility_DispatchedAs     []EventUtilityBatchInterrupted `test-gen-blockchain:"centrifuge-parachain"`
	Utility_ItemCompleted    []EventUtilityItemCompleted    `test-gen-blockchain:"centrifuge-parachain"`

	Vesting_VestingCompleted []EventVestingVestingCompleted `test-gen-blockchain:"centrifuge-parachain"`
	Vesting_VestingUpdated   []EventVestingVestingUpdated   `test-gen-blockchain:"centrifuge-parachain"`

	VoterList_Rebagged     []EventVoterListRebagged     `test-gen-blockchain:"polkadot"`
	VoterList_ScoreUpdated []EventVoterListScoreUpdated `test-gen-blockchain:"polkadot"`

	Whitelist_CallWhitelisted           []EventWhitelistCallWhitelisted        `test-gen-skip:"true"`
	Whitelist_WhitelistedCallRemoved    []EventWhitelistWhitelistedCallRemoved `test-gen-skip:"true"`
	Whitelist_WhitelistedCallDispatched []EventWhitelistWhitelistedCallRemoved `test-gen-skip:"true"`

	XcmPallet_Attempted                 []EventXcmPalletAttempted                 `test-gen-blockchain:"polkadot"`
	XcmPallet_Sent                      []EventXcmPalletSent                      `test-gen-blockchain:"polkadot"`
	XcmPallet_UnexpectedResponse        []EventXcmPalletUnexpectedResponse        `test-gen-blockchain:"polkadot"`
	XcmPallet_ResponseReady             []EventXcmPalletResponseReady             `test-gen-blockchain:"polkadot"`
	XcmPallet_Notified                  []EventXcmPalletNotified                  `test-gen-blockchain:"polkadot"`
	XcmPallet_NotifyOverweight          []EventXcmPalletNotifyOverweight          `test-gen-blockchain:"polkadot"`
	XcmPallet_NotifyDispatchError       []EventXcmPalletNotifyDispatchError       `test-gen-blockchain:"polkadot"`
	XcmPallet_NotifyDecodeFailed        []EventXcmPalletNotifyDecodeFailed        `test-gen-blockchain:"polkadot"`
	XcmPallet_InvalidResponder          []EventXcmPalletInvalidResponder          `test-gen-blockchain:"polkadot"`
	XcmPallet_InvalidResponderVersion   []EventXcmPalletInvalidResponderVersion   `test-gen-blockchain:"polkadot"`
	XcmPallet_ResponseTaken             []EventXcmPalletResponseTaken             `test-gen-blockchain:"polkadot"`
	XcmPallet_AssetsTrapped             []EventXcmPalletAssetsTrapped             `test-gen-blockchain:"polkadot"`
	XcmPallet_VersionChangeNotified     []EventXcmPalletVersionChangeNotified     `test-gen-blockchain:"polkadot"`
	XcmPallet_SupportedVersionChanged   []EventXcmPalletSupportedVersionChanged   `test-gen-blockchain:"polkadot"`
	XcmPallet_NotifyTargetSendFail      []EventXcmPalletNotifyTargetSendFail      `test-gen-blockchain:"polkadot"`
	XcmPallet_NotifyTargetMigrationFail []EventXcmPalletNotifyTargetMigrationFail `test-gen-blockchain:"polkadot"`
}

// DecodeEventRecords decodes the events records from an EventRecordRaw into a target t using the given Metadata m
// If this method returns an error like `unable to decode Phase for event #x: EOF`, it is likely that you have defined
// a custom event record with a wrong type. For example your custom event record has a field with a length prefixed
// type, such as types.Bytes, where your event in reallity contains a fixed width type, such as a types.U32.
func (e EventRecordsRaw) DecodeEventRecords(m *Metadata, t interface{}) error { //nolint:funlen
	log.Debug(fmt.Sprintf("will decode event records from raw hex: %#x", e))

	// ensure t is a pointer
	ttyp := reflect.TypeOf(t)
	if ttyp.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer, but is " + fmt.Sprint(ttyp))
	}
	// ensure t is not a nil pointer
	tval := reflect.ValueOf(t)
	if tval.IsNil() {
		return errors.New("target is a nil pointer")
	}
	val := tval.Elem()
	typ := val.Type()
	// ensure val can be set
	if !val.CanSet() {
		return fmt.Errorf("unsettable value %v", typ)
	}
	// ensure val points to a struct
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct, but is " + fmt.Sprint(typ))
	}

	decoder := scale.NewDecoder(bytes.NewReader(e))

	// determine number of events
	n, err := decoder.DecodeUintCompact()
	if err != nil {
		return err
	}

	log.Debug(fmt.Sprintf("found %v events", n))

	// iterate over events
	for i := uint64(0); i < n.Uint64(); i++ {
		log.Debug(fmt.Sprintf("decoding event #%v", i))

		// decode Phase
		phase := Phase{}
		err := decoder.Decode(&phase)
		if err != nil {
			return fmt.Errorf("unable to decode Phase for event #%v: %v", i, err)
		}

		// decode EventID
		id := EventID{}
		err = decoder.Decode(&id)
		if err != nil {
			return fmt.Errorf("unable to decode EventID for event #%v: %v", i, err)
		}

		log.Debug(fmt.Sprintf("event #%v has EventID %v", i, id))

		// ask metadata for method & event name for event
		moduleName, eventName, err := m.FindEventNamesForEventID(id)
		// moduleName, eventName, err := "System", "ExtrinsicSuccess", nil
		if err != nil {
			return fmt.Errorf("unable to find event with EventID %v in metadata for event #%v: %s", id, i, err)
		}

		log.Debug(fmt.Sprintf("event #%v is in module %v with event name %v", i, moduleName, eventName))

		// check whether name for eventID exists in t
		field := val.FieldByName(fmt.Sprintf("%v_%v", moduleName, eventName))
		if !field.IsValid() {
			return fmt.Errorf("unable to find field %v_%v for event #%v with EventID %v", moduleName, eventName, i, id)
		}

		// create a pointer to with the correct type that will hold the decoded event
		holder := reflect.New(field.Type().Elem())

		// ensure first field is for Phase, last field is for Topics
		numFields := holder.Elem().NumField()
		if numFields < 2 {
			return fmt.Errorf("expected event #%v with EventID %v, field %v_%v to have at least 2 fields "+
				"(for Phase and Topics), but has %v fields", i, id, moduleName, eventName, numFields)
		}
		phaseField := holder.Elem().FieldByIndex([]int{0})
		if phaseField.Type() != reflect.TypeOf(phase) {
			return fmt.Errorf("expected the first field of event #%v with EventID %v, field %v_%v to be of type "+
				"types.Phase, but got %v", i, id, moduleName, eventName, phaseField.Type())
		}
		topicsField := holder.Elem().FieldByIndex([]int{numFields - 1})
		if topicsField.Type() != reflect.TypeOf([]Hash{}) {
			return fmt.Errorf("expected the last field of event #%v with EventID %v, field %v_%v to be of type "+
				"[]types.Hash for Topics, but got %v", i, id, moduleName, eventName, topicsField.Type())
		}

		// set the phase we decoded earlier
		phaseField.Set(reflect.ValueOf(phase))

		// set the remaining fields
		for j := 1; j < numFields; j++ {
			err = decoder.Decode(holder.Elem().FieldByIndex([]int{j}).Addr().Interface())
			if err != nil {
				return fmt.Errorf("unable to decode field %v event #%v with EventID %v, field %v_%v: %v", j, i, id, moduleName,
					eventName, err)
			}
		}

		// add the decoded event to the slice
		field.Set(reflect.Append(field, holder.Elem()))

		log.Debug(fmt.Sprintf("decoded event #%v", i))
	}
	return nil
}

// Phase is an enum describing the current phase of the event (applying the extrinsic or finalized)
type Phase struct {
	IsApplyExtrinsic bool
	AsApplyExtrinsic uint32
	IsFinalization   bool
	IsInitialization bool
}

func (p *Phase) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		p.IsApplyExtrinsic = true
		err = decoder.Decode(&p.AsApplyExtrinsic)
	case 1:
		p.IsFinalization = true
	case 2:
		p.IsInitialization = true
	}

	if err != nil {
		return err
	}

	return nil
}

func (p Phase) Encode(encoder scale.Encoder) error {
	var err1, err2 error

	switch {
	case p.IsApplyExtrinsic:
		err1 = encoder.PushByte(0)
		err2 = encoder.Encode(p.AsApplyExtrinsic)
	case p.IsFinalization:
		err1 = encoder.PushByte(1)
	case p.IsInitialization:
		err1 = encoder.PushByte(2)
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

type EventID [2]byte
