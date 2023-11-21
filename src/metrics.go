package rtfblog

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	registry              *prometheus.Registry
	numRobotsServed       prometheus.Counter
	numForbiddenResponses prometheus.Counter
	numAdminRequests      prometheus.Counter
	numNonAdminRequests   prometheus.Counter
	numPanics             prometheus.Counter
	numInternalErrors     prometheus.Counter
	latenciesHist         prometheus.Histogram
}

func initMetrics() metrics {
	reg := prometheus.NewRegistry()
	factory := promauto.With(reg)
	numRobotsServed := factory.NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_robots_txt_served",
		Help:      "The total number of times robots.txt was served",
	})
	numForbiddenResponses := factory.NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_403s",
		Help:      "The total number of Forbidden responses",
	})
	numAdminRequests := factory.NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_admin_reqs",
		Help:      "The total number of requests to admin area",
	})
	numNonAdminRequests := factory.NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_non_admin_reqs",
		Help:      "The total number of requests to public pages",
	})
	numPanics := factory.NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_panics",
		Help:      "The total number of panics in the handler",
	})
	numInternalErrors := factory.NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_internal_errors",
		Help:      "The total number of internal errors in the handler",
	})
	latenciesHist := factory.NewHistogram(prometheus.HistogramOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "request_duration",
		Help:      "The time it took to serve each request",
		// 0.1ms, 2x on each bucket, 16 buckets
		Buckets: prometheus.ExponentialBuckets(1e-4, 2, 16),
	})
	return metrics{
		registry:              reg,
		numRobotsServed:       numRobotsServed,
		numForbiddenResponses: numForbiddenResponses,
		numAdminRequests:      numAdminRequests,
		numNonAdminRequests:   numNonAdminRequests,
		numPanics:             numPanics,
		numInternalErrors:     numInternalErrors,
		latenciesHist:         latenciesHist,
	}
}
