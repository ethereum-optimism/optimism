export interface OpNodeConfig {
  genesis: {
    l1: {
      hash: string
      number: number
    }
    l2: {
      hash: string
      number: number
    }
    l2_time: number
  }
  block_time: number
  max_sequencer_drift: number
  seq_window_size: number
  channel_timeout: number
  l1_chain_id: number
  l2_chain_id: number
  p2p_sequencer_address: string
  batch_inbox_address: string
  batch_sender_address: string
  deposit_contract_address: string
}
