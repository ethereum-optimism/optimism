package disburser

import (
	"github.com/ethereum-optimism/optimism/go/bss-core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const methodLabel = "method"

var (
	// DBMethodUpsertDeposits is a label for UpsertDeposits db method.
	DBMethodUpsertDeposits = prometheus.Labels{methodLabel: "upsert_deposits"}

	// DBMethodConfirmedDeposits is a label for ConfirmedDeposits db method.
	DBMethodConfirmedDeposits = prometheus.Labels{methodLabel: "confirmed_deposits"}

	// DBMethodLastProcessedBlock is a label for LastProcessedBlock db method.
	DBMethodLastProcessedBlock = prometheus.Labels{methodLabel: "last_processed_block"}

	// DBMethodUpsertPendingTx is a label for UpsertPendingTx db method.
	DBMethodUpsertPendingTx = prometheus.Labels{methodLabel: "upsert_pending_tx"}

	// DBMethodListPendingTxs is a label for ListPendingTxs db method.
	DBMethodListPendingTxs = prometheus.Labels{methodLabel: "list_pending_txs"}

	// DBMethodUpsertDisbursement is a label for UpsertDisbursement db method.
	DBMethodUpsertDisbursement = prometheus.Labels{methodLabel: "upsert_disbursement"}

	// DBMethodLatestDisbursementID is a label for LatestDisbursementID db method.
	DBMethodLatestDisbursementID = prometheus.Labels{methodLabel: "latest_disbursement_id"}

	// DBMethodDeletePendingTx is a label for DeletePendingTx db method.
	DBMethodDeletePendingTx = prometheus.Labels{methodLabel: "delete_pending_tx"}
)

// Metrics extends the BSS core metrics with additional metrics tracked by the
// sequencer driver.
type Metrics struct {
	*metrics.Base

	// FailedDatabaseMethods tracks the number of database failures for each
	// known database method.
	FailedDatabaseMethods *prometheus.CounterVec

	// DepositIDMismatch tracks whether or not our database is in sync with the
	// disrburser contract. 1 means in sync, 0 means out of sync.
	DepositIDMismatch prometheus.Gauge

	// MissingDisbursements tracks the number of deposits that are missing
	// disbursement below our supposed next deposit id.
	MissingDisbursements prometheus.Gauge

	// SuccessfulDisbursements tracks the number of disbursements that emit a
	// success event from a given tx.
	SuccessfulDisbursements prometheus.Counter

	// FailedDisbursements tracks the number of disbursements that emit a failed
	// event from a given tx.
	FailedDisbursements prometheus.Counter

	// PostgresLastDisbursedID tracks the latest disbursement id in postgres.
	PostgresLastDisbursedID prometheus.Gauge

	// ContractNextDisbursementID tracks the next disbursement id expected by
	// the disburser contract.
	ContractNextDisbursementID prometheus.Gauge

	// DisburserBalance tracks Teleportr's disburser account balance.
	DisburserBalance prometheus.Gauge

	// DepositContractBalance tracks Teleportr's deposit contract balance.
	DepositContractBalance prometheus.Gauge
}

// NewMetrics initializes a new, extended metrics object.
func NewMetrics(subsystem string) *Metrics {
	base := metrics.NewBase(subsystem, "")
	return &Metrics{
		Base: base,
		FailedDatabaseMethods: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:      "failed_database_operations",
			Help:      "Tracks the number of database failures",
			Subsystem: base.SubsystemName(),
		}, []string{methodLabel}),
		DepositIDMismatch: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "deposit_id_mismatch",
			Help: "Set to 1 when the postgres and the disrburser contract " +
				"disagree on the next deposit id, and 0 otherwise",
			Subsystem: base.SubsystemName(),
		}),
		MissingDisbursements: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "missing_disbursements",
			Help: "Number of deposits that are missing disbursements in " +
				"postgres below our supposed next deposit id",
			Subsystem: base.SubsystemName(),
		}),
		SuccessfulDisbursements: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "successful_disbursements",
			Help: "Number of disbursements that emit a success event " +
				"from a given tx",
			Subsystem: base.SubsystemName(),
		}),
		FailedDisbursements: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "failed_disbursements",
			Help: "Number of disbursements that emit a failed event " +
				"from a given tx",
			Subsystem: base.SubsystemName(),
		}),
		PostgresLastDisbursedID: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "postgres_last_disbursed_id",
			Help:      "Latest recorded disbursement id in postgres",
			Subsystem: base.SubsystemName(),
		}),
		ContractNextDisbursementID: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "contract_next_disbursement_id",
			Help:      "Next disbursement id expected by the disburser contract",
			Subsystem: base.SubsystemName(),
		}),
		DisburserBalance: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "disburser_balance",
			Help:      "Balance in Wei of Teleportr's disburser wallet",
			Subsystem: base.SubsystemName(),
		}),
		DepositContractBalance: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "deposit_contract_balance",
			Help:      "Balance in Wei of Teleportr's deposit contract",
			Subsystem: base.SubsystemName(),
		}),
	}
}
