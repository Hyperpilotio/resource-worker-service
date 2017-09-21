package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/run", func(c *gin.Context) {
		var work Work
		if err := c.BindJSON(&request); err == nil {
			startTime := time.Now()
			for _, definition := range request.Definitions {
				if err := definition.Run(); err != nil {
					// Error
				}
			}

			// Run the worker
			c.JSON(http.StatusOk, gin.H{
				"message":  "done",
				"duration": time.Now().Sub(startTime),
				// Put the results / summaries of the work in response
			})
		} else {
			// Handle error
		}
	})

	r.Run(":7998")
}
