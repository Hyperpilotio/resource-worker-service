package main

import (
	"time"
	"net/http"
	"github.com/gin-gonic/gin"
	// "github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	duration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "request_duration_seconds",
		Help: "Summary of request durations.",
	})
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count",
			Help: "Count of requests",
		},
		[]string{"resource"},
	)
)

type ResourceWork interface {
	Run() error
}

type CPURequest struct {
	Cycles int `form:"cycles" json:"cycles"`
}

func (request *CPURequest) Run() error {
	for i := 0; i < request.Cycles; i += 1 {
	}

	return nil
}

type NetworkRequest struct {
	Bandwidth int `form:"bandwidth" json:"bandwidth"`
}

type BlkIoRequest struct {
}

type ResourceRequest struct {
	*CPURequest     `form:"cpu" json:"cpu,omitempty"`
	*NetworkRequest `form:"network" json:"network,omitempty"`
	*BlkIoRequest   `from:"blkio" json:"blkio,omitempty"`
}

type Work struct {
	Requests []ResourceRequest `form:"definitions" json:"definitions"`
}

func main() {

	prometheus.MustRegister(duration)
	prometheus.MustRegister(counter)

	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/run", func(c *gin.Context) {

		defer func(begun time.Time) {
			duration.Observe(time.Since(begun).Seconds())
			counter.With(prometheus.Labels{
				"resource": "cpu",
			}).Inc()
		}(time.Now())

		startTime := time.Now()
		var work Work
		if err := c.BindJSON(&work); err == nil {
			for _, definition := range work.Requests {
				if err := definition.Run(); err != nil {
					// Error
				}
			}

			// Run the worker
			c.JSON(http.StatusOK, gin.H{
				"message":  "done",
				"duration": time.Since(startTime).Seconds(),
				// Put the results / summaries of the work in response
			})
		} else {
			// Handle error
		}
	})

	r.Run(":7998")
}
