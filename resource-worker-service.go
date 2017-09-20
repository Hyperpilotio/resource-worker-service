package main

import (
	"time"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/gin-gonic/gin"
)

type CPURequestParams struct {
	Intensity float64 `form:"intensity" json:"intensity"`
	Cycles    int     `form:"cycles" json:"cycles"`
}

type MemoryRequestParams struct {
	Intensity float64 `form:"intensity" json:"intensity"`
	NumChunks int     `form:"numChunks" json:"numChunks"`
}

type ResourceRequest struct {
	*CPURequestParams    `form:"cpu" json:"cpu,omitempty"`
	*MemoryRequestParams `form:"memory" json:"memory,omitempty"`
}

type Request struct {
	Definitions []ResourceRequest `form:"definitions" json:"definitions"`
}

func main() {
	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/run", func(c *gin.Context) {
		var request Request
		if c.BindJSON(&request) == nil {
			startTime := time.Now()
			// Run the worker
			c.JSON(200, gin.H{
				"message": "done",
				"duration": time.Now().Sub(startTime).Seconds(),
				// Put the results / summaries of the work in response
			})
		}
	})

	r.Run(":7998")
}
