package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "graph"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	BuildInfo         *prometheus.GaugeVec
	EventsEnabled     prometheus.Gauge
	HttpEnabled       prometheus.Gauge
	EventsProcessed   *prometheus.CounterVec
	InvalidEvents     prometheus.Counter
	UnsupportedEvents prometheus.Counter
}

const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// New initializes the available metrics.
func New(registerer prometheus.Registerer) *Metrics {
	m := &Metrics{
		BuildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "build_info",
			Help:      "Build information",
		}, []string{"version"}),
		EventsEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_enabled",
			Help:      "Whether this instance consumes events (1) or not (0)",
		}),
		HttpEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "http_enabled",
			Help:      "Whether this instance processes HTTP API calls (1) or not (0)",
		}),
		EventsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events",
			Help:      "Number of consumed events",
		}, []string{"event", "result"}),
		InvalidEvents: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_invalid",
			Help:      "Number of supported events with invalid data",
		}),
		UnsupportedEvents: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_unsupported",
			Help:      "Number of unsupported events that were consumed and ignored",
		}),
	}

	_ = prometheus.Register(m.BuildInfo)
	_ = prometheus.Register(m.EventsEnabled)
	_ = prometheus.Register(m.HttpEnabled)
	_ = prometheus.Register(m.EventsProcessed)
	_ = prometheus.Register(m.UnsupportedEvents)
	_ = prometheus.Register(m.InvalidEvents)
	// TODO: implement metrics
	return m
}
