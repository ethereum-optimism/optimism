export interface OVMTransactionData {
  timestamp: number
  queueOrigin: number
  ovmEntrypoint: string
  callBytes: string
  fromAddress: string
  l1MsgSenderAddress: string
  allowRevert: boolean
}