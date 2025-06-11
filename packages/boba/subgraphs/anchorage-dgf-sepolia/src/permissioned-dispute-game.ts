import { Resolved as ResolvedEvent } from "../generated/templates/PermissionedDisputeGame/PermissionedDisputeGame"
import { DisputeGameCreated } from "../generated/schema"

export function handleResolved(event: ResolvedEvent): void {
  // update dispute game
  let disputeGame = DisputeGameCreated.load(event.address)
  if (disputeGame == null) {
    return
  }
  disputeGame.resolvedStatus = event.params.status
  disputeGame.save()
}
