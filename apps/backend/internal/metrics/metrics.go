// Package metrics registers all Prometheus metrics for the Backend.
//
// Usage: call metrics.Register() once at startup (from main), then call
// the individual Inc/Observe helpers from handlers.
//
// All metrics are in the "pz_" namespace to avoid collisions.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Join metrics
var (
	JoinTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pz_join_total",
		Help: "Total number of join requests, partitioned by server and result.",
	}, []string{"server_id", "result"}) // result: ok | not_found | offline | error

	JoinDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pz_join_duration_seconds",
		Help:    "Latency of successful join resolutions.",
		Buckets: prometheus.DefBuckets,
	}, []string{"server_id"})
)

// Blob metrics
var (
	BlobUploadTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pz_blob_upload_total",
		Help: "Total number of blobs successfully stored by agents.",
	})

	BlobUploadBytes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pz_blob_upload_bytes_total",
		Help: "Total bytes received via blob PUT.",
	})

	BlobDownloadTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pz_blob_download_total",
		Help: "Total number of blob download requests served.",
	})
)

// Manifest metrics
var (
	ManifestPublishTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pz_manifest_publish_total",
		Help: "Total manifest publishes, partitioned by server.",
	}, []string{"server_id"})

	ManifestVersionsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pz_manifest_versions_total",
		Help: "Current number of stored manifest versions per server.",
	}, []string{"server_id"})
)

// Heartbeat metrics
var (
	HeartbeatTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pz_heartbeat_total",
		Help: "Total heartbeats received, partitioned by server.",
	}, []string{"server_id"})
)

// Agent status gauges — updated on every GET /agents call and heartbeat.
var (
	AgentsOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pz_agents_online",
		Help: "Number of agents currently in 'online' state.",
	})

	AgentsDegraded = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pz_agents_degraded",
		Help: "Number of agents currently in 'degraded' state.",
	})

	AgentsOffline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pz_agents_offline",
		Help: "Number of agents currently in 'offline' state.",
	})
)

// UpdateAgentGauges recomputes the online/degraded/offline gauges from a
// fresh snapshot. Call this after every heartbeat and from GET /agents.
func UpdateAgentGauges(online, degraded, offline int) {
	AgentsOnline.Set(float64(online))
	AgentsDegraded.Set(float64(degraded))
	AgentsOffline.Set(float64(offline))
}
