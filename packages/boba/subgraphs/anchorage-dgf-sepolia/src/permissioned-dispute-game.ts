import { Resolved as ResolvedEvent, PermissionedDisputeGame } from "../generated/templates/PermissionedDisputeGame/PermissionedDisputeGame"
import { DisputeGameCreated } from "../generated/schema"

export function handleResolved(event: ResolvedEvent): void {
  // update dispute game
  let disputeGame = DisputeGameCreated.load(event.address)
  if (disputeGame == null) {
    return
  }
  disputeGame.resolvedStatus = event.params.status

  // Get l2BlockNumber from contract since it's not in the event
  let contract = PermissionedDisputeGame.bind(event.address)
  let l2BlockNumberResult = contract.try_l2BlockNumber()
  if (!l2BlockNumberResult.reverted) {
    disputeGame.l2BlockNumber = l2BlockNumberResult.value
  }

  disputeGame.save()
}
