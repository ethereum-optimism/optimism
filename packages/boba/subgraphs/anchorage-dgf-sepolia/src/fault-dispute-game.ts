import { Resolved as ResolvedEvent } from "../generated/templates/FaultDisputeGame/FaultDisputeGame"
import { DisputeGameCreated } from "../generated/schema"
import { log } from '@graphprotocol/graph-ts'

export function handleResolved(event: ResolvedEvent): void {
  // update dispute game
  let disputeGame = DisputeGameCreated.load(event.address)
  if (disputeGame == null) {
    log.warning("unexpected null fault dispute game for address {}", [event.address.toHexString()])
    return
  }
  disputeGame.resolvedStatus = event.params.status
  log.info("Marked fault dispute game resolved as {} for address {}", [event.params.status.toString(), event.address.toHexString()])
  disputeGame.save()
}
