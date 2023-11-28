package evmclient

import "github.com/prometheus/client_golang/prometheus"

const (
	defaultMetricsNamespace = "mev_commit"
)

type metrics struct {
	AttemptedTxCount          prometheus.Counter
	SentTxCount               prometheus.Counter
	SuccessfulTxCount         prometheus.Counter
	CancelledTxCount          prometheus.Counter
	FailedTxCount             prometheus.Counter
	NotFoundDuringCancelCount prometheus.Counter
}

func newMetrics() *metrics {
	m := &metrics{
		AttemptedTxCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: defaultMetricsNamespace,
			Name:      "attempted_tx_count",
			Help:      "Number of attempted transactions",
		}),
		SentTxCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: defaultMetricsNamespace,
			Name:      "sent_tx_count",
			Help:      "Number of sent transactions",
		}),
		SuccessfulTxCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: defaultMetricsNamespace,
			Name:      "successful_tx_count",
			Help:      "Number of successful transactions",
		}),
		CancelledTxCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: defaultMetricsNamespace,
			Name:      "cancelled_tx_count",
			Help:      "Number of cancelled transactions",
		}),
		FailedTxCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: defaultMetricsNamespace,
			Name:      "failed_tx_count",
			Help:      "Number of failed transactions",
		}),
		NotFoundDuringCancelCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: defaultMetricsNamespace,
			Name:      "not_found_during_cancel_count",
			Help:      "Number of transactions not found during cancel",
		}),
	}

	return m
}

func (m *metrics) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.AttemptedTxCount,
		m.SentTxCount,
		m.SuccessfulTxCount,
		m.CancelledTxCount,
		m.FailedTxCount,
		m.NotFoundDuringCancelCount,
	}
}
