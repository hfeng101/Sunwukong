package main

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	demoMetrics = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "test_demo_metrics_total",

		},
	)


	demoMetricsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{

		},
		)

	demoMetricsHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{

		},
		)

	demoMetricsSummary = promauto.NewSummary(
		prometheus.SummaryOpts{
			Name: "test_demo_metrics_summary",

		},
		)
)

func recordMetrics(){
	go func() {
		demoMetrics.Inc()
	}()
}

func main() {

	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe("127.0.0.1:45641",nil)

	return
}
