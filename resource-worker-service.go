package main

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"errors"
	"net/http"
	"io/ioutil"
	"container/ring"
	"github.com/gin-gonic/gin"
	// "github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

func (request *NetworkRequest) Run() error {
	url := fmt.Sprintf("http://%s:7998/network-endpoint", networkWorkerPeers.Value)
	networkWorkerPeers = networkWorkerPeers.Next()

	content := make([]byte, request.Bandwidth)
	resp, err := http.Post(url, "text/plain", bytes.NewReader(content))

	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Return code %d", resp.StatusCode))
	}
	return nil
}

type BlkIoRequest struct {
}

func (request *BlkIoRequest) Run() error {
	return nil
}

type ResourceRequest struct {
	CPURequest     *CPURequest     `form:"cpu" json:"cpu,omitempty"`
	NetworkRequest *NetworkRequest `form:"network" json:"network,omitempty"`
	BlkIoRequest   *BlkIoRequest   `from:"blkio" json:"blkio,omitempty"`
}

func (request *ResourceRequest) Run() error {
	if request.CPURequest != nil {
		return request.CPURequest.Run()
	} else if request.NetworkRequest != nil {
		return request.NetworkRequest.Run()
	} else if request.BlkIoRequest != nil {
		return request.BlkIoRequest.Run()
	}
	return nil
}

type Work struct {
	Requests []ResourceRequest `form:"definitions" json:"definitions"`
}

func getNetworkWorkerPeerSelector() *ring.Ring {
	currentPod := os.Getenv("HOSTNAME")

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		statefulSet, _ := clientset.AppsV1beta1().StatefulSets("default").Get(
			"resource-worker",
			metav1.GetOptions{},
		)
		if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
			break
		} else {
			time.Sleep(3)
		}
	}

	pods, err := clientset.CoreV1().Pods("default").List(
		metav1.ListOptions{LabelSelector: "app=resource-worker"},
	)

	peers := ring.New(len(pods.Items) - 1)

	for _, pod := range pods.Items {
		podName := pod.GetName()
		if podName != currentPod {
			peers.Value = fmt.Sprintf(
				"%s.%s.%s.svc.cluster.local",
				podName,
				pod.Spec.Subdomain,
				pod.Namespace,
			)
			peers = peers.Next()
		}
	}

	return peers
}

var networkWorkerPeers *ring.Ring = getNetworkWorkerPeerSelector()

func main() {

	prometheus.MustRegister(duration)
	prometheus.MustRegister(counter)

	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/network-endpoint", func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": err.Error(),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"length": len(body),
		})
	})

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
