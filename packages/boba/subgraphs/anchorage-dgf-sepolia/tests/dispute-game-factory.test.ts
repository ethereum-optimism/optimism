import {
  assert,
  describe,
  test,
  clearStore,
  beforeAll,
  afterAll
} from "matchstick-as"
import { Address, BigInt, Bytes } from "@graphprotocol/graph-ts"
import { DisputeGameCreated } from "../generated/schema"
import { DisputeGameCreated as DisputeGameCreatedEvent } from "../generated/DisputeGameFactory/DisputeGameFactory"
import { handleDisputeGameCreated } from "../src/dispute-game-factory"
import { createDisputeGameCreatedEvent } from "./dispute-game-factory-utils"

// Tests structure (matchstick-as >=0.5.0)
// https://thegraph.com/docs/en/developer/matchstick/#tests-structure-0-5-0

describe("Describe entity assertions", () => {
  beforeAll(() => {
    let disputeProxy = Address.fromString(
      "0x0000000000000000000000000000000000000001"
    )
    let gameType = BigInt.fromI32(234)
    let rootClaim = Bytes.fromI32(1234567890)
    let newDisputeGameCreatedEvent = createDisputeGameCreatedEvent(
      disputeProxy,
      gameType,
      rootClaim
    )
    handleDisputeGameCreated(newDisputeGameCreatedEvent)
  })

  afterAll(() => {
    clearStore()
  })

  // For more test scenarios, see:
  // https://thegraph.com/docs/en/developer/matchstick/#write-a-unit-test

  test("DisputeGameCreated created and stored", () => {
    assert.entityCount("DisputeGameCreated", 1)

    // 0xa16081f360e3847006db660bae1c6d1b2e17ec2a is the default address used in newMockEvent() function
    assert.fieldEquals(
      "DisputeGameCreated",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "disputeProxy",
      "0x0000000000000000000000000000000000000001"
    )
    assert.fieldEquals(
      "DisputeGameCreated",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "gameType",
      "234"
    )
    assert.fieldEquals(
      "DisputeGameCreated",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "rootClaim",
      "1234567890"
    )

    // More assert options:
    // https://thegraph.com/docs/en/developer/matchstick/#asserts
  })
})
