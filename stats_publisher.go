package main

import (
	"os"
	"fmt"
	"time"
	"errors"
	"github.com/quipo/statsd"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

type prometheusPublisher struct {
	counters  map[string]*prometheus.CounterVec
	summaries map[string]*prometheus.SummaryVec
}

type StatsPublisher struct {
	To         string
	NodeName   string
	prometheus *prometheusPublisher
	statsd     *statsd.StatsdClient
}

func NewStatsPublisher(to string) (*StatsPublisher, error) {
	nodeName := os.Getenv("NODE_NAME")

	publisher := &StatsPublisher{To: to, NodeName: nodeName}

	switch to {
	case "prometheus":
		publisher.prometheus = &prometheusPublisher{
			counters:  make(map[string]*prometheus.CounterVec),
			summaries: make(map[string]*prometheus.SummaryVec),
		}
	case "statsd":
		prefix := "hyperpilot.resource-worker-service."
		statsdHost := os.Getenv("STATSD_HOST")
		if statsdHost == "" {
			return nil, errors.New("STATSD_HOST environment variable must be defined")
		}
		statsdPort := os.Getenv("STATSD_PORT")
		if statsdPort == "" {
			glog.Warning("STATSD_PORT environment variable not defined, using default 8125")
			statsdPort = "8125"
		}
		statsdUrl := fmt.Sprintf("%s:%s", statsdHost, statsdPort)
		statsdClient := statsd.NewStatsdClient(statsdUrl, prefix)
		if err := statsdClient.CreateSocket(); err != nil {
			return nil, errors.New("Failed to create statsd client: " + err.Error())
		}
		publisher.statsd = statsdClient
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

func (publisher *StatsPublisher) makeStatsdMetric(metric string, label string) string {
	return fmt.Sprintf("%s.%s.%s", metric, publisher.NodeName, label)
}

func (publisher *StatsPublisher) Inc(metric string, label string) error {
	switch publisher.To {
	case "prometheus":
		publisher.prometheus.counters[metric].With(
			prometheus.Labels{"label": label, "node": publisher.NodeName},
		).Inc()
	case "statsd":
		return publisher.statsd.Incr(publisher.makeStatsdMetric(metric, label), 1)
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
		return publisher.statsd.Timing(publisher.makeStatsdMetric(metric, label), int64(duration.Seconds() * 1000))
	}
	return nil
}
