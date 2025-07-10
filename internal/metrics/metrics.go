package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds the Prometheus metrics for the application.
// It includes counters for commands received, messages sent,
// new users, and a histogram for database query durations.
type Metrics struct {
	CommandReceived  *prometheus.CounterVec   // Counter for received commands
	SentMessages     *prometheus.CounterVec   // Counter for sent messages
	NewUsers         prometheus.Counter       // Counter for new users
	DBQueryDuration  *prometheus.HistogramVec // Histogram for database query durations
	ReportGeneration *prometheus.HistogramVec // Histogram for report query durations
}

// NewMetrics creates a new Metrics instance with the provided Prometheus Registerer.
// It initializes counters, histograms, and gauges for tracking geocoding tasks,
// API errors, request durations, and active workers.
//
// Parameters:
//   - reg: A Prometheus Registerer used to register the metrics.
//
// Returns:
//   - A pointer to the newly created Metrics instance.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		CommandReceived: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "telegram_commands_received_total",
			Help: "Total number of used commands",
		}, []string{"command"}), // command: /start, login, logout
		SentMessages: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "telegram_messages_sent_total",
			Help: "Output bot activity",
		}, []string{"type"}), // type: text, reply, error, reactions
		NewUsers: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "telegram_new_users_total",
			Help: "Total number of new users via /start command",
		}),
		DBQueryDuration: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "telegram_db_query_duration_seconds",
			Help:    "Duration of database queries.",
			Buckets: prometheus.DefBuckets,
		}, []string{"query_type"}), // query_type: 'get_employee', 'upsert_task'
		ReportGeneration: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name: "telegram_report_generation_duration_seconds",
			Help: "Duration of report excel generation.",
		}, []string{"period"}), // period: last_7d, last_1m, current_1m
	}
}
