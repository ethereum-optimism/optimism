import {
  assert,
  describe,
  test,
  clearStore,
  beforeAll,
  afterAll
} from "matchstick-as/assembly/index"
import { Bytes, Address, BigInt } from "@graphprotocol/graph-ts"
import { FailedRelayedMessage } from "../generated/schema"
import { FailedRelayedMessage as FailedRelayedMessageEvent } from "../generated/L1CrossDomainMessenger/L1CrossDomainMessenger"
import { handleFailedRelayedMessage } from "../src/l-1-cross-domain-messenger"
import { createFailedRelayedMessageEvent } from "./l-1-cross-domain-messenger-utils"

// Tests structure (matchstick-as >=0.5.0)
// https://thegraph.com/docs/en/developer/matchstick/#tests-structure-0-5-0

describe("Describe entity assertions", () => {
  beforeAll(() => {
    let msgHash = Bytes.fromI32(1234567890)
    let newFailedRelayedMessageEvent = createFailedRelayedMessageEvent(msgHash)
    handleFailedRelayedMessage(newFailedRelayedMessageEvent)
  })

  afterAll(() => {
    clearStore()
  })

  // For more test scenarios, see:
  // https://thegraph.com/docs/en/developer/matchstick/#write-a-unit-test

  test("FailedRelayedMessage created and stored", () => {
    assert.entityCount("FailedRelayedMessage", 1)

    // 0xa16081f360e3847006db660bae1c6d1b2e17ec2a is the default address used in newMockEvent() function
    assert.fieldEquals(
      "FailedRelayedMessage",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "msgHash",
      "1234567890"
    )

    // More assert options:
    // https://thegraph.com/docs/en/developer/matchstick/#asserts
  })
})
