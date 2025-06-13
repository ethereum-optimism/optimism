import {
  Resolved as ResolvedLegacyEvent
} from "../generated/templates/FaultDisputeGame/FaultDisputeGame"
import {
  Resolved as ResolvedNewEvent,
  PermissionedDisputeGame as PermissionedDisputeGameContract
} from "../generated/templates/PermissionedDisputeGame/PermissionedDisputeGame"
import { DisputeGameCreated } from "../generated/schema"
import { log } from '@graphprotocol/graph-ts'

export function handleResolvedLegacy(event: ResolvedLegacyEvent): void {
  // update dispute game
  let disputeGame = DisputeGameCreated.load(event.address)
  if (disputeGame == null) {
    log.warning("unexpected null dispute game for address {}", [event.address.toHexString()])
    return
  }
  disputeGame.resolvedStatus = event.params.status
  // l2BlockNumber is available directly in the legacy event
  disputeGame.l2BlockNumber = event.params.l2BlockNumber
  log.info("Marked dispute game resolved as {} with l2BlockNumber {} for address {}", [
    event.params.status.toString(),
    event.params.l2BlockNumber.toString(),
    event.address.toHexString()
  ])
  disputeGame.save()
}

export function handleResolvedNew(event: ResolvedNewEvent): void {
  // update dispute game
  let disputeGame = DisputeGameCreated.load(event.address)
  if (disputeGame == null) {
    log.warning("unexpected null dispute game for address {}", [event.address.toHexString()])
    return
  }
  disputeGame.resolvedStatus = event.params.status

  // For new events, we need to query the contract for l2BlockNumber
  let contract = PermissionedDisputeGameContract.bind(event.address)
  let l2BlockNumberResult = contract.try_l2BlockNumber()
  if (!l2BlockNumberResult.reverted) {
    disputeGame.l2BlockNumber = l2BlockNumberResult.value
    log.info("Marked dispute game resolved as {} with l2BlockNumber {} for address {}", [
      event.params.status.toString(),
      l2BlockNumberResult.value.toString(),
      event.address.toHexString()
    ])
  } else {
    log.warning("could not resolve l2BlockNumber for new resolved event at address {}", [event.address.toHexString()])
    log.info("Marked dispute game resolved as {} for address {}", [
      event.params.status.toString(),
      event.address.toHexString()
    ])
  }
  disputeGame.save()
}