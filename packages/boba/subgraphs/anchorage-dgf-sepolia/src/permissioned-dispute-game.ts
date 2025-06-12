import { Resolved as ResolvedEvent, PermissionedDisputeGame } from "../generated/templates/PermissionedDisputeGame/PermissionedDisputeGame"
import { DisputeGameCreated } from "../generated/schema"
import { log } from '@graphprotocol/graph-ts'

export function handleResolved(event: ResolvedEvent): void {
  // update dispute game
  let disputeGame = DisputeGameCreated.load(event.address)
  if (disputeGame == null) {
    log.warning("unexpected null permissioned dispute game for address {}", [event.address.toHexString()])
    return
  }
  disputeGame.resolvedStatus = event.params.status
  log.debug("Marked permissioned dispute game resolved as {} for address {}", [event.params.status.toString(), event.address.toHexString()])
  disputeGame.save()
}
