import { Registry } from 'prom-client'
import { describe, expect, it } from 'vitest'

import { RelayerMetrics } from '@/metrics.js'

describe('RelayerMetrics', () => {
  it('registers every metric in the provided registry', async () => {
    const registry = new Registry()
    new RelayerMetrics(registry)

    const output = await registry.metrics()

    const expected = [
      'relayer_cycles_total',
      'relayer_cycle_duration_seconds',
      'relayer_messages_from_indexer',
      'relayer_ponder_request_duration_seconds',
      'relayer_ponder_errors_total',
      'relayer_ponder_last_success_timestamp',
      'relayer_eoa_balance_eth',
      'relayer_module_message_backlog',
      'relayer_module_message_skipped_total',
      'relayer_module_failure_registry_size',
      'relayer_module_failure_registry_oldest_age_seconds',
      'relayer_module_relay_attempt_failed_total',
      'relayer_module_relay_attempt_duration_seconds',
      'relayer_module_relay_attempt_retry_total',
      'relayer_module_relay_tx_broadcast_total',
      'relayer_module_relay_tx_executed_total',
      'relayer_module_relay_tx_last_executed_timestamp',
      'relayer_module_relay_tx_in_flight',
      'relayer_module_relay_tx_in_flight_age_seconds',
    ]
    for (const name of expected) {
      expect(output).toContain(`# TYPE ${name}`)
    }
  })

  it('accepts increments with the declared label set', async () => {
    const registry = new Registry()
    const metrics = new RelayerMetrics(registry)

    metrics.cyclesTotal.inc()
    metrics.messagesFromIndexer.set({ src: '420120000', dst: '420120001' }, 7)
    metrics.moduleRelayTxBroadcastTotal.inc({
      module: 'eth-bridge',
      src: '420120000',
      dst: '420120001',
      relayer_eoa: '0xabc',
    })
    metrics.moduleRelayAttemptFailedTotal.inc({
      module: 'eth-bridge',
      src: '420120000',
      dst: '420120001',
      relayer_eoa: '0xabc',
      stage: 'simulation',
      reason: 'already_relayed',
    })
    metrics.moduleRelayAttemptDurationSeconds.observe(
      {
        module: 'eth-bridge',
        src: '420120000',
        dst: '420120001',
        relayer_eoa: '0xabc',
        outcome: 'broadcast',
      },
      0.42,
    )

    const output = await registry.metrics()
    expect(output).toContain('relayer_cycles_total 1')
    expect(output).toContain(
      'relayer_messages_from_indexer{src="420120000",dst="420120001"} 7',
    )
    expect(output).toContain(
      'relayer_module_relay_tx_broadcast_total{module="eth-bridge",src="420120000",dst="420120001",relayer_eoa="0xabc"} 1',
    )
    expect(output).toMatch(
      /relayer_module_relay_attempt_failed_total\{.*stage="simulation".*reason="already_relayed".*\} 1/,
    )
  })

  it('isolates metrics to the given registry', async () => {
    const a = new Registry()
    const b = new Registry()
    const metricsA = new RelayerMetrics(a)
    new RelayerMetrics(b)

    metricsA.cyclesTotal.inc(5)

    expect(await a.metrics()).toContain('relayer_cycles_total 5')
    expect(await b.metrics()).toContain('relayer_cycles_total 0')
  })
})
