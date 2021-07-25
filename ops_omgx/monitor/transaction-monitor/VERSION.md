# DataBase

## OMGXV1

* Table `block`

  | Variable      | Description             |
  | ------------- | ----------------------- |
  | `Hash`        | Block hash              |
  | `parentHash`  | The last block hash     |
  | `blockNumber` | The number of block     |
  | `timestamp`   | Timestamp of this block |
  | `nonce`       | N/A                     |
  | `gasLimit`    | N/A                     |
  | `gasUsed`     | N/A                     |

* Table `transaction`

  | Variable      | Description              |
  | ------------- | ------------------------ |
  | `hash`        | Hash of this transaction |
  | `blockHash`   | N/A                      |
  | `blockNumber` | N/A                      |
  | `from`        | N/A                      |
  | `to`          | N/A                      |
  | `value`       | N/A                      |
  | `nonce`       | N/A                      |
  | `gasLimit`    | N/A                      |
  | `gasPrice`    | N/A                      |
  | `timestamp`   | N/A                      |

* Table `receipt`

  | Variable                                  | Description                                                  |
  | ----------------------------------------- | ------------------------------------------------------------ |
  | `hash`                                    | Hash of this transaction                                     |
  | `blockHash`                               | N/A                                                          |
  | `blockNumber`                             | N/A                                                          |
  | `from`                                    | N/A                                                          |
  | `to`                                      | N/A                                                          |
  | `gasUsed`                                 | N/A                                                          |
  | `cumulativeGasUsed`                       | N/A                                                          |
  | `crossDomainMessage`                      | Whether the transaction sends the cross domain message       |
  | `crossDomainMessageFinalize`              | Whether the cross domain message is finalized                |
  | `crossDomainMessageSendTime`              | When cross domain message is sent                            |
  | `crossDomainMessageEstimateFinalizedTime` | The estimate time when the cross domain message is finalized |
  | `timestamp`                               | N/A                                                          |

# Chain Scanner

## V1.0.0



# Message Scanner

## V1.0.0