package publisher

import (
	"os"
	"fmt"
	"time"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/DataDog/datadog-go/statsd"
)

type prometheusPublisher struct {
	counters  map[string]*prometheus.CounterVec
	summaries map[string]*prometheus.SummaryVec
}

type StatsPublisher struct {
	To         string
	NodeName   string
	prometheus *prometheusPublisher
	statsd     *statsd.Client
}

func NewStatsPublisher(to string) (*StatsPublisher, error) {
	nodeName := os.Getenv("NODE_NAME")

	publisher := &StatsPublisher{To: to, NodeName: nodeName}

	switch to {
	case "prometheus":
		publisher.prometheus = &prometheusPublisher{
			counters: make(map[string]*prometheus.CounterVec),
			summaries: make(map[string]*prometheus.SummaryVec),
		}
	case "statsd":
		statsdClient, err := statsd.New(nodeName + ":8125")
		if err != nil {
			return nil, errors.New("Failed to create statsd client: " + err.Error())
		}
		publisher.statsd = statsdClient
		publisher.statsd.Namespace = "hyperpilot.resource-worker-service."
		publisher.statsd.Tags = append(
			publisher.statsd.Tags,
			fmt.Sprintf("node:%s", nodeName),
		)
	default:
		return nil, errors.New("Unrecognized stats publisher service: " + to)
	}

	return publisher, nil
}

func (publisher *StatsPublisher) RegisterCounter(metric string, help string) {
	if publisher.To == "prometheus" {
		publisher.prometheus.counters[metric] = prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: metric, Help: help},
			[]string{"label", "node"},
		)
		prometheus.MustRegister(publisher.prometheus.counters[metric])
	}
}

func (publisher *StatsPublisher) RegisterTimer(metric string, help string) {
	if publisher.To == "prometheus" {
		publisher.prometheus.summaries[metric] = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{Name: metric, Help: help},
			[]string{"label", "node"},
		)
		prometheus.MustRegister(publisher.prometheus.summaries[metric])
	}
}

func (publisher *StatsPublisher) Inc(metric string, label string) error {
	switch publisher.To {
	case "prometheus":
		publisher.prometheus.counters[metric].With(
			prometheus.Labels{"label": label, "node": publisher.NodeName},
		).Inc()
	case "statsd":
		return publisher.statsd.Incr(metric, []string{fmt.Sprintf("label:%s", label)}, 1)
	}
	return nil
}

func (publisher *StatsPublisher) Timing(metric string, label string, duration time.Duration) error {
	switch publisher.To {
	case "prometheus":
		publisher.prometheus.summaries[metric].With(
			prometheus.Labels{"label": label, "node": publisher.NodeName},
		).Observe(duration.Seconds() * 1000)
	case "statsd":
		return publisher.statsd.Timing(metric, duration, []string{fmt.Sprintf("label:%s", label)}, 1)
	}
	return nil
}
