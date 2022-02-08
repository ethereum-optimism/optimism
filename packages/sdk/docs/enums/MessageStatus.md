[@eth-optimism/sdk](../README.md) / [Exports](../modules.md) / MessageStatus

# Enumeration: MessageStatus

Enum describing the status of a message.

## Table of contents

### Enumeration members

- [FAILED\_L1\_TO\_L2\_MESSAGE](MessageStatus.md#failed_l1_to_l2_message)
- [IN\_CHALLENGE\_PERIOD](MessageStatus.md#in_challenge_period)
- [READY\_FOR\_RELAY](MessageStatus.md#ready_for_relay)
- [RELAYED](MessageStatus.md#relayed)
- [STATE\_ROOT\_NOT\_PUBLISHED](MessageStatus.md#state_root_not_published)
- [UNCONFIRMED\_L1\_TO\_L2\_MESSAGE](MessageStatus.md#unconfirmed_l1_to_l2_message)

## Enumeration members

### FAILED\_L1\_TO\_L2\_MESSAGE

• **FAILED\_L1\_TO\_L2\_MESSAGE** = `1`

Message is an L1 to L2 message and the transaction to execute the message failed.
When this status is returned, you will need to resend the L1 to L2 message, probably with a
higher gas limit.

#### Defined in

[packages/sdk/src/interfaces/types.ts:109](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L109)

___

### IN\_CHALLENGE\_PERIOD

• **IN\_CHALLENGE\_PERIOD** = `3`

Message is an L2 to L1 message and awaiting the challenge period.

#### Defined in

[packages/sdk/src/interfaces/types.ts:119](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L119)

___

### READY\_FOR\_RELAY

• **READY\_FOR\_RELAY** = `4`

Message is ready to be relayed.

#### Defined in

[packages/sdk/src/interfaces/types.ts:124](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L124)

___

### RELAYED

• **RELAYED** = `5`

Message has been relayed.

#### Defined in

[packages/sdk/src/interfaces/types.ts:129](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L129)

___

### STATE\_ROOT\_NOT\_PUBLISHED

• **STATE\_ROOT\_NOT\_PUBLISHED** = `2`

Message is an L2 to L1 message and no state root has been published yet.

#### Defined in

[packages/sdk/src/interfaces/types.ts:114](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L114)

___

### UNCONFIRMED\_L1\_TO\_L2\_MESSAGE

• **UNCONFIRMED\_L1\_TO\_L2\_MESSAGE** = `0`

Message is an L1 to L2 message and has not been processed by the L2.

#### Defined in

[packages/sdk/src/interfaces/types.ts:102](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/interfaces/types.ts#L102)
